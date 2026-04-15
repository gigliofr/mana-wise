package handlers

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
)

// CardsHandler handles card-related endpoints.
type CardsHandler struct {
	cardRepo    domain.CardRepository
	resolveCard *usecase.ResolveCardByNameUseCase
}

type cardMetadataBatchRequest struct {
	Names []string `json:"names"`
}

type cardMetadataItem struct {
	CardID          string `json:"card_id,omitempty"`
	Name            string `json:"name"`
	Rarity          string `json:"rarity,omitempty"`
	SetCode         string `json:"set_code,omitempty"`
	CollectorNumber string `json:"collector_number,omitempty"`
}

type cardMetadataBatchResponse struct {
	Items   []cardMetadataItem `json:"items"`
	Missing []string           `json:"missing,omitempty"`
}

// NewCardsHandler creates a CardsHandler.
func NewCardsHandler(cardRepo domain.CardRepository, resolveCard *usecase.ResolveCardByNameUseCase) *CardsHandler {
	return &CardsHandler{cardRepo: cardRepo, resolveCard: resolveCard}
}

// SearchByName handles GET /cards/search?name=... with fuzzy fallback.
func (h *CardsHandler) SearchByName(w http.ResponseWriter, r *http.Request) {
	if h.resolveCard == nil {
		jsonError(w, "card resolver is not configured", http.StatusServiceUnavailable)
		return
	}

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		jsonError(w, "name query parameter is required", http.StatusBadRequest)
		return
	}

	card, err := h.resolveCard.Execute(r.Context(), name)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, card)
}

// MetadataBatch handles POST /cards/metadata/batch.
func (h *CardsHandler) MetadataBatch(w http.ResponseWriter, r *http.Request) {
	if h.cardRepo == nil {
		jsonError(w, "card repository unavailable", http.StatusServiceUnavailable)
		return
	}

	var req cardMetadataBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	cleaned := make([]string, 0, len(req.Names))
	seen := map[string]bool{}
	for _, raw := range req.Names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		cleaned = append(cleaned, name)
	}
	if len(cleaned) == 0 {
		jsonError(w, "names must contain at least one non-empty card name", http.StatusBadRequest)
		return
	}

	cards, err := h.cardRepo.FindByNames(r.Context(), cleaned)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	byName := map[string]*domain.Card{}
	for _, c := range cards {
		if c == nil {
			continue
		}
		byName[strings.ToLower(strings.TrimSpace(c.Name))] = c
	}

	items := make([]cardMetadataItem, 0, len(cleaned))
	missing := make([]string, 0)
	for _, name := range cleaned {
		key := strings.ToLower(name)
		card := byName[key]
		if card == nil {
			missing = append(missing, name)
			continue
		}
		items = append(items, cardMetadataItem{
			CardID:          card.ID,
			Name:            card.Name,
			Rarity:          strings.ToLower(strings.TrimSpace(card.Rarity)),
			SetCode:         strings.ToLower(strings.TrimSpace(card.SetCode)),
			CollectorNumber: strings.TrimSpace(card.CollectorNumber),
		})
	}

	jsonOK(w, cardMetadataBatchResponse{Items: items, Missing: missing})
}

