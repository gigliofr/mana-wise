package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/manawise/api/domain"
)

// FreemiumGate is an HTTP middleware that enforces daily analysis limits
// for free users. It atomically checks and increments the quota in a single
// database operation, preventing TOCTOU races under concurrent requests.
func FreemiumGate(userRepo domain.UserRepository, tracker domain.AnalyticsTracker) func(http.Handler) http.Handler {
	if tracker == nil {
		tracker = domain.NoopAnalyticsTracker{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			if userID == "" {
				writeJSONError(w, "unauthenticated", http.StatusUnauthorized)
				return
			}

			user, err := userRepo.FindByID(r.Context(), userID)
			if err != nil || user == nil {
				writeJSONError(w, "user not found", http.StatusUnauthorized)
				return
			}

			if user.HasActivePro() {
				next.ServeHTTP(w, r)
				return
			}

			if user.Plan == domain.PlanPro && user.ProUntil != nil && !user.ProUntil.After(time.Now().UTC()) {
				user.Plan = domain.PlanFree
				user.ProUntil = nil
				_ = userRepo.Update(r.Context(), user)
			}

			plan := string(user.Plan)
			today := domain.CurrentBusinessDay()
			allowed, err := userRepo.CheckAndIncrementDailyAnalyses(r.Context(), userID, today, domain.FreeDailyLimit)
			if err != nil {
				writeJSONError(w, "quota check failed", http.StatusInternalServerError)
				return
			}
			if !allowed {
				_ = tracker.Track(r.Context(), userID, "daily_limit_reached", map[string]interface{}{
					"plan":  plan,
					"limit": domain.FreeDailyLimit,
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error":       "daily limit reached",
					"limit":       domain.FreeDailyLimit,
					"remaining":   0,
					"upgrade_url": "/upgrade",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}