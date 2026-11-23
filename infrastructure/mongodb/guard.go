package mongodb

import (
	"context"
	"errors"

	"github.com/gigliofr/mana-wise/infrastructure/circuitbreaker"
	"go.mongodb.org/mongo-driver/mongo"
)

// guard wraps every repository operation with a per-operation timeout and,
// when configured, the shared MongoDB circuit breaker.
type guard struct {
	cb *circuitbreaker.CircuitBreaker
}

func newGuard() guard { return guard{cb: defaultCircuitBreaker} }

// read runs fn with the read timeout.
func (g guard) read(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, cancel := WithReadTimeout(ctx)
	defer cancel()
	return g.run(ctx, fn)
}

// write runs fn with the write timeout.
func (g guard) write(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, cancel := WithWriteTimeout(ctx)
	defer cancel()
	return g.run(ctx, fn)
}

func (g guard) run(ctx context.Context, fn func(ctx context.Context) error) error {
	if g.cb == nil {
		return fn(ctx)
	}
	return g.cb.Call(func() error { return fn(ctx) })
}

// isMongoFailure tells the circuit breaker which errors mean "MongoDB is
// unhealthy". Missing documents, duplicate keys, other write/command errors
// caused by the request itself and client cancellations do not count.
func isMongoFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, mongo.ErrNoDocuments) || errors.Is(err, context.Canceled) {
		return false
	}
	if mongo.IsDuplicateKeyError(err) {
		return false
	}
	if mongo.IsTimeout(err) || mongo.IsNetworkError(err) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		// Server-side errors labelled as transient count; others are request errors.
		return cmdErr.HasErrorLabel("TransientTransactionError") || cmdErr.HasErrorLabel("RetryableWriteError")
	}
	var writeErr mongo.WriteException
	if errors.As(err, &writeErr) {
		return false
	}
	return true
}