// GetCard handles GET /cards/{id}.
func (h *CardsHandler) GetCard(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	card, err := h.cardRepo.FindByID(r.Context(), id)
	if err != nil {
		jsonError(w, fmt.Sprintf("DB error: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	if card == nil {
		jsonError(w, "card not found", http.StatusNotFound)
		return
	}
	jsonOK(w, card)
}

// PriceTrend handles GET /cards/{id}/price-trend.
func (h *CardsHandler) PriceTrend(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	card, err := h.cardRepo.FindByID(r.Context(), id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if card == nil {
		jsonError(w, "card not found", http.StatusNotFound)
		return
	}
	jsonOK(w, computePriceTrend(card))
}

// PriceTrendByName handles GET /cards/by-name/price-trend?name=...
func (h *CardsHandler) PriceTrendByName(w http.ResponseWriter, r *http.Request) {
	card, ok := h.resolveFromQueryName(w, r)
	if !ok {
		return
	}
	jsonOK(w, computePriceTrend(card))
}

// PriceTrendResponse summarises price movement over multiple windows.
type PriceTrendResponse struct {
	CardID     string                 `json:"card_id"`
	CardName   string                 `json:"card_name"`
	Current    float64                `json:"current_usd"`
	Change7d   *float64               `json:"change_7d_pct,omitempty"`
	Change30d  *float64               `json:"change_30d_pct,omitempty"`
	Change90d  *float64               `json:"change_90d_pct,omitempty"`
	SpikeAlert bool                   `json:"spike_alert"`
	History    []domain.PriceSnapshot `json:"history"`
}

func computePriceTrend(card *domain.Card) PriceTrendResponse {
	resp := PriceTrendResponse{
		CardID:   card.ID,
		CardName: card.Name,
		History:  card.PriceHistory,
	}

	latest := card.LatestPrice()
	if latest == nil {
		return resp
	}
	resp.Current = latest.USD

	findPrice := func(daysBack int) *domain.PriceSnapshot {
		target := latest.Date.AddDate(0, 0, -daysBack)
		var closest *domain.PriceSnapshot
		for i := range card.PriceHistory {
			p := &card.PriceHistory[i]
			if p.Date.After(target) {
				continue
			}
			if closest == nil || p.Date.After(closest.Date) {
				closest = p
			}
		}
		return closest
	}

	changePct := func(old, current float64) *float64 {
		if old == 0 {
			return nil
		}
		v := math.Round(((current-old)/old*100)*100) / 100
		return &v
	}

	if p := findPrice(7); p != nil {
		resp.Change7d = changePct(p.USD, latest.USD)
		if resp.Change7d != nil && *resp.Change7d > 20 {
			resp.SpikeAlert = true
		}
	}
	if p := findPrice(30); p != nil {
		resp.Change30d = changePct(p.USD, latest.USD)
	}
	if p := findPrice(90); p != nil {
		resp.Change90d = changePct(p.USD, latest.USD)
	}

	return resp
}

// GetSynergies handles GET /cards/{id}/synergies?n=10.
func (h *CardsHandler) GetSynergies(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	target, ok := h.loadCardByID(w, r, id)
	if !ok {
		return
	}
	h.synergiesForCard(w, r, target)
}

// SynergiesByName handles GET /cards/by-name/synergies?name=...&n=10.
func (h *CardsHandler) SynergiesByName(w http.ResponseWriter, r *http.Request) {
	target, ok := h.resolveFromQueryName(w, r)
	if !ok {
		return
	}
	h.synergiesForCard(w, r, target)
}

func (h *CardsHandler) synergiesForCard(w http.ResponseWriter, r *http.Request, target *domain.Card) {
	n := 10
	minScore := 0.0
	if raw := r.URL.Query().Get("n"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			n = v
		}
	}
	if raw := r.URL.Query().Get("min_score"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			minScore = v
		}
	}
	if n > 100 {
		n = 100
	}

	if len(target.EmbeddingVector) == 0 {
		jsonError(w, "card has no embedding — run /embed/batch first", http.StatusUnprocessableEntity)
		return
	}

	candidates, err := h.cardRepo.FindWithEmbeddings(r.Context(), 2000)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	top := newTopK(n)
	for _, c := range candidates {
		if c.ID == target.ID || len(c.EmbeddingVector) == 0 {
			continue
		}
		score := cosineSimilarity(target.EmbeddingVector, c.EmbeddingVector)
		if score < minScore {
			continue
		}
		top.Push(ranked{Card: c, Score: score})
	}

	jsonOK(w, top.SortedDesc())
}

func (h *CardsHandler) loadCardByID(w http.ResponseWriter, r *http.Request, id string) (*domain.Card, bool) {
	card, err := h.cardRepo.FindByID(r.Context(), id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	if card == nil {
		jsonError(w, "card not found", http.StatusNotFound)
		return nil, false
	}
	return card, true
}

func (h *CardsHandler) resolveFromQueryName(w http.ResponseWriter, r *http.Request) (*domain.Card, bool) {
	if h.resolveCard == nil {
		jsonError(w, "card resolver is not configured", http.StatusServiceUnavailable)
		return nil, false
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		jsonError(w, "name query parameter is required", http.StatusBadRequest)
		return nil, false
	}
	card, err := h.resolveCard.Execute(r.Context(), name)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return nil, false
	}
	return card, true
}

// cosineSimilarity computes cosine similarity between two equal-length vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

type ranked struct {
	Card  *domain.Card `json:"card"`
	Score float64      `json:"score"`
}

type rankedMinHeap []ranked

func (h rankedMinHeap) Len() int            { return len(h) }
func (h rankedMinHeap) Less(i, j int) bool  { return h[i].Score < h[j].Score }
func (h rankedMinHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *rankedMinHeap) Push(x interface{}) { *h = append(*h, x.(ranked)) }
func (h *rankedMinHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

type topK struct {
	limit int
	h     rankedMinHeap
}

func newTopK(limit int) *topK {
	if limit <= 0 {
		limit = 10
	}
	t := &topK{limit: limit, h: rankedMinHeap{}}
	heap.Init(&t.h)
	return t
}

func (t *topK) Push(v ranked) {
	if t.h.Len() < t.limit {
		heap.Push(&t.h, v)
		return
	}
	if t.h[0].Score >= v.Score {
		return
	}
	heap.Pop(&t.h)
	heap.Push(&t.h, v)
}

func (t *topK) SortedDesc() []ranked {
	out := make([]ranked, t.h.Len())
	for i := len(out) - 1; i >= 0; i-- {
		out[i] = heap.Pop(&t.h).(ranked)
	}
	return out
}

// helpers

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  msg,
		"code":   statusCodeSlug(code),
		"status": code,
	})
}

func statusCodeSlug(code int) string {
	status := strings.ToLower(strings.TrimSpace(http.StatusText(code)))
	if status == "" {
		return "unknown_error"
	}
	status = strings.ReplaceAll(status, "-", " ")
	status = strings.ReplaceAll(status, "  ", " ")
	status = strings.ReplaceAll(status, " ", "_")
	return status
}
