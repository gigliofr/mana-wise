package middleware

import (
	"testing"
	"time"
)

func TestRateLimitBucket_AllowsInitially(t *testing.T) {
	bucket := NewRateLimitBucket(5, 1)

	for i := 0; i < 5; i++ {
		allowed, _, _ := bucket.Allow()
		if !allowed {
			t.Fatalf("expected allow for token %d, got denied", i+1)
		}
	}

	// 6th should be denied
	allowed, _, _ := bucket.Allow()
	if allowed {
		t.Fatal("expected deny on 6th token")
	}
}

func TestRateLimitBucket_RefillsOverTime(t *testing.T) {
	bucket := NewRateLimitBucket(1, 2) // 2 tokens per second

	// Consume initial token
	allowed, _, _ := bucket.Allow()
	if !allowed {
		t.Fatal("expected allow for first token")
	}

	// Next should be denied immediately
	allowed, _, _ = bucket.Allow()
	if allowed {
		t.Fatal("expected deny without waiting")
	}

	// Wait for refill
	time.Sleep(600 * time.Millisecond)

	// Should have 1+ token now
	allowed, _, _ = bucket.Allow()
	if !allowed {
		t.Fatal("expected allow after refill time")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	limiter := NewRateLimiter(2, 1)
	defer limiter.Stop()

	// Key 1 exhausts its tokens
	for i := 0; i < 2; i++ {
		allowed, _, _ := limiter.Allow("key1")
		if !allowed {
			t.Fatalf("expected allow for key1 token %d", i+1)
		}
	}

	// Key 2 should still have tokens
	allowed, _, _ := limiter.Allow("key2")
	if !allowed {
		t.Fatal("expected allow for key2 (different bucket)")
	}
}
