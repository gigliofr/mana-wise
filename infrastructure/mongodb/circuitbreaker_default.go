package mongodb

import "github.com/gigliofr/mana-wise/infrastructure/circuitbreaker"

var defaultCircuitBreaker *circuitbreaker.CircuitBreaker

// SetDefaultCircuitBreaker sets a package-level circuit breaker used by repositories.
func SetDefaultCircuitBreaker(cb *circuitbreaker.CircuitBreaker) {
	defaultCircuitBreaker = cb
}
