package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
)

// AdminHandler provides admin-only operations.
type AdminHandler struct {
	userRepo domain.UserRepository
	metrics  domain.AnalyticsMetricsProvider
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(userRepo domain.UserRepository, metrics domain.AnalyticsMetricsProvider) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, metrics: metrics}
}

// UpdateUserPlanRequest is the JSON body for POST /admin/user/plan.
type UpdateUserPlanRequest struct {
	Email string `json:"email"`
	Plan  string `json:"plan"`
}

// UpdateUserPlan handles POST /admin/user/plan (secret-key protected).
func (h *AdminHandler) UpdateUserPlan(w http.ResponseWriter, r *http.Request) {
	var req UpdateUserPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Plan = strings.ToLower(strings.TrimSpace(req.Plan))

	if req.Email == "" || (req.Plan != "free" && req.Plan != "pro") {
		jsonError(w, "email and plan (free/pro) are required", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.FindByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	// Update plan.
	plan := domain.PlanFree
	if req.Plan == "pro" {
		plan = domain.PlanPro
	}
	user.Plan = plan

	if err := h.userRepo.Update(r.Context(), user); err != nil {
		jsonError(w, "failed to update user", http.StatusInternalServerError)
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
		jsonError(w, "metrics provider unavailable", http.StatusServiceUnavailable)
		return
	}

	jsonOK(w, map[string]interface{}{
		"snapshot": h.metrics.Snapshot(),
	})
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
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
