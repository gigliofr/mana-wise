package middleware

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimitBucket implements a token bucket.
//
// capacity is the maximum burst (number of requests that can be served back to
// back) and refillRate is expressed in tokens per second.
type RateLimitBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64
	lastRefill time.Time
	lastSeen   time.Time
	mu         sync.Mutex
}

// NewRateLimitBucket creates a new, full token bucket.
// A capacity below 1 would make the bucket unable to ever serve a request,
// so it is clamped to 1.
func NewRateLimitBucket(capacity float64, refillPerSecond float64) *RateLimitBucket {
	if capacity < 1 {
		capacity = 1
	}
	now := time.Now()
	return &RateLimitBucket{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: refillPerSecond,
		lastRefill: now,
		lastSeen:   now,
	}
}

// Allow checks if a request is allowed and deducts a token if so.
// resetAt is the unix time at which at least one token will be available again.
func (b *RateLimitBucket) Allow() (allowed bool, remaining int, resetAt int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.allowAt(time.Now())
}

func (b *RateLimitBucket) allowAt(now time.Time) (bool, int, int64) {
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		b.tokens = math.Min(b.capacity, b.tokens+elapsed*b.refillRate)
	}
	b.lastRefill = now
	b.lastSeen = now

	if b.tokens >= 1.0 {
		b.tokens--
		return true, int(b.tokens), b.nextTokenAt(now)
	}
	return false, 0, b.nextTokenAt(now)
}

func (b *RateLimitBucket) nextTokenAt(now time.Time) int64 {
	if b.refillRate <= 0 {
		return now.Add(time.Hour).Unix()
	}
	missing := 1.0 - b.tokens
	if missing <= 0 {
		return now.Unix()
	}
	wait := time.Duration(missing / b.refillRate * float64(time.Second))
	return now.Add(wait).Unix()
}

// RateLimiter keeps one token bucket per key. A single instance serves all keys;
// buckets that stay idle longer than idleTTL are evicted by a background janitor.
type RateLimiter struct {
	buckets  map[string]*RateLimitBucket
	mu       sync.Mutex
	idleTTL  time.Duration
	done     chan struct{}
	stopOnce sync.Once
}

// NewRateLimiter creates a limiter and starts its janitor goroutine.
// Call Stop to release it.
func NewRateLimiter(idleTTL time.Duration) *RateLimiter {
	if idleTTL <= 0 {
		idleTTL = 10 * time.Minute
	}
	rl := &RateLimiter{
		buckets: make(map[string]*RateLimitBucket),
		idleTTL: idleTTL,
		done:    make(chan struct{}),
	}
	go rl.janitor(idleTTL / 2)
	return rl
}

// Allow consumes a token from the bucket identified by key, creating it with the
// given capacity and refill rate if needed. If the limits for an existing key
// changed (e.g. a plan upgrade), the bucket is recreated with the new limits.
func (rl *RateLimiter) Allow(key string, capacity, refillPerSecond float64) (allowed bool, remaining int, resetAt int64) {
	rl.mu.Lock()
	bucket, ok := rl.buckets[key]
	if !ok || bucket.capacity != math.Max(1, capacity) || bucket.refillRate != refillPerSecond {
		bucket = NewRateLimitBucket(capacity, refillPerSecond)
		rl.buckets[key] = bucket
	}
	rl.mu.Unlock()
	return bucket.Allow()
}

// Len returns the number of tracked buckets (useful for metrics and tests).
func (rl *RateLimiter) Len() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return len(rl.buckets)
}

// Stop stops the janitor goroutine. It is safe to call more than once.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() { close(rl.done) })
}

func (rl *RateLimiter) janitor(every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			rl.evictIdle(now)
		case <-rl.done:
			return
		}
	}
}

func (rl *RateLimiter) evictIdle(now time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for key, b := range rl.buckets {
		b.mu.Lock()
		idle := now.Sub(b.lastSeen) > rl.idleTTL
		b.mu.Unlock()
		if idle {
			delete(rl.buckets, key)
		}
	}
}

// RateLimitMiddleware limits requests per user (or per client IP for
// unauthenticated requests). requestsPerMinute returns the allowed requests per
// minute for a plan; the same value is used as burst capacity.
//
// It must run after JWTAuth (to read user and plan from the context) and before
// any middleware that consumes quota, such as FreemiumGate.
func RateLimitMiddleware(limiter *RateLimiter, requestsPerMinute func(plan string) int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			plan := PlanFromContext(r.Context())
			perMinute := requestsPerMinute(plan)
			if perMinute < 1 {
				perMinute = 1
			}

			key := UserIDFromContext(r.Context())
			if key == "" {
				key = "ip:" + clientIP(r)
			}

			allowed, remaining, resetAt := limiter.Allow(key, float64(perMinute), float64(perMinute)/60.0)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(perMinute))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

			if !allowed {
				retryAfter := resetAt - time.Now().Unix()
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":       "rate limit exceeded",
					"code":        "ERR_RATE_LIMITED",
					"retry_after": retryAfter,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP returns the client address. Forwarded headers are already folded
// into RemoteAddr by chi's RealIP middleware, so they are not trusted here.
func clientIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
