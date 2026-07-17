package resilience

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/sony/gobreaker/v2"
)

// Breaker is the minimal surface this package depends on, satisfied by
// *gobreaker.CircuitBreaker[T]. Defined as an interface so callers can
// inject a fake in tests to force the open-state path.
type Breaker[T any] interface {
	Execute(req func() (T, error)) (T, error)
}

// Settings configures a Breaker's failure threshold and recovery timing.
type Settings struct {
	// ConsecutiveFailures is how many consecutive failures while closed
	// trip the breaker open.
	ConsecutiveFailures uint32
	// OpenTimeout is how long the breaker stays open before allowing a
	// half-open trial request through.
	OpenTimeout time.Duration
	// HalfOpenMaxRequests is how many trial requests are allowed through
	// while half-open before deciding to close or re-open.
	HalfOpenMaxRequests uint32
}

// New builds a named circuit breaker. State transitions are logged through
// logger: opening at Warn (something is actually down), closing/half-open
// trials at Info.
func New[T any](name string, settings Settings, logger *slog.Logger) Breaker[T] {
	return gobreaker.NewCircuitBreaker[T](gobreaker.Settings{
		Name:        name,
		MaxRequests: settings.HalfOpenMaxRequests,
		Timeout:     settings.OpenTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= settings.ConsecutiveFailures
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			level := slog.LevelInfo
			if to == gobreaker.StateOpen {
				level = slog.LevelWarn
			}
			logger.Log(context.Background(), level,
				"circuit breaker state change",
				slog.String("breaker", name),
				slog.String("from", from.String()),
				slog.String("to", to.String()),
			)
		},
	})
}

// Open reports whether err indicates the breaker rejected the call outright
// (open, or half-open with its trial quota exhausted) rather than the
// underlying call itself having failed.
func Open(err error) bool {
	return errors.Is(err, gobreaker.ErrOpenState) ||
		errors.Is(err, gobreaker.ErrTooManyRequests)
}
