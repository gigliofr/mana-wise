package usecase

import (
	"context"
	"fmt"
	"sync"
)

// Job is a unit of work processed by the pool.
type Job[I, O any] struct {
	Input I
	Index int
}

// Result wraps the output of a processed job.
type Result[I, O any] struct {
	Output O
	Input  I
	Index  int
	Err    error
}

// WorkerPool processes a slice of inputs concurrently using a fixed pool of goroutines.
// poolSize controls parallelism. Results are returned in an unordered channel.
func WorkerPool[I, O any](
	ctx context.Context,
	poolSize int,
	inputs []I,
	fn func(ctx context.Context, input I) (O, error),
) []Result[I, O] {
	if poolSize <= 0 {
		poolSize = 1
	}

	jobs := make(chan Job[I, O], len(inputs))
	results := make(chan Result[I, O], len(inputs))

	var wg sync.WaitGroup
	for w := 0; w < poolSize; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				runWorkerJob(ctx, job, fn, results)
			}
		}()
	}

	for i, inp := range inputs {
		jobs <- Job[I, O]{Input: inp, Index: i}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	out := make([]Result[I, O], 0, len(inputs))
	for r := range results {
		out = append(out, r)
	}
	return out
}

func runWorkerJob[I, O any](ctx context.Context, job Job[I, O], fn func(ctx context.Context, input I) (O, error), results chan<- Result[I, O]) {
	defer func() {
		if rec := recover(); rec != nil {
			results <- Result[I, O]{Input: job.Input, Index: job.Index, Err: panicError(rec)}
		}
	}()

	select {
	case <-ctx.Done():
		results <- Result[I, O]{Input: job.Input, Index: job.Index, Err: ctx.Err()}
	default:
		out, err := fn(ctx, job.Input)
		results <- Result[I, O]{Output: out, Input: job.Input, Index: job.Index, Err: err}
	}
}

func panicError(rec interface{}) error {
	switch v := rec.(type) {
	case error:
		return v
	default:
		return fmt.Errorf("worker panic: %v", v)
	}
}
