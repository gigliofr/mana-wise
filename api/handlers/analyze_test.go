package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manawise/api/api/middleware"
	"github.com/manawise/api/domain"
	"github.com/manawise/api/infrastructure/scryfall"
	"github.com/manawise/api/usecase"
)

type analyzeMockCardRepo struct {
	cardsByName map[string]*domain.Card
}

func (r *analyzeMockCardRepo) FindByID(ctx context.Context, id string) (*domain.Card, error) {
	return nil, nil
}

func (r *analyzeMockCardRepo) FindByScryfallID(ctx context.Context, scryfallID string) (*domain.Card, error) {
	return nil, nil
}

func (r *analyzeMockCardRepo) FindByName(ctx context.Context, name string) (*domain.Card, error) {
	return r.cardsByName[name], nil
}

func (r *analyzeMockCardRepo) FindByNames(ctx context.Context, names []string) ([]*domain.Card, error) {
	out := make([]*domain.Card, 0, len(names))
	for _, n := range names {
		if c, ok := r.cardsByName[n]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *analyzeMockCardRepo) FindForEmbedding(ctx context.Context, limit int, onlyMissing bool) ([]*domain.Card, error) {
	return nil, nil
}

func (r *analyzeMockCardRepo) Upsert(ctx context.Context, card *domain.Card) error {
	return nil
}

func (r *analyzeMockCardRepo) UpsertMany(ctx context.Context, cards []*domain.Card) error {
	return nil
}

func (r *analyzeMockCardRepo) UpdateEmbedding(ctx context.Context, id string, vector []float64) error {
	return nil
}

func (r *analyzeMockCardRepo) FindWithEmbeddings(ctx context.Context, limit int) ([]*domain.Card, error) {
	return nil, nil
}

func (r *analyzeMockCardRepo) CountAll(ctx context.Context) (int64, error) {
	return 0, nil
}

type analyzeMockFetcher struct{}

func (f *analyzeMockFetcher) GetCardByName(ctx context.Context, name string) (*scryfall.ScryfallCard, error) {
	return nil, context.Canceled
}

func (f *analyzeMockFetcher) GetCardByFuzzyName(ctx context.Context, name string) (*scryfall.ScryfallCard, error) {
	return nil, context.Canceled
}

type analyzeMockUserRepo struct {
	checkAndIncrementCalls int
	lastUserID             string
	lastDay                string
	allowIncrement         bool // controls what CheckAndIncrementDailyAnalyses returns
}

func (r *analyzeMockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, nil
}

func (r *analyzeMockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}

func (r *analyzeMockUserRepo) Create(ctx context.Context, user *domain.User) error {
	return nil
}

func (r *analyzeMockUserRepo) Update(ctx context.Context, user *domain.User) error {
	return nil
}

func (r *analyzeMockUserRepo) CheckAndIncrementDailyAnalyses(ctx context.Context, userID, today string, limit int) (bool, error) {
	r.checkAndIncrementCalls++
	r.lastUserID = userID
	r.lastDay = today
	return true, nil // always allow in tests
}

func TestAnalyzeHandler_ArenaDeckPayload_Success(t *testing.T) {
	repo := &analyzeMockCardRepo{cardsByName: map[string]*domain.Card{
		"Elfi di Llanowar": {
			ID:         "c-elfi",
			Name:       "Elfi di Llanowar",
			CMC:        1,
			TypeLine:   "Creature - Elf Druid",
			OracleText: "{T}: Add {G}.",
			Colors:     []string{"G"},
			Legalities: map[string]string{"standard": "legal"},
		},
		"Foresta": {
			ID:         "c-foresta",
			Name:       "Foresta",
			CMC:        0,
			TypeLine:   "Basic Land - Forest",
			OracleText: "",
			Colors:     []string{"G"},
			Legalities: map[string]string{"standard": "legal"},
		},
	}}
	uc := usecase.NewAnalyzeDeckUseCase(&analyzeMockFetcher{}, repo, 2)
	users := &analyzeMockUserRepo{}
	tracker := &mockTracker{}
	h := NewAnalyzeHandler(uc, nil, users, tracker)

	payload := map[string]string{
		"format":   "Standard",
		"decklist": "Mazzo\n4 Elfi di Llanowar (FDN) 227\n12 Foresta (EOE) 276",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-42")
	ctx = context.WithValue(ctx, middleware.ContextKeyPlan, string(domain.PlanFree))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp AnalyzeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Deterministic.Format != "standard" {
		t.Fatalf("expected normalized format standard, got %q", resp.Deterministic.Format)
	}
	if resp.Deterministic.Mana.TotalCards != 16 {
		t.Fatalf("expected total cards 16 from quantities, got %d", resp.Deterministic.Mana.TotalCards)
	}
	// Quota increment now happens atomically in FreemiumGate middleware,
	// not in the handler itself. Calling the handler directly skips it.
	if users.checkAndIncrementCalls != 0 {
		t.Fatalf("handler must not call CheckAndIncrementDailyAnalyses; got %d call(s)", users.checkAndIncrementCalls)
	}
	if tracker.calls != 1 || tracker.event != "analysis_completed" {
		t.Fatalf("expected one analysis_completed tracking event, got calls=%d event=%q", tracker.calls, tracker.event)
	}
	// Verifica legalità multi-formato
	if resp.Legality == nil {
		t.Fatalf("expected legality field in response")
	}
	if l, ok := resp.Legality["standard"]; !ok {
		t.Fatalf("expected legality for standard format")
	} else {
		if l.IsLegal {
			t.Fatalf("expected deck to be illegal in standard with 16 cards")
		}
		if l.DeckSize != 16 {
			t.Fatalf("expected standard legality deck size 16, got %d", l.DeckSize)
		}
	}
}
