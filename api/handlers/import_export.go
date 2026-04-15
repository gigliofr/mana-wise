package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/manawise/api/api/middleware"
	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
)

// DeckImportExportHandler handles import/export operations for decks.
type DeckImportExportHandler struct {
	deckRepo      domain.DeckRepository
	userRepo      domain.UserRepository
	cardRepo      domain.CardRepository
	resolveCardUC *usecase.ResolveCardByNameUseCase
}

// NewDeckImportExportHandler creates a new handler.
func NewDeckImportExportHandler(
	deckRepo domain.DeckRepository,
	userRepo domain.UserRepository,
	cardRepo domain.CardRepository,
	resolveCardUC *usecase.ResolveCardByNameUseCase,
) *DeckImportExportHandler {
	return &DeckImportExportHandler{
		deckRepo:      deckRepo,
		userRepo:      userRepo,
		cardRepo:      cardRepo,
		resolveCardUC: resolveCardUC,
	}
}

type deckImportRequest struct {
	Format   string `json:"format"`
	Data     string `json:"data"`
	DeckName string `json:"deck_name,omitempty"`
}

type deckImportResponse struct {
	DeckID      string   `json:"deck_id"`
	CardsParsed int      `json:"cards_parsed"`
	MainCount   int      `json:"main_count"`
	SideCount   int      `json:"side_count"`
	Warnings    []string `json:"warnings,omitempty"`
}

func jsonErrorResponse(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": msg, "code": code, "status": code})
}

// Import parses a decklist in various formats and creates a deck.
// POST /api/v1/decks/import
func (h *DeckImportExportHandler) Import(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		jsonErrorResponse(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req deckImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErrorResponse(w, "invalid request body", http.StatusBadRequest)
		return
	}

	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = "text"
	}

	parser := GetParserForFormat(format)
	entries, warnings, err := parser.Parse(req.Data)
	if err != nil {
		jsonErrorResponse(w, "failed to parse decklist: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(entries) == 0 {
		jsonErrorResponse(w, "no cards parsed from input", http.StatusBadRequest)
		return
	}

	deckCards := make([]domain.DeckCard, 0)
	mainCount := 0
	sideCount := 0

	for _, entry := range entries {
		card, err := h.resolveCardUC.Execute(r.Context(), entry.CardName)
		if err != nil || card == nil {
			warnings = append(warnings, fmt.Sprintf("Could not resolve card '%s'; skipping", entry.CardName))
			continue
		}

		deckCard := domain.DeckCard{
			CardID:      card.ID,
			CardName:    card.Name,
			Quantity:    entry.Quantity,
			IsSideboard: entry.IsSideboard,
			IsCommander: false,
		}
		deckCards = append(deckCards, deckCard)

		if entry.IsSideboard {
			sideCount += entry.Quantity
		} else {
			mainCount += entry.Quantity
		}
	}

	if len(deckCards) == 0 {
		jsonErrorResponse(w, "no cards could be resolved from decklist", http.StatusBadRequest)
		return
	}

	deckName := strings.TrimSpace(req.DeckName)
	if deckName == "" {
		deckName = fmt.Sprintf("Imported Deck (%s)", strings.ToUpper(format))
	}

	newDeck := &domain.Deck{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      deckName,
		Format:    "unknown",
		Cards:     deckCards,
		IsPublic:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.deckRepo.Create(r.Context(), newDeck); err != nil {
		jsonErrorResponse(w, "failed to save deck: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := deckImportResponse{
		DeckID:      newDeck.ID,
		CardsParsed: len(deckCards),
		MainCount:   mainCount,
		SideCount:   sideCount,
		Warnings:    warnings,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

type deckExportResponse struct {
	Format    string `json:"format"`
	DeckName  string `json:"deck_name"`
	Data      string `json:"data"`
	CardCount int    `json:"card_count"`
}

// Export generates a decklist in the requested format.
// GET /api/v1/decks/{id}/export?format=arena|mtgo|moxfield|text
func (h *DeckImportExportHandler) Export(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		jsonErrorResponse(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	deckID := chi.URLParam(r, "id")
	if deckID == "" {
		jsonErrorResponse(w, "missing deck id", http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "text"
	}
	format = strings.ToLower(strings.TrimSpace(format))

	deck, err := h.deckRepo.FindByID(r.Context(), deckID)
	if err != nil || deck == nil {
		jsonErrorResponse(w, "deck not found", http.StatusNotFound)
		return
	}

	if deck.UserID != userID {
		jsonErrorResponse(w, "forbidden", http.StatusForbidden)
		return
	}

	exporter := GetExporterForFormat(format)
	data := exporter.Export(deck.Cards, false)

	resp := deckExportResponse{
		Format:    format,
		DeckName:  deck.Name,
		Data:      data,
		CardCount: len(deck.Cards),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
