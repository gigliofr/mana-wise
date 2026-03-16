package middleware

import (
	"fmt"
	"net/http"

	"github.com/manawise/api/domain"
)

// FreemiumGate is an HTTP middleware that enforces daily analysis limits
// for free users. It calls userRepo to check/update quota atomically.
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
			user, err := userRepo.FindByID(r.Context(), userID)
			if err != nil || user == nil {
				http.Error(w, `{"error":"user not found"}`, http.StatusUnauthorized)
				return
			}

			if !user.CanAnalyze(today) {
				remaining := user.RemainingAnalyses(today)
				_ = tracker.Track(r.Context(), userID, "daily_limit_reached", map[string]interface{}{
					"plan":      plan,
					"remaining": remaining,
					"limit":     domain.FreeDailyLimit,
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				fmt.Fprintf(w, `{"error":"daily limit reached","limit":%d,"remaining":%d,"upgrade_url":"/upgrade"}`,
					domain.FreeDailyLimit, remaining)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
