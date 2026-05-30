package middleware

import (
	"context"
	"net/http"
	"time"
)

// ContextDeadlineMiddleware adds a request-level deadline to expensive operations.
// This ensures handlers have a bounded execution time and can be cancelled gracefully.
func ContextDeadlineMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
