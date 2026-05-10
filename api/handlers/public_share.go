package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
	"github.com/go-chi/chi/v5"
)

type PublicShareHandler struct {
	Repo      domain.SharedAnalysisLinkRepository
	DeckRepo  domain.DeckRepository
	AnalyzeUC *usecase.AnalyzeDeckUseCase
}

type sharedAnalysisBundle struct {
	Token  string
	Link   *domain.SharedAnalysisLink
	Deck   *domain.Deck
	Result *usecase.AnalyzeDeckResponse
}

func NewPublicShareHandler(repo domain.SharedAnalysisLinkRepository, deckRepo domain.DeckRepository, analyzeUC *usecase.AnalyzeDeckUseCase) *PublicShareHandler {
	return &PublicShareHandler{Repo: repo, DeckRepo: deckRepo, AnalyzeUC: analyzeUC}
}

// ServeHTTP handles GET /share/{token}
func (h *PublicShareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bundle, status, message := h.loadSharedAnalysis(r.Context(), chi.URLParam(r, "token"))
	if bundle == nil {
		jsonError(w, message, status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"deck_id":    bundle.Deck.ID,
		"analysis":   bundle.Result.Result,
		"shared_by":  bundle.Link.UserID,
		"expires_at": bundle.Link.ExpiresAt,
	})
}

func (h *PublicShareHandler) loadSharedAnalysis(ctx context.Context, token string) (*sharedAnalysisBundle, int, string) {
	if strings.TrimSpace(token) == "" {
		return nil, http.StatusBadRequest, "missing token"
	}
	if h == nil || h.Repo == nil || h.DeckRepo == nil || h.AnalyzeUC == nil {
		return nil, http.StatusServiceUnavailable, "shared analysis is unavailable"
	}
	link, err := h.Repo.FindByID(ctx, token)
	if err != nil || link == nil {
		return nil, http.StatusNotFound, "Questo link non esiste o è stato rimosso."
	}
	if time.Now().After(link.ExpiresAt) {
		return nil, http.StatusGone, "Questo link è scaduto. Chiedi una nuova condivisione."
	}
	_ = h.Repo.IncrementVisit(ctx, token, time.Now())
	deck, err := h.DeckRepo.FindByID(ctx, link.DeckID)
	if err != nil || deck == nil {
		return nil, http.StatusNotFound, "deck not found"
	}
	result, err := h.AnalyzeUC.Execute(ctx, usecase.AnalyzeDeckRequest{
		Decklist: deckToDecklist(deck),
		Format:   deck.Format,
	})
	if err != nil || result == nil {
		return nil, http.StatusInternalServerError, "analysis unavailable"
	}
	return &sharedAnalysisBundle{Token: token, Link: link, Deck: deck, Result: result}, 0, ""
}
