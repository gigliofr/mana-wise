package circuitbreaker

import (
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State string

const (
	StateClosed     State = "closed"
	StateOpen       State = "open"
	StateHalfOpen   State = "half-open"
)

// CircuitBreaker implements a simple circuit breaker pattern.
type CircuitBreaker struct {
	name           string
	failureCount   int
	successCount   int
	lastFailureAt  time.Time
	mu             sync.RWMutex

	// Configuration
	failureThreshold int           // Number of failures before opening
	successThreshold int           // Number of successes in half-open before closing
	timeout          time.Duration // Duration to wait before transitioning to half-open

	// State tracking
	state               State
	openedAt            time.Time
	// state change callbacks. Use AddStateChangeCallback to register multiple observers.
	stateChangeCallbacks []func(oldState, newState State)
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(name string, failureThreshold int, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:              name,
		failureThreshold:  failureThreshold,
		successThreshold:  successThreshold,
		timeout:           timeout,
		state:             StateClosed,
		failureCount:      0,
		successCount:      0,
		stateChangeCallbacks: nil,
	}
}

// SetStateChangeCallback registers a callback for state transitions.
func (cb *CircuitBreaker) SetStateChangeCallback(fn func(oldState, newState State)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	// Backwards-compatible: replace all callbacks with a single one.
	cb.stateChangeCallbacks = []func(oldState, newState State){fn}
}

// AddStateChangeCallback appends an observer for state transitions.
func (cb *CircuitBreaker) AddStateChangeCallback(fn func(oldState, newState State)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.stateChangeCallbacks = append(cb.stateChangeCallbacks, fn)
}

// Call executes fn if the circuit is closed or half-open.
// On success, it records a success; on error, it records a failure.
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Auto-transition from Open to Half-Open after timeout.
	if cb.state == StateOpen && time.Since(cb.openedAt) > cb.timeout {
		cb.transitionTo(StateHalfOpen)
	}

	// Reject if still open.
	if cb.state == StateOpen {
		return fmt.Errorf("circuit breaker %q is open", cb.name)
	}

	// Execute the call.
	err := fn()

	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns current statistics.
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return map[string]interface{}{
		"name":              cb.name,
		"state":             string(cb.state),
		"failure_count":     cb.failureCount,
		"success_count":     cb.successCount,
		"last_failure_at":   cb.lastFailureAt.Unix(),
		"opened_at":         cb.openedAt.Unix(),
		"time_until_retry":  cb.timeUntilRetry(),
	}
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionTo(StateClosed)
	cb.failureCount = 0
	cb.successCount = 0
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	cb.lastFailureAt = time.Now()

	if cb.failureCount >= cb.failureThreshold {
		cb.transitionTo(StateOpen)
		cb.openedAt = time.Now()
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == StateClosed {
		cb.successCount = 0
		return
	}

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.transitionTo(StateClosed)
			cb.failureCount = 0
		}
	}
}

func (cb *CircuitBreaker) transitionTo(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState

	for _, fn := range cb.stateChangeCallbacks {
		// Call callbacks synchronously to ensure observers see transitions in-order.
		fn(oldState, newState)
	}
}

func (cb *CircuitBreaker) timeUntilRetry() int64 {
	if cb.state != StateOpen {
		return 0
	}
	nextRetryTime := cb.openedAt.Add(cb.timeout)
	remaining := time.Until(nextRetryTime)
	if remaining < 0 {
		return 0
	}
	return remaining.Milliseconds()
}
