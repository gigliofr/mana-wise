package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half-open"
)

// ErrOpen is matched (errors.Is) by the error returned when a call is rejected
// because the circuit is open, or because a half-open probe is already in flight.
var ErrOpen = errors.New("circuit breaker is open")

// OpenError is the rejection error; its message keeps the historical format.
type OpenError struct{ Name string }

func (e *OpenError) Error() string        { return fmt.Sprintf("circuit breaker %q is open", e.Name) }
func (e *OpenError) Is(target error) bool { return target == ErrOpen }

// CircuitBreaker implements the circuit breaker pattern.
//
// The protected function runs outside the internal lock, so concurrent calls
// are not serialized. The circuit opens after failureThreshold consecutive
// failures; after timeout it lets a single probe through (half-open) and closes
// again after successThreshold successful probes.
type CircuitBreaker struct {
	name string
	mu   sync.Mutex

	failureThreshold int
	successThreshold int
	timeout          time.Duration
	isFailure        func(error) bool

	state         State
	failureCount  int // consecutive failures while closed
	successCount  int // successful probes while half-open
	probeInFlight bool
	lastFailureAt time.Time
	openedAt      time.Time

	callbacksMu          sync.RWMutex
	stateChangeCallbacks []func(oldState, newState State)
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(name string, failureThreshold int, successThreshold int, timeout time.Duration) *CircuitBreaker {
	if failureThreshold < 1 {
		failureThreshold = 1
	}
	if successThreshold < 1 {
		successThreshold = 1
	}
	return &CircuitBreaker{
		name:             name,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		state:            StateClosed,
		isFailure:        DefaultIsFailure,
	}
}

// DefaultIsFailure counts every error except a cancelled request: a client that
// goes away says nothing about the health of the dependency.
func DefaultIsFailure(err error) bool {
	return err != nil && !errors.Is(err, context.Canceled)
}

// SetFailurePredicate sets which errors count as dependency failures (for
// example, to ignore "duplicate key" or validation errors).
func (cb *CircuitBreaker) SetFailurePredicate(fn func(error) bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if fn == nil {
		fn = DefaultIsFailure
	}
	cb.isFailure = fn
}

// SetStateChangeCallback registers a single callback, replacing existing ones.
func (cb *CircuitBreaker) SetStateChangeCallback(fn func(oldState, newState State)) {
	cb.callbacksMu.Lock()
	defer cb.callbacksMu.Unlock()
	cb.stateChangeCallbacks = []func(oldState, newState State){fn}
}

// AddStateChangeCallback appends an observer for state transitions.
// Callbacks run synchronously, in order, outside the breaker's lock.
func (cb *CircuitBreaker) AddStateChangeCallback(fn func(oldState, newState State)) {
	cb.callbacksMu.Lock()
	defer cb.callbacksMu.Unlock()
	cb.stateChangeCallbacks = append(cb.stateChangeCallbacks, fn)
}

type transition struct{ from, to State }

// Call executes fn unless the circuit is open. Errors from fn are returned
// unchanged; a rejection returns an error wrapping ErrOpen.
func (cb *CircuitBreaker) Call(fn func() error) error {
	probe, transitions, err := cb.before()
	cb.notify(transitions)
	if err != nil {
		return err
	}

	callErr := fn()

	cb.notify(cb.after(probe, callErr))
	return callErr
}

func (cb *CircuitBreaker) before() (probe bool, transitions []transition, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && time.Since(cb.openedAt) >= cb.timeout {
		transitions = append(transitions, cb.setState(StateHalfOpen))
		cb.successCount = 0
		cb.probeInFlight = false
	}

	switch cb.state {
	case StateOpen:
		return false, transitions, &OpenError{Name: cb.name}
	case StateHalfOpen:
		if cb.probeInFlight {
			return false, transitions, &OpenError{Name: cb.name}
		}
		cb.probeInFlight = true
		return true, transitions, nil
	default:
		return false, transitions, nil
	}
}

func (cb *CircuitBreaker) after(probe bool, callErr error) []transition {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if probe {
		cb.probeInFlight = false
	}
	failed := cb.isFailure(callErr)

	var transitions []transition
	switch cb.state {
	case StateHalfOpen:
		if failed {
			cb.lastFailureAt = time.Now()
			cb.openedAt = time.Now()
			transitions = append(transitions, cb.setState(StateOpen))
			return transitions
		}
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.failureCount = 0
			cb.successCount = 0
			transitions = append(transitions, cb.setState(StateClosed))
		}
	case StateClosed:
		if failed {
			cb.failureCount++
			cb.lastFailureAt = time.Now()
			if cb.failureCount >= cb.failureThreshold {
				cb.openedAt = time.Now()
				transitions = append(transitions, cb.setState(StateOpen))
			}
			return transitions
		}
		// Only consecutive failures open the circuit.
		cb.failureCount = 0
	}
	return transitions
}

// setState changes the state and returns the transition (from == to when unchanged).
func (cb *CircuitBreaker) setState(s State) transition {
	t := transition{from: cb.state, to: s}
	cb.state = s
	return t
}

func (cb *CircuitBreaker) notify(transitions []transition) {
	if len(transitions) == 0 {
		return
	}
	cb.callbacksMu.RLock()
	callbacks := append([]func(State, State){}, cb.stateChangeCallbacks...)
	cb.callbacksMu.RUnlock()
	for _, t := range transitions {
		if t.from == t.to {
			continue
		}
		for _, fn := range callbacks {
			fn(t.from, t.to)
		}
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Stats returns current statistics.
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return map[string]interface{}{
		"name":             cb.name,
		"state":            string(cb.state),
		"failure_count":    cb.failureCount,
		"success_count":    cb.successCount,
		"last_failure_at":  cb.lastFailureAt.Unix(),
		"opened_at":        cb.openedAt.Unix(),
		"time_until_retry": cb.timeUntilRetry(),
	}
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	t := cb.setState(StateClosed)
	cb.failureCount = 0
	cb.successCount = 0
	cb.probeInFlight = false
	cb.mu.Unlock()
	cb.notify([]transition{t})
}

func (cb *CircuitBreaker) timeUntilRetry() int64 {
	if cb.state != StateOpen {
		return 0
	}
	remaining := time.Until(cb.openedAt.Add(cb.timeout))
	if remaining < 0 {
		return 0
	}
	return remaining.Milliseconds()
}
