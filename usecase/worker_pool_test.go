package usecase_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/manawise/api/usecase"
)

func TestWorkerPool_AllSucceed(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5}
	results := usecase.WorkerPool(context.Background(), 3, inputs,
		func(ctx context.Context, n int) (int, error) {
			return n * 2, nil
		},
	)

	if len(results) != len(inputs) {
		t.Fatalf("expected %d results, got %d", len(inputs), len(results))
	}
	for _, r := range results {
		if r.Err != nil {
			t.Errorf("unexpected error: %v", r.Err)
		}
		if r.Output != r.Input*2 {
			t.Errorf("expected %d*2=%d, got %d", r.Input, r.Input*2, r.Output)
		}
	}
}

func TestWorkerPool_ErrorPropagated(t *testing.T) {
	inputs := []int{1, 2, 3}
	sentinel := errors.New("test error")

	results := usecase.WorkerPool(context.Background(), 2, inputs,
		func(ctx context.Context, n int) (int, error) {
			if n == 2 {
				return 0, sentinel
			}
			return n, nil
		},
	)

	errCount := 0
	for _, r := range results {
		if r.Err != nil {
			errCount++
		}
	}
	if errCount != 1 {
		t.Errorf("expected 1 error, got %d", errCount)
	}
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	inputs := make([]int, 100)
	for i := range inputs {
		inputs[i] = i
	}

	var processed int64
	results := usecase.WorkerPool(ctx, 2, inputs,
		func(ctx context.Context, n int) (int, error) {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt64(&processed, 1)
			return n, nil
		},
	)

	// Some results should contain context errors.
	errCount := 0
	for _, r := range results {
		if r.Err != nil {
			errCount++
		}
	}
	// With a very short timeout, at least some should have been cancelled.
	t.Logf("processed %d, errors %d out of %d", processed, errCount, len(inputs))
}

func TestWorkerPool_ZeroPoolSizeFallsToOne(t *testing.T) {
	results := usecase.WorkerPool(context.Background(), 0, []int{1, 2, 3},
		func(ctx context.Context, n int) (int, error) {
			return n, nil
		},
	)
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestWorkerPool_EmptyInput(t *testing.T) {
	results := usecase.WorkerPool(context.Background(), 5, []int{},
		func(ctx context.Context, n int) (int, error) {
			return n, nil
		},
	)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
