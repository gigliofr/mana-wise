package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
)

type PublicShareHandler struct {
	Repo      domain.SharedAnalysisLinkRepository
	DeckRepo  domain.DeckRepository
	AnalyzeUC *usecase.AnalyzeDeckUseCase
}

func NewPublicShareHandler(repo domain.SharedAnalysisLinkRepository, deckRepo domain.DeckRepository, analyzeUC *usecase.AnalyzeDeckUseCase) *PublicShareHandler {
	return &PublicShareHandler{Repo: repo, DeckRepo: deckRepo, AnalyzeUC: analyzeUC}
}

// ServeHTTP handles GET /share/{token}
func (h *PublicShareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if strings.TrimSpace(token) == "" {
		jsonError(w, "missing token", http.StatusBadRequest)
		return
	}
	link, err := h.Repo.FindByID(r.Context(), token)
	if err != nil || link == nil {
		jsonError(w, "Questo link non esiste o è stato rimosso.", http.StatusNotFound)
		return
	}
	if time.Now().After(link.ExpiresAt) {
		jsonError(w, "Questo link è scaduto. Chiedi una nuova condivisione.", http.StatusGone)
		return
	}
	// Tracking visita
	_ = h.Repo.IncrementVisit(r.Context(), token, time.Now())
	deck, err := h.DeckRepo.FindByID(r.Context(), link.DeckID)
	if err != nil || deck == nil {
		jsonError(w, "deck not found", http.StatusNotFound)
		return
	}
	// Run analysis (fresh)
	result, err := h.AnalyzeUC.Execute(r.Context(), usecase.AnalyzeDeckRequest{
		Decklist: deckToDecklist(deck),
		Format:   deck.Format,
	})
	if err != nil || result == nil {
		jsonError(w, "analysis unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"deck_id": deck.ID,
		"analysis": result.Result,
		"shared_by": link.UserID,
		"expires_at": link.ExpiresAt,
	})
}
