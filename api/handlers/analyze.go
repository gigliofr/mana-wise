package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/manawise/api/api/middleware"
	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
)

// AnalyzeRequest is the JSON body for POST /analyze.
type AnalyzeRequest struct {
	Decklist string `json:"decklist"`
	Format   string `json:"format"`
	Locale   string `json:"locale,omitempty"`
}

// AnalyzeResponse is the JSON response for POST /analyze.
type AnalyzeResponse struct {
	Deterministic domain.AnalysisResult                 `json:"deterministic"`
	AISuggestions string                                `json:"ai_suggestions"`
	AISource      string                                `json:"ai_source,omitempty"`
	AIError       string                                `json:"ai_error,omitempty"`
	LatencyMs     int64                                 `json:"latency_ms"`
	Legality      map[string]usecase.DeckLegalityResult `json:"legality"`
	Sideboard     *sideboardResponseInfo                `json:"sideboard,omitempty"`
}

// sideboardResponseInfo is the sideboard portion included in the analyze response.
type sideboardResponseInfo struct {
	TotalCards int `json:"total_cards"`
}

// AnalyzeHandler handles POST /analyze requests.
type AnalyzeHandler struct {
	analyzeDeck *usecase.AnalyzeDeckUseCase
	ai          *usecase.AISuggester
	userRepo    domain.UserRepository
	tracker     domain.AnalyticsTracker
}

// NewAnalyzeHandler creates an AnalyzeHandler.
func NewAnalyzeHandler(uc *usecase.AnalyzeDeckUseCase, ai *usecase.AISuggester, userRepo domain.UserRepository, tracker domain.AnalyticsTracker) *AnalyzeHandler {
	if tracker == nil {
		tracker = domain.NoopAnalyticsTracker{}
	}
	return &AnalyzeHandler{analyzeDeck: uc, ai: ai, userRepo: userRepo, tracker: tracker}
}

// ServeHTTP handles POST /analyze.
func (h *AnalyzeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.Format = domain.NormalizeFormat(strings.TrimSpace(req.Format))
	req.Decklist = strings.TrimSpace(req.Decklist)
	req.Locale = normalizeLocale(strings.TrimSpace(req.Locale), r.Header.Get("Accept-Language"))

	if req.Decklist == "" {
		jsonError(w, "decklist is required", http.StatusBadRequest)
		return
	}
	if req.Format == "" {
		jsonError(w, "format is required", http.StatusBadRequest)
		return
	}

	result, err := h.analyzeDeck.Execute(r.Context(), usecase.AnalyzeDeckRequest{
		Decklist: req.Decklist,
		Format:   req.Format,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// LLM suggestions (best-effort — do not fail the request if LLM is unavailable).
	var aiSuggestions string
	var aiSource string
	var aiError string
	if h.ai != nil {
		extErr, suggestErr := error(nil), error(nil)
		aiSuggestions, aiSource, extErr, suggestErr = h.ai.Suggest(r.Context(), req.Decklist, req.Format, req.Locale, &result.Result, result.RawCards)
		if suggestErr != nil {
			aiError = suggestErr.Error()
		} else if extErr != nil {
			aiError = "LLM unavailable (falling back to internal rules): " + extErr.Error()
		}
	}

	result.Result.LatencyMs = time.Since(start).Milliseconds()
	userID := middleware.UserIDFromContext(r.Context())
	plan := middleware.PlanFromContext(r.Context())
	_ = h.tracker.Track(r.Context(), userID, "analysis_completed", map[string]interface{}{
		"format":     req.Format,
		"plan":       plan,
		"latency_ms": result.Result.LatencyMs,
	})

	legality := usecase.DetermineDeckLegalityAllFormats(result.RawCards, result.Quantities)

	// Augment Commander legality with color identity check when commander is known.
	if result.Commander != nil && len(result.Commander.Cards) > 0 {
		if cmdResult, ok := legality["commander"]; ok {
			violations := usecase.CheckCommanderColorIdentity(result.Commander.Cards, result.RawCards, result.Quantities)
			if len(violations) > 0 {
				cmdResult.IllegalCards = append(cmdResult.IllegalCards, violations...)
				cmdResult.IsLegal = false
				legality["commander"] = cmdResult
			}
		}
	}

	var sb *sideboardResponseInfo
	if result.Sideboard != nil {
		sb = &sideboardResponseInfo{TotalCards: result.Sideboard.TotalCards}
	}

	jsonOK(w, AnalyzeResponse{
		Deterministic: result.Result,
		AISuggestions: aiSuggestions,
		AISource:      aiSource,
		AIError:       aiError,
		LatencyMs:     result.Result.LatencyMs,
		Legality:      legality,
		Sideboard:     sb,
	})
}

func normalizeLocale(requestLocale, acceptLanguage string) string {
	locale := strings.ToLower(strings.TrimSpace(requestLocale))
	if strings.HasPrefix(locale, "it") {
		return "it"
	}
	if strings.HasPrefix(locale, "en") {
		return "en"
	}
	accept := strings.ToLower(strings.TrimSpace(acceptLanguage))
	if strings.HasPrefix(accept, "it") {
		return "it"
	}
	return "en"
}
