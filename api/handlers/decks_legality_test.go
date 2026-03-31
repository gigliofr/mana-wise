package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/manawise/api/api/middleware"
	"github.com/manawise/api/domain"
)

type legalityMockDeckRepo struct {
	deck *domain.Deck
}

func (r *legalityMockDeckRepo) FindByID(ctx context.Context, id string) (*domain.Deck, error) {
	if r.deck != nil && r.deck.ID == id {
		return r.deck, nil
	}
	return nil, nil
}

func (r *legalityMockDeckRepo) FindByUserID(ctx context.Context, userID string) ([]*domain.Deck, error) {
	return nil, nil
}

func (r *legalityMockDeckRepo) Create(ctx context.Context, deck *domain.Deck) error {
	return nil
}

func (r *legalityMockDeckRepo) Update(ctx context.Context, deck *domain.Deck) error {
	return nil
}

func (r *legalityMockDeckRepo) Delete(ctx context.Context, id string) error {
	return nil
}

type legalityMockCardRepo struct {
	byID   map[string]*domain.Card
	byName map[string]*domain.Card
}

func (r *legalityMockCardRepo) FindByID(ctx context.Context, id string) (*domain.Card, error) {
	return r.byID[id], nil
}

func (r *legalityMockCardRepo) FindByScryfallID(ctx context.Context, scryfallID string) (*domain.Card, error) {
	return nil, nil
}

func (r *legalityMockCardRepo) FindByName(ctx context.Context, name string) (*domain.Card, error) {
	return r.byName[name], nil
}

func (r *legalityMockCardRepo) FindByNames(ctx context.Context, names []string) ([]*domain.Card, error) {
	return nil, nil
}

func (r *legalityMockCardRepo) FindForEmbedding(ctx context.Context, limit int, onlyMissing bool) ([]*domain.Card, error) {
	return nil, nil
}

func (r *legalityMockCardRepo) Upsert(ctx context.Context, card *domain.Card) error {
	return nil
}

func (r *legalityMockCardRepo) UpsertMany(ctx context.Context, cards []*domain.Card) error {
	return nil
}

func (r *legalityMockCardRepo) UpdateEmbedding(ctx context.Context, id string, vector []float64) error {
	return nil
}

func (r *legalityMockCardRepo) FindWithEmbeddings(ctx context.Context, limit int) ([]*domain.Card, error) {
	return nil, nil
}

func (r *legalityMockCardRepo) CountAll(ctx context.Context) (int64, error) {
	return 0, nil
}

type legalityResponse struct {
	DeckID  string                            `json:"deck_id"`
	Formats map[string]map[string]interface{} `json:"formats"`
	Checked string                            `json:"checked_at"`
}

func runLegalityRequest(t *testing.T, h *DeckHandler, path string) *httptest.ResponseRecorder {
	t.Helper()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-1")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/api/v1/decks/{id}/legality", h.Legality)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func runAnalysisRequest(t *testing.T, h *DeckHandler, path string) *httptest.ResponseRecorder {
	t.Helper()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-1")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/api/v1/decks/{id}/analysis", h.Analysis)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func runSimulateRequest(t *testing.T, h *DeckHandler, path string) *httptest.ResponseRecorder {
	t.Helper()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-1")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Post("/api/v1/decks/{id}/simulate", h.Simulate)

	req := httptest.NewRequest(http.MethodPost, path, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestDeckLegalityHandler_OK(t *testing.T) {
	h := NewDeckHandler(
		&legalityMockDeckRepo{deck: &domain.Deck{
			ID:     "d-1",
			UserID: "u-1",
			Cards: []domain.DeckCard{
				{CardID: "c1", CardName: "Lightning Bolt", Quantity: 4, IsSideboard: false},
				{CardID: "c2", CardName: "Mountain", Quantity: 56, IsSideboard: false},
			},
		}},
		nil,
		&legalityMockCardRepo{
			byID: map[string]*domain.Card{
				"c1": {ID: "c1", Name: "Lightning Bolt", TypeLine: "Instant", Legalities: map[string]string{"modern": "legal", "standard": "not_legal", "legacy": "legal", "vintage": "legal", "pioneer": "not_legal", "pauper": "legal", "commander": "legal"}},
				"c2": {ID: "c2", Name: "Mountain", TypeLine: "Basic Land - Mountain", Legalities: map[string]string{"modern": "legal", "standard": "legal", "legacy": "legal", "vintage": "legal", "pioneer": "legal", "pauper": "legal", "commander": "legal"}},
			},
		},
		nil,
		nil,
		nil,
	)

	rr := runLegalityRequest(t, h, "/api/v1/decks/d-1/legality")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["deck_id"] != "d-1" {
		t.Fatalf("expected deck_id d-1, got %v", resp["deck_id"])
	}
	if _, ok := resp["formats"]; !ok {
		t.Fatalf("expected formats in response")
	}
	checked, _ := resp["checked_at"].(string)
	if _, err := time.Parse(time.RFC3339, checked); err != nil {
		t.Fatalf("checked_at not RFC3339: %q", checked)
	}
}

func TestDeckLegalityHandler_CardNotFound(t *testing.T) {
	h := NewDeckHandler(
		&legalityMockDeckRepo{deck: &domain.Deck{
			ID:     "d-2",
			UserID: "u-1",
			Cards: []domain.DeckCard{
				{CardID: "missing", CardName: "Unknown Card", Quantity: 4, IsSideboard: false},
			},
		}},
		nil,
		&legalityMockCardRepo{byID: map[string]*domain.Card{}},
		nil,
		nil,
		nil,
	)

	rr := runLegalityRequest(t, h, "/api/v1/decks/d-2/legality")

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestDeckAnalysisHandler_UnavailableWhenAnalyzeNil(t *testing.T) {
	h := NewDeckHandler(
		&legalityMockDeckRepo{deck: &domain.Deck{ID: "d-3", UserID: "u-1"}},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	rr := runAnalysisRequest(t, h, "/api/v1/decks/d-3/analysis")
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestDeckSimulateHandler_UnavailableWhenMulliganNil(t *testing.T) {
	h := NewDeckHandler(
		&legalityMockDeckRepo{deck: &domain.Deck{ID: "d-4", UserID: "u-1"}},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	rr := runSimulateRequest(t, h, "/api/v1/decks/d-4/simulate")
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rr.Code, rr.Body.String())
	}
}
