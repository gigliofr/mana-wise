package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Regression: Call used to hold the mutex while fn ran, serializing every call.
func TestCircuitBreaker_CallsRunConcurrently(t *testing.T) {
	cb := NewCircuitBreaker("test", 5, 1, time.Second)
	const n = 10
	var running, maxRunning int32
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cb.Call(func() error {
				cur := atomic.AddInt32(&running, 1)
				for {
					m := atomic.LoadInt32(&maxRunning)
					if cur <= m || atomic.CompareAndSwapInt32(&maxRunning, m, cur) {
						break
					}
				}
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt32(&running, -1)
				return nil
			})
		}()
	}
	wg.Wait()
	if maxRunning < 2 {
		t.Fatalf("calls were serialized (max concurrent = %d)", maxRunning)
	}
	if elapsed := time.Since(start); elapsed > 400*time.Millisecond {
		t.Fatalf("10 calls of 50ms took %s: they are not running in parallel", elapsed)
	}
}

func TestCircuitBreaker_OnlyConsecutiveFailuresOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, 1, time.Second)
	fail := func() error { return errors.New("fail") }
	ok := func() error { return nil }

	for i := 0; i < 10; i++ {
		_ = cb.Call(fail)
		_ = cb.Call(fail)
		_ = cb.Call(ok)
	}
	if cb.State() != StateClosed {
		t.Fatalf("interleaved failures must not open the circuit, got %s", cb.State())
	}
	_ = cb.Call(fail)
	_ = cb.Call(fail)
	_ = cb.Call(fail)
	if cb.State() != StateOpen {
		t.Fatalf("3 consecutive failures must open the circuit, got %s", cb.State())
	}
}

func TestCircuitBreaker_OpenReturnsErrOpen(t *testing.T) {
	cb := NewCircuitBreaker("mongodb", 1, 1, time.Hour)
	_ = cb.Call(func() error { return errors.New("fail") })
	called := false
	err := cb.Call(func() error { called = true; return nil })
	if !errors.Is(err, ErrOpen) {
		t.Fatalf("expected ErrOpen, got %v", err)
	}
	if called {
		t.Fatal("fn must not run while open")
	}
}

func TestCircuitBreaker_HalfOpenAllowsSingleProbe(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, 1, 20*time.Millisecond)
	_ = cb.Call(func() error { return errors.New("fail") })
	time.Sleep(40 * time.Millisecond)

	release := make(chan struct{})
	probeStarted := make(chan struct{})
	go func() {
		_ = cb.Call(func() error { close(probeStarted); <-release; return nil })
	}()
	<-probeStarted

	err := cb.Call(func() error { return nil })
	if !errors.Is(err, ErrOpen) {
		t.Fatalf("second call during probe should be rejected, got %v", err)
	}
	close(release)

	deadline := time.Now().Add(time.Second)
	for cb.State() != StateClosed && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if cb.State() != StateClosed {
		t.Fatalf("successful probe should close the circuit, got %s", cb.State())
	}
}

func TestCircuitBreaker_CanceledContextIsNotAFailure(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, 1, time.Hour)
	_ = cb.Call(func() error { return fmt.Errorf("query: %w", context.Canceled) })
	if cb.State() != StateClosed {
		t.Fatalf("client cancellation must not open the circuit, got %s", cb.State())
	}
	_ = cb.Call(func() error { return context.DeadlineExceeded })
	if cb.State() != StateOpen {
		t.Fatalf("a timeout is a dependency failure, got %s", cb.State())
	}
}

func TestCircuitBreaker_CallbackCanReadStateWithoutDeadlock(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, 1, time.Hour)
	var seen State
	cb.AddStateChangeCallback(func(_, _ State) { seen = cb.State() })
	done := make(chan struct{})
	go func() { _ = cb.Call(func() error { return errors.New("fail") }); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("deadlock: callback calling State()")
	}
	if seen != StateOpen {
		t.Fatalf("callback saw %s, want open", seen)
	}
}

func TestCircuitBreaker_CustomFailurePredicate(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, 1, time.Hour)
	userErr := errors.New("duplicate key")
	cb.SetFailurePredicate(func(err error) bool { return err != nil && !errors.Is(err, userErr) })
	_ = cb.Call(func() error { return userErr })
	if cb.State() != StateClosed {
		t.Fatalf("ignored error must not open the circuit, got %s", cb.State())
	}
}
