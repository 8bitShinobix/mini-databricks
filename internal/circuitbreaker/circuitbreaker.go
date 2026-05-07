package circuitbreaker

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/sony/gobreaker"
)

func New(name string) *gobreaker.CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,                // allow 1 request in half-open state
		Interval:    30 * time.Second, // reset failure count every 30s
		Timeout:     30 * time.Second, // stay open for 30s before half-open
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// open circuit after 3 consecutive failures
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("circuit breaker state changed",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	}
	return gobreaker.NewCircuitBreaker(settings)
}

// Execute wraps a function call with circuit breaker protection
func Execute[T any](cb *gobreaker.CircuitBreaker, fn func() (T, error)) (T, error) {
	result, err := cb.Execute(func() (any, error) {
		return fn()
	})
	if err != nil {
		var zero T
		return zero, fmt.Errorf("circuit breaker [%s]: %w", cb.Name(), err)
	}
	return result.(T), nil
}
