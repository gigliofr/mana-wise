package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gigliofr/mana-wise/api/middleware"
	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
	"github.com/go-chi/chi/v5"
)

type pdfMockDeckRepo struct {
	deck *domain.Deck
}

func (r *pdfMockDeckRepo) FindByID(ctx context.Context, id string) (*domain.Deck, error) {
	if r.deck != nil && r.deck.ID == id {
		return r.deck, nil
	}
	return nil, nil
}

func (r *pdfMockDeckRepo) FindByUserID(ctx context.Context, userID string) ([]*domain.Deck, error) {
	return nil, nil
}
func (r *pdfMockDeckRepo) Create(ctx context.Context, deck *domain.Deck) error {
	r.deck = deck
	return nil
}
func (r *pdfMockDeckRepo) Update(ctx context.Context, deck *domain.Deck) error {
	r.deck = deck
	return nil
}
func (r *pdfMockDeckRepo) Delete(ctx context.Context, id string) error { return nil }

func TestPublicSharePDFHandler_ReturnsPDFAttachment(t *testing.T) {
	repo := newMemorySharedAnalysisRepo()
	deckRepo := &pdfMockDeckRepo{deck: &domain.Deck{
		ID:     "deck-123",
		Format: "commander",
		Cards:  []domain.DeckCard{{CardName: "Sol Ring", Quantity: 1}, {CardName: "Island", Quantity: 99}},
	}}
	cardRepo := &analyzeMockCardRepo{cardsByName: map[string]*domain.Card{
		"Sol Ring": {ID: "c-sol-ring", Name: "Sol Ring", CMC: 1, TypeLine: "Artifact", Colors: []string{}, Legalities: map[string]string{"commander": "legal"}},
		"Island":   {ID: "c-island", Name: "Island", CMC: 0, TypeLine: "Basic Land - Island", Colors: []string{}, Legalities: map[string]string{"commander": "legal"}},
	}}
	analyzeUC := usecase.NewAnalyzeDeckUseCase(&analyzeMockFetcher{}, cardRepo, 2)
	payload := usecase.ShareAnalysisRequest{DeckID: "deck-123", Channel: "link", TTL: usecase.DefaultShareLinkTTL}
	_, err := usecase.ShareAnalysis(context.Background(), repo, payload, "https://mana-wise.geniuscrafters.it")
	if err != nil {
		t.Fatalf("create share link: %v", err)
	}
	var token string
	for id := range repo.links {
		token = id
	}
	if token == "" {
		t.Fatal("expected generated token")
	}

	h := NewPublicSharePDFHandler(repo, deckRepo, analyzeUC)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analysis/share/"+token+"/pdf", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-1"))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("token", token)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("expected application/pdf, got %q", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); cd == "" || !strings.Contains(cd, "inline") {
		t.Fatalf("expected inline Content-Disposition, got %q", cd)
	}
	body, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(body) < 4 || !bytes.HasPrefix(body, []byte("%PDF")) {
		t.Fatalf("expected PDF output, got prefix %q", string(body[:min(len(body), 8)]))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
