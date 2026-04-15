package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gigliofr/mana-wise/api/middleware"
	"github.com/gigliofr/mana-wise/domain"
)

func runCollectionGapRequest(t *testing.T, h *DeckHandler, path string) *httptest.ResponseRecorder {
	t.Helper()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-1")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/api/v1/users/me/collection/gaps/{deck_id}", h.CollectionGaps)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestCollectionGaps_OK(t *testing.T) {
	now := time.Now().UTC()
	deck := &domain.Deck{
		ID:     "d-gap",
		UserID: "u-1",
		Cards: []domain.DeckCard{
			{CardID: "c1", CardName: "Lightning Bolt", Quantity: 4},
			{CardID: "c2", CardName: "Solitude", Quantity: 2},
		},
	}
	cardRepo := &legalityMockCardRepo{
		byID: map[string]*domain.Card{
			"c1": {ID: "c1", Name: "Lightning Bolt", CurrentPrices: map[string]float64{"usd": 1.0}, UpdatedAt: now},
			"c2": {ID: "c2", Name: "Solitude", CurrentPrices: map[string]float64{"usd": 20.0}, UpdatedAt: now},
		},
		byName: map[string]*domain.Card{},
	}
	h := NewDeckHandler(&legalityMockDeckRepo{deck: deck}, nil, cardRepo, nil, nil, nil)

	rr := runCollectionGapRequest(t, h, "/api/v1/users/me/collection/gaps/d-gap?owned=Lightning%20Bolt:2,Solitude:1")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp collectionGapResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.DeckID != "d-gap" {
		t.Fatalf("expected deck_id d-gap, got %s", resp.DeckID)
	}
	if resp.CompletionPct <= 0 || resp.CompletionPct >= 100 {
		t.Fatalf("expected partial completion, got %d", resp.CompletionPct)
	}
	if len(resp.Missing) == 0 {
		t.Fatalf("expected missing cards")
	}
	if resp.TotalToAcquireUSD <= 0 {
		t.Fatalf("expected positive total_to_acquire_usd, got %f", resp.TotalToAcquireUSD)
	}
}

func TestCollectionGaps_NoOwned_AllMissing(t *testing.T) {
	now := time.Now().UTC()
	deck := &domain.Deck{
		ID:     "d-gap-2",
		UserID: "u-1",
		Cards: []domain.DeckCard{{CardID: "c1", CardName: "Solitude", Quantity: 2}},
	}
	cardRepo := &legalityMockCardRepo{
		byID: map[string]*domain.Card{
			"c1": {ID: "c1", Name: "Solitude", CurrentPrices: map[string]float64{"usd": 20.0}, UpdatedAt: now},
		},
		byName: map[string]*domain.Card{},
	}
	h := NewDeckHandler(&legalityMockDeckRepo{deck: deck}, nil, cardRepo, nil, nil, nil)

	rr := runCollectionGapRequest(t, h, "/api/v1/users/me/collection/gaps/d-gap-2")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp collectionGapResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.CompletionPct != 0 {
		t.Fatalf("expected 0 completion with empty inventory, got %d", resp.CompletionPct)
	}
	if len(resp.Missing) != 1 || resp.Missing[0].Qty != 2 {
		t.Fatalf("expected full missing qty, got %+v", resp.Missing)
	}
}

func TestCollectionGaps_Unauthorized(t *testing.T) {
	r := chi.NewRouter()
	h := NewDeckHandler(&legalityMockDeckRepo{}, nil, &legalityMockCardRepo{}, nil, nil, nil)
	r.Get("/api/v1/users/me/collection/gaps/{deck_id}", h.CollectionGaps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/collection/gaps/d-1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
