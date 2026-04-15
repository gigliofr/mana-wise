package middleware

import (
	"net/http"
	"sync"
	"time"
)

// ipBucketState holds the sliding-window state for a single IP.
type ipBucketState struct {
	mu         sync.Mutex
	timestamps []time.Time
}

// rateLimiter is a per-IP sliding-window rate limiter.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*ipBucketState
	window  time.Duration
	limit   int
}

func newRateLimiter(window time.Duration, limit int) *rateLimiter {
	rl := &rateLimiter{
		buckets: make(map[string]*ipBucketState),
		window:  window,
		limit:   limit,
	}
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.evict()
		}
	}()
	return rl
}

// Allow returns true if the request from ip is within the rate limit.
func (rl *rateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &ipBucketState{}
		rl.buckets[ip] = b
	}
	rl.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	valid := b.timestamps[:0]
	for _, t := range b.timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	b.timestamps = valid

	if len(b.timestamps) >= rl.limit {
		return false
	}
	b.timestamps = append(b.timestamps, now)
	return true
}

func (rl *rateLimiter) evict() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-2 * rl.window)
	for ip, b := range rl.buckets {
		b.mu.Lock()
		if len(b.timestamps) == 0 || b.timestamps[len(b.timestamps)-1].Before(cutoff) {
			delete(rl.buckets, ip)
		}
		b.mu.Unlock()
	}
}

// AuthRateLimit returns a middleware that limits auth endpoints to limit
// requests per window per IP. Exceeding requests receive HTTP 429.
func AuthRateLimit(window time.Duration, limit int) func(http.Handler) http.Handler {
	rl := newRateLimiter(window, limit)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if !rl.Allow(ip) {
				writeJSONError(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}