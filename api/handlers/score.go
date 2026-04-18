package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gigliofr/mana-wise/api/middleware"
	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
)

// ScoreRequest is the JSON body for scoring requests.
type ScoreRequest struct {
	Decklist string `json:"decklist"`
	Format   string `json:"format"`
}

// ScoreHandler handles endpoints that calculate detailed scores.
type ScoreHandler struct {
	analyzeDeck *usecase.AnalyzeDeckUseCase
	scoreUC     *usecase.ScoreUseCase
	impactUC    *usecase.ImpactScoreUseCase
	bracketUC   *usecase.CommanderBracketUseCase
	userRepo    domain.UserRepository
}

// NewScoreHandler creates a ScoreHandler.
func NewScoreHandler(
	analyzeDeck *usecase.AnalyzeDeckUseCase,
	scoreUC *usecase.ScoreUseCase,
	impactUC *usecase.ImpactScoreUseCase,
	bracketUC *usecase.CommanderBracketUseCase,
	userRepo domain.UserRepository,
) *ScoreHandler {
	return &ScoreHandler{
		analyzeDeck: analyzeDeck,
		scoreUC:     scoreUC,
		impactUC:    impactUC,
		bracketUC:   bracketUC,
		userRepo:    userRepo,
	}
}

// Score handles POST /score requests.
func (h *ScoreHandler) Score(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req ScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Format = domain.NormalizeFormat(strings.TrimSpace(req.Format))
	req.Decklist = strings.TrimSpace(req.Decklist)

	if req.Decklist == "" {
		jsonError(w, "decklist is required", http.StatusBadRequest)
		return
	}
	if req.Format == "" {
		jsonError(w, "format is required", http.StatusBadRequest)
		return
	}

	// Parse decklist
	result, err := h.analyzeDeck.Execute(r.Context(), usecase.AnalyzeDeckRequest{
		Decklist: req.Decklist,
		Format:   req.Format,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// Calculate score
	scoreResult, err := h.scoreUC.Execute(r.Context(), result.RawCards, result.Quantities)
	if err != nil {
		jsonError(w, "score calculation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Build response
	detail := &domain.ScoreDetail{
		Score:        scoreResult.Score,
		TotalImpact:  scoreResult.TotalImpact,
		TippingPoint: scoreResult.TippingPoint,
		ImpactByCMC:  scoreResult.ImpactByCMC,
		ManaScrew:    scoreResult.ManaAnalysis.ManaScrew,
		ManaFlood:    scoreResult.ManaAnalysis.ManaFlood,
		SweetSpot:    scoreResult.ManaAnalysis.SweetSpot,
		CardImpacts:  scoreResult.CardImpacts,
	}

	// Check freemium quota if user is authenticated
	if userID, ok := r.Context().Value(middleware.ContextKeyUserID).(string); ok && userID != "" {
		h.userRepo.CheckAndIncrementDailyAnalyses(r.Context(), userID, "score", 1)
	}

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"score_detail": detail,
		"commander_bracket": func() *domain.CommanderBracketAssessment {
			if h.bracketUC == nil || req.Format != "commander" {
				return nil
			}
			return h.bracketUC.Evaluate(result.RawCards, result.Quantities)
		}(),
		"latency_ms":   time.Since(start).Milliseconds(),
	}
	json.NewEncoder(w).Encode(resp)
}
