package handlers

import (
	"crypto/subtle"
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
	commanderBrackets  CommanderBracketStore
}

// CommanderBracketStore gives concurrency-safe access to the live bracket
// configuration (implemented by usecase.CommanderBracketUseCase).
type CommanderBracketStore interface {
	Config() domain.CommanderBracketConfig
	SetConfig(domain.CommanderBracketConfig)
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(userRepo domain.UserRepository, metrics domain.AnalyticsMetricsProvider, commanderBrackets CommanderBracketStore) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, metrics: metrics, commanderBrackets: commanderBrackets}
}

// UpdateUserPlanRequest is the JSON body for POST /admin/user/plan.
type UpdateUserPlanRequest struct {
	Email string `json:"email"`
	Plan  string `json:"plan"`
	// DurationDays grants Pro for the given number of days, extending any
	// active entitlement. 0 grants Pro without expiry.
	DurationDays int `json:"duration_days,omitempty"`
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

	if req.DurationDays < 0 || req.DurationDays > 3660 {
		WriteAPIErrorFromMsg(w, "duration_days must be between 0 and 3660", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	if req.Plan == "pro" {
		user.Plan = domain.PlanPro
		if req.DurationDays > 0 {
			base := now
			if user.ProUntil != nil && user.ProUntil.After(base) {
				base = *user.ProUntil
			}
			until := base.AddDate(0, 0, req.DurationDays)
			user.ProUntil = &until
		} else {
			user.ProUntil = nil
		}
	} else {
		user.Plan = domain.PlanFree
		user.ProUntil = nil
	}
	user.UpdatedAt = now

	if err := h.userRepo.Update(r.Context(), user); err != nil {
		WriteAPIErrorFromMsg(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]interface{}{
		"email":     user.Email,
		"plan":      user.Plan,
		"pro_until": user.ProUntil,
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
	jsonOK(w, CommanderBracketsResponse{Config: h.commanderBrackets.Config()})
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

	req.Config.GameChangers = normalizeStringSlice(req.Config.GameChangers)
	req.Config.ExtraTurnCards = normalizeStringSlice(req.Config.ExtraTurnCards)
	req.Config.MassLandDenialCards = normalizeStringSlice(req.Config.MassLandDenialCards)
	req.Config.TutorKeywords = normalizeStringSlice(req.Config.TutorKeywords)
	req.Config.ExtraTurnKeywords = normalizeStringSlice(req.Config.ExtraTurnKeywords)
	req.Config.MassLandDenialKeywords = normalizeStringSlice(req.Config.MassLandDenialKeywords)
	req.Config.FastManaKeywords = normalizeStringSlice(req.Config.FastManaKeywords)
	req.Config.Enabled = true

	h.commanderBrackets.SetConfig(req.Config)
	jsonOK(w, CommanderBracketsResponse{Config: h.commanderBrackets.Config()})
}

// AdminSecretMiddleware checks for a secret key header.
//
// When ADMIN_SECRET is not configured the admin API is disabled: there is no
// built-in default secret, because a default published in the repository would
// let anyone call the admin endpoints (including granting Pro).
func AdminSecretMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := strings.TrimSpace(os.Getenv("ADMIN_SECRET"))
		if secret == "" {
			WriteAPIErrorFromMsg(w, "admin API disabled: ADMIN_SECRET is not configured", http.StatusServiceUnavailable)
			return
		}

		authHeader := r.Header.Get("X-Admin-Secret")
		if subtle.ConstantTimeCompare([]byte(authHeader), []byte(secret)) != 1 {
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
