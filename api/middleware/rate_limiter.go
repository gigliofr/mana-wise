package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gigliofr/mana-wise/domain"
)

// RateLimitBucket implements token bucket for rate limiting.
type RateLimitBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimitBucket creates a new token bucket.
func NewRateLimitBucket(capacity float64, refillPerSecond float64) *RateLimitBucket {
	return &RateLimitBucket{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: refillPerSecond,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and deducts a token if so.
func (b *RateLimitBucket) Allow() (allowed bool, remaining int, resetAt int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = min(b.capacity, b.tokens+elapsed*b.refillRate)
	b.lastRefill = now

	if b.tokens >= 1.0 {
		b.tokens--
		remaining = int(b.tokens)
		resetAt = now.Add(time.Duration(float64(time.Second) / b.refillRate)).Unix()
		return true, remaining, resetAt
	}

	remaining = 0
	resetAt = now.Add(time.Duration(float64(time.Second) / b.refillRate)).Unix()
	return false, remaining, resetAt
}

// RateLimiter manages per-key rate limiting.
type RateLimiter struct {
	buckets       map[string]*RateLimitBucket
	mu            sync.RWMutex
	capacity      float64 // tokens
	refillPerSec  float64 // tokens per second
	cleanupTicker *time.Ticker
	done          chan struct{}
}

// NewRateLimiter creates a rate limiter with given capacity and refill rate.
func NewRateLimiter(capacity float64, refillPerSecond float64) *RateLimiter {
	rl := &RateLimiter{
		buckets:       make(map[string]*RateLimitBucket),
		capacity:      capacity,
		refillPerSec:  refillPerSecond,
		cleanupTicker: time.NewTicker(5 * time.Minute),
		done:          make(chan struct{}),
	}

	// Start cleanup goroutine to remove stale buckets.
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a key is allowed to proceed.
func (rl *RateLimiter) Allow(key string) (allowed bool, remaining int, resetAt int64) {
	rl.mu.Lock()
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = NewRateLimitBucket(rl.capacity, rl.refillPerSec)
		rl.buckets[key] = bucket
	}
	rl.mu.Unlock()

	return bucket.Allow()
}

// Stop stops the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.done)
	rl.cleanupTicker.Stop()
}

func (rl *RateLimiter) cleanupLoop() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			for key := range rl.buckets {
				delete(rl.buckets, key)
			}
			rl.mu.Unlock()
		case <-rl.done:
			return
		}
	}
}

// RateLimitMiddleware creates a middleware that rate-limits based on (user_id, ip) pairs.
// Different limits apply based on user plan.
func RateLimitMiddleware(userRepo domain.UserRepository, requestsPerMinute func(plan string) float64) func(http.Handler) http.Handler {
	limiters := make(map[string]*RateLimiter)
	var mu sync.RWMutex

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			ip := getClientIP(r)
			plan := PlanFromContext(r.Context())

			requestLimit := requestsPerMinute(plan) / 60.0 // convert to per-second

			limitKey := fmt.Sprintf("%s:%s:%s", userID, ip, plan)

			mu.RLock()
			limiter, exists := limiters[limitKey]
			mu.RUnlock()

			if !exists {
				limiter = NewRateLimiter(requestLimit, requestLimit)
				mu.Lock()
				limiters[limitKey] = limiter
				mu.Unlock()
			}

			allowed, remaining, resetAt := limiter.Allow(limitKey)
			if !allowed {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", requestLimit*60))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))
				w.Header().Set("Retry-After", fmt.Sprintf("%d", resetAt-time.Now().Unix()))
				w.Header().Set("Content-Type", "application/json")

				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "rate limit exceeded",
				})
				return
			}

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", requestLimit*60))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))

			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For (reverse proxy)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ips := strings.FieldsFunc(forwarded, func(r rune) bool { return r == ',' })
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
