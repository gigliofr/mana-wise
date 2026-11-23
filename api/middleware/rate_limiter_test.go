package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitBucket_AllowsBurstThenDenies(t *testing.T) {
	bucket := NewRateLimitBucket(5, 1)

	for i := 0; i < 5; i++ {
		if allowed, _, _ := bucket.Allow(); !allowed {
			t.Fatalf("expected allow for token %d, got denied", i+1)
		}
	}
	if allowed, _, _ := bucket.Allow(); allowed {
		t.Fatal("expected deny on 6th token")
	}
}

func TestRateLimitBucket_RefillsOverTime(t *testing.T) {
	bucket := NewRateLimitBucket(1, 2) // 2 tokens per second
	start := time.Now()

	if ok, _, _ := bucket.allowAt(start); !ok {
		t.Fatal("expected allow for first token")
	}
	if ok, _, _ := bucket.allowAt(start); ok {
		t.Fatal("expected deny without waiting")
	}
	if ok, _, _ := bucket.allowAt(start.Add(600 * time.Millisecond)); !ok {
		t.Fatal("expected allow after refill time")
	}
}

// Regression: a capacity below 1 used to make every request fail.
func TestRateLimitBucket_CapacityClampedToOne(t *testing.T) {
	bucket := NewRateLimitBucket(5.0/3600.0, 5.0/3600.0)
	if allowed, _, _ := bucket.Allow(); !allowed {
		t.Fatal("first request must be allowed even with a fractional capacity")
	}
}

func TestRateLimitBucket_ResetAtIsInTheFutureWhenEmpty(t *testing.T) {
	bucket := NewRateLimitBucket(1, 1.0/60.0) // 1 per minute
	now := time.Now()
	bucket.allowAt(now)
	_, _, resetAt := bucket.allowAt(now)
	if wait := resetAt - now.Unix(); wait < 50 || wait > 61 {
		t.Fatalf("resetAt should be ~60s ahead, got %ds", wait)
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	limiter := NewRateLimiter(time.Minute)
	defer limiter.Stop()

	for i := 0; i < 2; i++ {
		if allowed, _, _ := limiter.Allow("key1", 2, 1); !allowed {
			t.Fatalf("expected allow for key1 token %d", i+1)
		}
	}
	if allowed, _, _ := limiter.Allow("key1", 2, 1); allowed {
		t.Fatal("expected key1 to be exhausted")
	}
	if allowed, _, _ := limiter.Allow("key2", 2, 1); !allowed {
		t.Fatal("expected allow for key2 (different bucket)")
	}
}

func TestRateLimiter_LimitChangeRecreatesBucket(t *testing.T) {
	limiter := NewRateLimiter(time.Minute)
	defer limiter.Stop()

	limiter.Allow("user", 1, 1.0/60)
	if allowed, _, _ := limiter.Allow("user", 1, 1.0/60); allowed {
		t.Fatal("expected free bucket to be exhausted")
	}
	// Plan upgrade: higher limits must apply immediately.
	if allowed, _, _ := limiter.Allow("user", 50, 50.0/60); !allowed {
		t.Fatal("expected allow after limit increase")
	}
}

func TestRateLimiter_EvictsOnlyIdleBuckets(t *testing.T) {
	limiter := NewRateLimiter(time.Minute)
	defer limiter.Stop()

	limiter.Allow("idle", 5, 1)
	limiter.Allow("active", 5, 1)

	limiter.mu.Lock()
	limiter.buckets["idle"].lastSeen = time.Now().Add(-2 * time.Minute)
	limiter.mu.Unlock()

	limiter.evictIdle(time.Now())

	if limiter.Len() != 1 {
		t.Fatalf("expected 1 bucket after eviction, got %d", limiter.Len())
	}
	limiter.mu.Lock()
	_, stillThere := limiter.buckets["active"]
	limiter.mu.Unlock()
	if !stillThere {
		t.Fatal("active bucket must not be evicted")
	}
}

func TestRateLimiter_StopIsIdempotent(t *testing.T) {
	limiter := NewRateLimiter(time.Minute)
	limiter.Stop()
	limiter.Stop()
}

func TestRateLimitMiddleware_KeysByUserAndUsesPlanLimit(t *testing.T) {
	limiter := NewRateLimiter(time.Minute)
	defer limiter.Stop()

	mw := RateLimitMiddleware(limiter, func(plan string) int {
		if plan == "pro" {
			return 2
		}
		return 1
	})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))

	call := func(userID, plan string) int {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		ctx := context.WithValue(req.Context(), ContextKeyUserID, userID)
		ctx = context.WithValue(ctx, ContextKeyPlan, plan)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req.WithContext(ctx))
		return rr.Code
	}

	if code := call("a", "free"); code != http.StatusOK {
		t.Fatalf("free first: %d", code)
	}
	if code := call("a", "free"); code != http.StatusTooManyRequests {
		t.Fatalf("free second: %d, want 429", code)
	}
	if code := call("b", "pro"); code != http.StatusOK {
		t.Fatalf("pro first: %d", code)
	}
	if code := call("b", "pro"); code != http.StatusOK {
		t.Fatalf("pro second: %d", code)
	}
	if code := call("b", "pro"); code != http.StatusTooManyRequests {
		t.Fatalf("pro third: %d, want 429", code)
	}
}
