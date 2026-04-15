package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gigliofr/mana-wise/api/middleware"
)

type collectionGapMissingLine struct {
	Card         string  `json:"card"`
	Qty          int     `json:"qty"`
	PriceUSD     float64 `json:"price_usd"`
	LineTotalUSD float64 `json:"line_total_usd"`
}

type collectionGapResponse struct {
	DeckID             string                     `json:"deck_id"`
	CompletionPct      int                        `json:"completion_pct"`
	Missing            []collectionGapMissingLine `json:"missing"`
	TotalToAcquireUSD  float64                    `json:"total_to_acquire_usd"`
	CheckedAtUTC       string                     `json:"checked_at"`
	InventorySource    string                     `json:"inventory_source"`
	Warnings           []string                   `json:"warnings,omitempty"`
}

// CollectionGaps handles GET /api/v1/users/me/collection/gaps/{deck_id}.
// V1 inventory source: query param `owned` (example: owned=Lightning Bolt:2,Fury:1).
func (h *DeckHandler) CollectionGaps(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.cardRepo == nil {
		jsonError(w, "card repository unavailable", http.StatusServiceUnavailable)
		return
	}

	deckID := strings.TrimSpace(chi.URLParam(r, "deck_id"))
	if deckID == "" {
		jsonError(w, "missing deck id", http.StatusBadRequest)
		return
	}

	deck, err := h.repo.FindByID(r.Context(), deckID)
	if err != nil {
		jsonError(w, "failed to retrieve deck", http.StatusInternalServerError)
		return
	}
	if deck == nil {
		jsonError(w, "deck not found", http.StatusNotFound)
		return
	}
	if deck.UserID != userID && !deck.IsPublic {
		jsonError(w, "deck not found", http.StatusNotFound)
		return
	}

	ownedMap := parseOwnedInventoryFromQuery(r)
	warnings := []string{}
	type reqLine struct {
		name string
		qty  int
		usd  float64
	}
	required := map[string]reqLine{}

	for _, entry := range deck.Cards {
		if strings.TrimSpace(entry.CardName) == "" || entry.Quantity <= 0 {
			continue
		}
		card, resolveErr := h.resolveDeckCardEntry(r, entry)
		if resolveErr != nil {
			warnings = append(warnings, "Could not resolve card for gap analysis: "+strings.TrimSpace(entry.CardName))
			continue
		}

		name := strings.TrimSpace(entry.CardName)
		if card != nil && strings.TrimSpace(card.Name) != "" {
			name = card.Name
		}
		usd, _, _ := extractCardUnitPrices(card)
		key := strings.ToLower(name)

		existing := required[key]
		existing.name = name
		existing.qty += entry.Quantity
		if usd > 0 {
			existing.usd = usd
		}
		required[key] = existing
	}

	totalRequired := 0
	totalMissing := 0
	totalToAcquire := 0.0
	missing := make([]collectionGapMissingLine, 0)

	for key, req := range required {
		totalRequired += req.qty
		owned := ownedMap[key]
		missingQty := req.qty - owned
		if missingQty <= 0 {
			continue
		}
		totalMissing += missingQty
		lineTotal := round4(float64(missingQty) * req.usd)
		totalToAcquire += lineTotal
		missing = append(missing, collectionGapMissingLine{
			Card:         req.name,
			Qty:          missingQty,
			PriceUSD:     round4(req.usd),
			LineTotalUSD: lineTotal,
		})
	}

	completion := 100
	if totalRequired > 0 {
		ownedTotal := totalRequired - totalMissing
		completion = int(float64(ownedTotal) * 100.0 / float64(totalRequired))
	}

	inventorySource := "query_owned_v1"
	if len(ownedMap) == 0 {
		inventorySource = "empty_inventory_v1"
	}

	jsonOK(w, collectionGapResponse{
		DeckID:            deck.ID,
		CompletionPct:     completion,
		Missing:           missing,
		TotalToAcquireUSD: round4(totalToAcquire),
		CheckedAtUTC:      time.Now().UTC().Format(time.RFC3339),
		InventorySource:   inventorySource,
		Warnings:          warnings,
	})
}

func parseOwnedInventoryFromQuery(r *http.Request) map[string]int {
	out := map[string]int{}
	values := r.URL.Query()["owned"]
	for _, raw := range values {
		for _, token := range strings.Split(raw, ",") {
			pair := strings.SplitN(strings.TrimSpace(token), ":", 2)
			if len(pair) != 2 {
				continue
			}
			name := strings.ToLower(strings.TrimSpace(pair[0]))
			if name == "" {
				continue
			}
			qty, err := strconv.Atoi(strings.TrimSpace(pair[1]))
			if err != nil || qty <= 0 {
				continue
			}
			out[name] += qty
		}
	}
	return out
}
