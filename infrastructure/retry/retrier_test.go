package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrier_SuccessOnFirstAttempt(t *testing.T) {
	retrier := NewRetrier(3, ExponentialBackoff(10*time.Millisecond, 100*time.Millisecond), DefaultIsRetryableTransientError)

	attempts := 0
	err, retryCount, _ := retrier.Do(context.Background(), func(ctx context.Context) error {
		attempts++
		return nil
	})

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if retryCount != 0 {
		t.Fatalf("expected 0 retries, got %d", retryCount)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetrier_RetryOnTransientError(t *testing.T) {
	retrier := NewRetrier(3, ExponentialBackoff(10*time.Millisecond, 50*time.Millisecond), DefaultIsRetryableTransientError)

	attempts := 0
	err, retryCount, totalDelay := retrier.Do(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("timeout")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected eventual success, got error: %v", err)
	}
	if retryCount != 2 {
		t.Fatalf("expected 2 retries, got %d", retryCount)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if totalDelay == 0 {
		t.Fatal("expected non-zero total delay")
	}
}

func TestRetrier_StopsOnNonRetryableError(t *testing.T) {
	retrier := NewRetrier(3, ExponentialBackoff(10*time.Millisecond, 50*time.Millisecond), DefaultIsRetryableTransientError)

	attempts := 0
	err, retryCount, _ := retrier.Do(context.Background(), func(ctx context.Context) error {
		attempts++
		return errors.New("invalid input")
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if retryCount != 1 {
		t.Fatalf("expected 1 attempt (no retry), got %d", retryCount)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetrier_ExhaustsRetries(t *testing.T) {
	retrier := NewRetrier(2, ExponentialBackoff(5*time.Millisecond, 20*time.Millisecond), DefaultIsRetryableTransientError)

	attempts := 0
	err, retryCount, _ := retrier.Do(context.Background(), func(ctx context.Context) error {
		attempts++
		return errors.New("timeout")
	})

	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if retryCount != 3 {
		t.Fatalf("expected 3 attempts (initial + 2 retries), got %d", retryCount)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetrier_ContextCancellation(t *testing.T) {
	retrier := NewRetrier(5, ExponentialBackoff(50*time.Millisecond, 100*time.Millisecond), DefaultIsRetryableTransientError)

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	err, _, _ := retrier.Do(ctx, func(ctx context.Context) error {
		attempts++
		return errors.New("timeout")
	})

	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if attempts > 2 {
		t.Fatalf("expected early cancellation, got %d attempts", attempts)
	}
}
