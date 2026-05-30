package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// BackoffStrategy computes delay for attempt number.
type BackoffStrategy func(attempt int) time.Duration

// ExponentialBackoff implements exponential backoff with jitter.
func ExponentialBackoff(baseDelay time.Duration, maxDelay time.Duration) BackoffStrategy {
	return func(attempt int) time.Duration {
		if attempt <= 0 {
			return 0
		}
		delay := time.Duration(math.Pow(2, float64(attempt-1))) * baseDelay
		if delay > maxDelay {
			delay = maxDelay
		}
		// Add jitter: ±20% of delay
		jitter := time.Duration(rand.Int63n(int64(delay / 5)))
		return delay + jitter
	}
}

// Retrier manages retry logic.
type Retrier struct {
	maxRetries int
	backoff    BackoffStrategy
	isRetryable func(error) bool
}

// NewRetrier creates a new retrier.
func NewRetrier(maxRetries int, backoff BackoffStrategy, isRetryable func(error) bool) *Retrier {
	return &Retrier{
		maxRetries:  maxRetries,
		backoff:     backoff,
		isRetryable: isRetryable,
	}
}

// Do retries the provided function up to maxRetries times.
// Returns (result, error, retryCount, totalDelay).
func (r *Retrier) Do(ctx context.Context, fn func(ctx context.Context) error) (error, int, time.Duration) {
	var lastErr error
	var totalDelay time.Duration

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		// Check context before attempt
		if ctx.Err() != nil {
			return ctx.Err(), attempt, totalDelay
		}

		err := fn(ctx)
		if err == nil {
			return nil, attempt, totalDelay
		}

		lastErr = err

		// Check if retryable and if we have retries left
		if attempt >= r.maxRetries || !r.isRetryable(err) {
			return err, attempt + 1, totalDelay
		}

		// Compute backoff delay
		delay := r.backoff(attempt + 1)
		totalDelay += delay

		// Wait with context cancellation
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err(), attempt + 1, totalDelay
		}
	}

	return lastErr, r.maxRetries + 1, totalDelay
}

// DefaultIsRetryableTransientError checks for common transient errors.
func DefaultIsRetryableTransientError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	transientErrors := []string{
		"context deadline exceeded",
		"timeout",
		"timed out",
		"too many requests",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"connection refused",
		"connection reset",
		"i/o timeout",
	}

	for _, transient := range transientErrors {
		if contains(errStr, transient) {
			return true
		}
	}

	return false
}

func contains(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
