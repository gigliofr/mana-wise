package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gigliofr/mana-wise/domain"
)

// AdminHandler provides admin-only operations.
type AdminHandler struct {
	userRepo           domain.UserRepository
	metrics            domain.AnalyticsMetricsProvider
	commanderBrackets  *domain.CommanderBracketConfig
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(userRepo domain.UserRepository, metrics domain.AnalyticsMetricsProvider, commanderBrackets *domain.CommanderBracketConfig) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, metrics: metrics, commanderBrackets: commanderBrackets}
}

// UpdateUserPlanRequest is the JSON body for POST /admin/user/plan.
type UpdateUserPlanRequest struct {
	Email string `json:"email"`
	Plan  string `json:"plan"`
}

// CommanderBracketsResponse wraps the current commander bracket configuration.
type CommanderBracketsResponse struct {
	Config domain.CommanderBracketConfig `json:"config"`
}

// UpdateCommanderBracketsRequest updates the commander bracket configuration.
type UpdateCommanderBracketsRequest struct {
	Config domain.CommanderBracketConfig `json:"config"`
}

// UpdateUserPlan handles POST /admin/user/plan (secret-key protected).
func (h *AdminHandler) UpdateUserPlan(w http.ResponseWriter, r *http.Request) {
	var req UpdateUserPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIErrorFromMsg(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Plan = strings.ToLower(strings.TrimSpace(req.Plan))

	if req.Email == "" || (req.Plan != "free" && req.Plan != "pro") {
		WriteAPIErrorFromMsg(w, "email and plan (free/pro) are required", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.FindByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		WriteAPIErrorFromMsg(w, "user not found", http.StatusNotFound)
		return
	}

	// Update plan.
	plan := domain.PlanFree
	if req.Plan == "pro" {
		plan = domain.PlanPro
	}
	user.Plan = plan

	if err := h.userRepo.Update(r.Context(), user); err != nil {
		WriteAPIErrorFromMsg(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]interface{}{
		"email": user.Email,
		"plan":  user.Plan,
	})
}

// FunnelMetrics handles GET /admin/metrics/funnel (secret-key protected).
func (h *AdminHandler) FunnelMetrics(w http.ResponseWriter, r *http.Request) {
	if h.metrics == nil {
		WriteAPIErrorFromMsg(w, "metrics provider unavailable", http.StatusServiceUnavailable)
		return
	}

	jsonOK(w, map[string]interface{}{
		"snapshot": h.metrics.Snapshot(),
	})
}

// StabilityMetrics handles GET /admin/metrics/stability (secret-key protected).
func (h *AdminHandler) StabilityMetrics(w http.ResponseWriter, r *http.Request) {
	if h.metrics == nil {
		WriteAPIErrorFromMsg(w, "metrics provider unavailable", http.StatusServiceUnavailable)
		return
	}

	snapshot := h.metrics.Snapshot()
	cacheHitRatio := 0.0
	if snapshot.CacheHits+snapshot.CacheMisses > 0 {
		cacheHitRatio = float64(snapshot.CacheHits) / float64(snapshot.CacheHits+snapshot.CacheMisses)
	}

	jsonOK(w, map[string]interface{}{
		"timestamp_unix_ms": time.Now().UnixMilli(),
		"cache_stats": map[string]interface{}{
			"hits":       snapshot.CacheHits,
			"misses":     snapshot.CacheMisses,
			"hit_ratio":  cacheHitRatio,
		},
		"errors_24h":                  snapshot.EventCounts,
		"analysis_fallbacks":          snapshot.AnalysisFallbacks,
		"analysis_by_ai_source":       snapshot.AnalysisByAISource,
		"forwarding_errors":           snapshot.ForwardingErrors,
		"last_event_at_unix_ms":       snapshot.LastEventAtUnixMilli,
	})
}

// CommanderBrackets returns the current bracket configuration.
func (h *AdminHandler) CommanderBrackets(w http.ResponseWriter, r *http.Request) {
	if h.commanderBrackets == nil {
		WriteAPIErrorFromMsg(w, "commander bracket config unavailable", http.StatusServiceUnavailable)
		return
	}
	jsonOK(w, CommanderBracketsResponse{Config: *h.commanderBrackets})
}

// UpdateCommanderBrackets updates the in-memory bracket configuration.
func (h *AdminHandler) UpdateCommanderBrackets(w http.ResponseWriter, r *http.Request) {
	if h.commanderBrackets == nil {
		WriteAPIErrorFromMsg(w, "commander bracket config unavailable", http.StatusServiceUnavailable)
		return
	}

	var req UpdateCommanderBracketsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIErrorFromMsg(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Config.DecisiveCards = normalizeStringSlice(req.Config.DecisiveCards)
	req.Config.TutorKeywords = normalizeStringSlice(req.Config.TutorKeywords)
	req.Config.ExtraTurnKeywords = normalizeStringSlice(req.Config.ExtraTurnKeywords)
	req.Config.MassLandDenialKeywords = normalizeStringSlice(req.Config.MassLandDenialKeywords)
	req.Config.ComboKeywords = normalizeStringSlice(req.Config.ComboKeywords)
	req.Config.FastManaKeywords = normalizeStringSlice(req.Config.FastManaKeywords)
	req.Config.Enabled = true

	*h.commanderBrackets = req.Config
	jsonOK(w, CommanderBracketsResponse{Config: *h.commanderBrackets})
}

// AdminSecretMiddleware checks for a secret key header.
func AdminSecretMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := os.Getenv("ADMIN_SECRET")
		if secret == "" {
			secret = "change-me-in-production"
		}

		authHeader := r.Header.Get("X-Admin-Secret")
		if authHeader != secret {
			WriteAPIErrorFromMsg(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func normalizeStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}
