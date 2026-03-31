package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/manawise/api/api/middleware"
	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
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
	out := make([]*domain.Card, 0, len(names))
	seen := map[string]bool{}
	for _, n := range names {
		if c, ok := r.byName[n]; ok && c != nil {
			if !seen[c.ID] {
				out = append(out, c)
				seen[c.ID] = true
			}
			continue
		}
		for _, c := range r.byName {
			if c != nil && strings.EqualFold(c.Name, n) {
				if !seen[c.ID] {
					out = append(out, c)
					seen[c.ID] = true
				}
				break
			}
		}
	}
	return out, nil
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

func runSynergiesRequest(t *testing.T, h *DeckHandler, path string) *httptest.ResponseRecorder {
	t.Helper()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-1")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/api/v1/decks/{id}/synergies", h.Synergies)

	req := httptest.NewRequest(http.MethodGet, path, nil)
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

func TestDeckAnalysisHandler_ContainsCurveAndMetaFields(t *testing.T) {
	bolt := &domain.Card{ID: "c1", Name: "Lightning Bolt", CMC: 1, TypeLine: "Instant", Legalities: map[string]string{"modern": "legal"}}
	mountain := &domain.Card{ID: "c2", Name: "Mountain", CMC: 0, TypeLine: "Basic Land - Mountain", Legalities: map[string]string{"modern": "legal"}}
	cardRepo := &legalityMockCardRepo{
		byID: map[string]*domain.Card{"c1": bolt, "c2": mountain},
		byName: map[string]*domain.Card{"Lightning Bolt": bolt, "Mountain": mountain},
	}

	deck := &domain.Deck{
		ID:     "d-a1",
		UserID: "u-1",
		Format: "modern",
		Cards: []domain.DeckCard{
			{CardID: "c1", CardName: "Lightning Bolt", Quantity: 4},
			{CardID: "c2", CardName: "Mountain", Quantity: 56},
		},
	}

	analyzeUC := usecase.NewAnalyzeDeckUseCase(&analyzeMockFetcher{}, cardRepo, 2)
	classifyUC := usecase.NewDeckClassifierUseCase(cardRepo)
	h := NewDeckHandler(&legalityMockDeckRepo{deck: deck}, nil, cardRepo, analyzeUC, classifyUC, nil)

	rr := runAnalysisRequest(t, h, "/api/v1/decks/d-a1/analysis")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["curve"]; !ok {
		t.Fatalf("expected curve field")
	}
	if _, ok := resp["meta_fit_score"]; !ok {
		t.Fatalf("expected meta_fit_score field")
	}
	if _, ok := resp["deviation_from_meta"]; !ok {
		t.Fatalf("expected deviation_from_meta field")
	}
}

func TestDeckSimulateHandler_ContainsProbabilityMetrics(t *testing.T) {
	deck := &domain.Deck{
		ID:     "d-s1",
		UserID: "u-1",
		Format: "modern",
		Cards: []domain.DeckCard{
			{CardName: "Lightning Bolt", Quantity: 4},
			{CardName: "Goblin Guide", Quantity: 4},
			{CardName: "Mountain", Quantity: 20},
			{CardName: "Lava Spike", Quantity: 4},
			{CardName: "Monastery Swiftspear", Quantity: 4},
			{CardName: "Skullcrack", Quantity: 4},
			{CardName: "Rift Bolt", Quantity: 4},
			{CardName: "Searing Blaze", Quantity: 4},
			{CardName: "Eidolon of the Great Revel", Quantity: 4},
			{CardName: "Searing Blood", Quantity: 4},
			{CardName: "Light Up the Stage", Quantity: 4},
		},
	}

	h := NewDeckHandler(
		&legalityMockDeckRepo{deck: deck},
		nil,
		nil,
		nil,
		nil,
		usecase.NewMulliganAssistantUseCase(nil),
	)

	rr := runSimulateRequest(t, h, "/api/v1/decks/d-s1/simulate")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, key := range []string{"keep_probability", "avg_lands_t1", "p_two_lands_t2", "p_one_drop", "curve_out_t4"} {
		if _, ok := resp[key]; !ok {
			t.Fatalf("expected %s field", key)
		}
	}

	reasoningRaw, ok := resp["reasoning"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected reasoning object")
	}
	if _, ok := reasoningRaw["verdict"]; !ok {
		t.Fatalf("expected reasoning.verdict field")
	}
	if _, ok := reasoningRaw["risk"]; !ok {
		t.Fatalf("expected reasoning.risk field")
	}
	if _, ok := reasoningRaw["signals"]; !ok {
		t.Fatalf("expected reasoning.signals field")
	}
}

