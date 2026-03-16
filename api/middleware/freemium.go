package middleware

import (
	"fmt"
	"net/http"

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
				http.Error(w, `{"error":"unauthenticated"}`, http.StatusUnauthorized)
				return
			}

			plan := PlanFromContext(r.Context())
			if plan == string(domain.PlanPro) {
				next.ServeHTTP(w, r)
				return
			}

			today := domain.CurrentBusinessDay()
			// Atomic check-and-increment: quota is reserved only if the DB
			// update succeeds, preventing concurrent requests from bypassing
			// the daily limit.
			allowed, err := userRepo.CheckAndIncrementDailyAnalyses(r.Context(), userID, today, domain.FreeDailyLimit)
			if err != nil {
				http.Error(w, `{"error":"quota check failed"}`, http.StatusInternalServerError)
				return
			}
			if !allowed {
				_ = tracker.Track(r.Context(), userID, "daily_limit_reached", map[string]interface{}{
					"plan":  plan,
					"limit": domain.FreeDailyLimit,
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				fmt.Fprintf(w, `{"error":"daily limit reached","limit":%d,"remaining":0,"upgrade_url":"/upgrade"}`,
					domain.FreeDailyLimit)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
