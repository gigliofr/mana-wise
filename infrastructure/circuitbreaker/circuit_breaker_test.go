package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, 1, 100*time.Millisecond)

	// First 2 failures should stay closed.
	for i := 0; i < 2; i++ {
		err := cb.Call(func() error { return errors.New("fail") })
		if err == nil {
			t.Fatal("expected error")
		}
		if cb.State() != StateClosed {
			t.Fatalf("expected closed after failure %d, got %v", i+1, cb.State())
		}
	}

	// Third failure should open the circuit.
	err := cb.Call(func() error { return errors.New("fail") })
	if err == nil {
		t.Fatal("expected error")
	}
	if cb.State() != StateOpen {
		t.Fatalf("expected open after 3rd failure, got %v", cb.State())
	}

	// Further calls should fail immediately.
	err = cb.Call(func() error { return nil })
	if err == nil || err.Error() != "circuit breaker \"test\" is open" {
		t.Fatalf("expected circuit open error, got %v", err)
	}
}

func TestCircuitBreaker_OpenToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, 1, 50*time.Millisecond)

	// Fail once to open.
	cb.Call(func() error { return errors.New("fail") })
	if cb.State() != StateOpen {
		t.Fatal("expected open")
	}

	// Wait for timeout and try again.
	time.Sleep(100 * time.Millisecond)

	// Next call should transition to half-open internally.
	cb.Call(func() error { return nil })

	// After successful call in half-open, should close.
	if cb.State() != StateClosed {
		t.Fatalf("expected closed after success in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, 2, 50*time.Millisecond)

	// Open the circuit.
	cb.Call(func() error { return errors.New("fail") })

	time.Sleep(100 * time.Millisecond)

	// Enter half-open with a successful call.
	cb.Call(func() error { return nil })
	if cb.State() != StateHalfOpen {
		t.Fatalf("expected half-open after success, but got %v", cb.State())
	}

	// Fail in half-open should go back to open.
	cb.Call(func() error { return errors.New("fail") })
	if cb.State() != StateOpen {
		t.Fatalf("expected open after failure in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker("test", 2, 1, 100*time.Millisecond)

	cb.Call(func() error { return errors.New("fail") })
	cb.Call(func() error { return errors.New("fail") })

	stats := cb.Stats()
	if stats["name"] != "test" {
		t.Fatalf("expected name 'test', got %v", stats["name"])
	}
	if stats["state"] != "open" {
		t.Fatalf("expected state 'open', got %v", stats["state"])
	}
	if stats["failure_count"].(int) != 2 {
		t.Fatalf("expected failure_count 2, got %v", stats["failure_count"])
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, 1, 100*time.Millisecond)

	cb.Call(func() error { return errors.New("fail") })
	if cb.State() != StateOpen {
		t.Fatal("expected open")
	}

	cb.Reset()
	if cb.State() != StateClosed {
		t.Fatalf("expected closed after reset, got %v", cb.State())
	}

	// Should accept calls now.
	err := cb.Call(func() error { return nil })
	if err != nil {
		t.Fatalf("expected no error after reset, got %v", err)
	}
}

func TestCircuitBreaker_StateChangeCallback(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, 1, 50*time.Millisecond)

	var states []State
	cb.SetStateChangeCallback(func(oldState, newState State) {
		states = append(states, newState)
	})

	cb.Call(func() error { return errors.New("fail") })
	time.Sleep(100 * time.Millisecond)
	cb.Call(func() error { return nil })

	if len(states) < 2 {
		t.Fatalf("expected at least 2 state changes, got %d", len(states))
	}
	if states[0] != StateOpen {
		t.Fatalf("expected first state change to open, got %v", states[0])
	}
	if states[1] != StateHalfOpen {
		t.Fatalf("expected second state change to half-open, got %v", states[1])
	}
}