func TestDeckSynergiesHandler_OK(t *testing.T) {
	oracle := &domain.Card{ID: "c-oracle", Name: "Thassa's Oracle", CMC: 2, TypeLine: "Creature - Merfolk Wizard", OracleText: "When this enters..."}
	consult := &domain.Card{ID: "c-consult", Name: "Demonic Consultation", CMC: 1, TypeLine: "Instant", OracleText: "Name a card. Exile top six cards..."}
	ponder := &domain.Card{ID: "c-ponder", Name: "Ponder", CMC: 1, TypeLine: "Sorcery", OracleText: "Look at the top three cards... Draw a card."}
	duress := &domain.Card{ID: "c-duress", Name: "Duress", CMC: 1, TypeLine: "Sorcery", OracleText: "Target opponent reveals their hand..."}
	island := &domain.Card{ID: "c-island", Name: "Island", CMC: 0, TypeLine: "Basic Land - Island"}
	swamp := &domain.Card{ID: "c-swamp", Name: "Swamp", CMC: 0, TypeLine: "Basic Land - Swamp"}

	cardRepo := &legalityMockCardRepo{
		byID: map[string]*domain.Card{
			"c-oracle":  oracle,
			"c-consult": consult,
			"c-ponder":  ponder,
			"c-duress":  duress,
			"c-island":  island,
			"c-swamp":   swamp,
		},
		byName: map[string]*domain.Card{
			"Thassa's Oracle":    oracle,
			"Demonic Consultation": consult,
			"Ponder":             ponder,
			"Duress":             duress,
			"Island":             island,
			"Swamp":              swamp,
		},
	}

	deck := &domain.Deck{
		ID:     "d-syn-1",
		UserID: "u-1",
		Format: "modern",
		Cards: []domain.DeckCard{
			{CardID: "c-oracle", CardName: "Thassa's Oracle", Quantity: 2},
			{CardID: "c-consult", CardName: "Demonic Consultation", Quantity: 2},
			{CardID: "c-ponder", CardName: "Ponder", Quantity: 4},
			{CardID: "c-duress", CardName: "Duress", Quantity: 4},
			{CardID: "c-island", CardName: "Island", Quantity: 24},
			{CardID: "c-swamp", CardName: "Swamp", Quantity: 24},
		},
	}

	h := NewDeckHandler(&legalityMockDeckRepo{deck: deck}, nil, cardRepo, nil, nil, nil)
	rr := runSynergiesRequest(t, h, "/api/v1/decks/d-syn-1/synergies")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["combos"]; !ok {
		t.Fatalf("expected combos field")
	}
	if _, ok := resp["synergy_score"]; !ok {
		t.Fatalf("expected synergy_score field")
	}
	if _, ok := resp["packages"]; !ok {
		t.Fatalf("expected packages field")
	}

	combos, _ := resp["combos"].([]interface{})
	if len(combos) == 0 {
		t.Fatalf("expected at least one detected combo")
	}
}

func TestDeckSynergiesHandler_UnavailableWhenCardRepoNil(t *testing.T) {
	h := NewDeckHandler(
		&legalityMockDeckRepo{deck: &domain.Deck{ID: "d-syn-2", UserID: "u-1"}},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	rr := runSynergiesRequest(t, h, "/api/v1/decks/d-syn-2/synergies")
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rr.Code, rr.Body.String())
	}
}
