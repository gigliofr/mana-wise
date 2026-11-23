package mongodb

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gigliofr/mana-wise/infrastructure/circuitbreaker"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestIsMongoFailure(t *testing.T) {
	dup := mongo.WriteException{WriteErrors: mongo.WriteErrors{{Code: 11000, Message: "E11000 duplicate key"}}}
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"no documents", mongo.ErrNoDocuments, false},
		{"wrapped no documents", fmt.Errorf("x: %w", mongo.ErrNoDocuments), false},
		{"client cancelled", context.Canceled, false},
		{"duplicate key", dup, false},
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"unknown error", errors.New("connection refused"), true},
	}
	for _, tc := range cases {
		if got := isMongoFailure(tc.err); got != tc.want {
			t.Errorf("%s: isMongoFailure=%v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestGuard_AppliesTimeoutsAndBreaker(t *testing.T) {
	var deadline time.Time
	g := guard{}
	_ = g.read(context.Background(), func(ctx context.Context) error {
		deadline, _ = ctx.Deadline()
		return nil
	})
	if until := time.Until(deadline); until <= 0 || until > DefaultQueryTimeouts().Read {
		t.Fatalf("read must carry the read timeout, got %s", until)
	}
	_ = g.write(context.Background(), func(ctx context.Context) error {
		deadline, _ = ctx.Deadline()
		return nil
	})
	if until := time.Until(deadline); until <= 0 || until > DefaultQueryTimeouts().Write {
		t.Fatalf("write must carry the write timeout, got %s", until)
	}

	cb := circuitbreaker.NewCircuitBreaker("mongodb", 1, 1, time.Hour)
	cb.SetFailurePredicate(isMongoFailure)
	g = guard{cb: cb}
	_ = g.read(context.Background(), func(context.Context) error { return mongo.ErrNoDocuments })
	if cb.State() != circuitbreaker.StateClosed {
		t.Fatal("a missing document must not open the circuit")
	}
	_ = g.read(context.Background(), func(context.Context) error { return errors.New("server selection timeout") })
	if cb.State() != circuitbreaker.StateOpen {
		t.Fatal("an infrastructure error must open the circuit")
	}
	if err := g.write(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("calls must fail fast while open, got %v", err)
	}
}
