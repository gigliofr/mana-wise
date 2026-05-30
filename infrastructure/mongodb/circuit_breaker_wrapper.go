package mongodb

import (
	"log/slog"
	"time"

	"github.com/gigliofr/mana-wise/infrastructure/circuitbreaker"
)

// CircuitBreakerConfig holds configuration for MongoDB circuit breaker.
type CircuitBreakerConfig struct {
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
}

// DefaultCircuitBreakerConfig returns standard MongoDB circuit breaker settings.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,        // Open after 5 failures
		SuccessThreshold: 1,        // Close after 1 success in half-open
		Timeout:          30 * time.Second, // Wait 30s before trying half-open
	}
}

// NewCircuitBreakerForMongoDB creates a circuit breaker for MongoDB operations.
func NewCircuitBreakerForMongoDB(logger *slog.Logger) *circuitbreaker.CircuitBreaker {
	config := DefaultCircuitBreakerConfig()
	cb := circuitbreaker.NewCircuitBreaker(
		"mongodb",
		config.FailureThreshold,
		config.SuccessThreshold,
		config.Timeout,
	)
	return cb
}
