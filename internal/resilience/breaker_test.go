package resilience

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNew_TripsOpenAfterConsecutiveFailures(t *testing.T) {
	b := New[string]("test", Settings{
		ConsecutiveFailures: 3,
		OpenTimeout:         time.Minute,
		HalfOpenMaxRequests: 1,
	}, discardLogger())

	failing := func() (string, error) { return "", errors.New("boom") }

	for i := 0; i < 3; i++ {
		_, err := b.Execute(failing)
		require.Error(t, err)
		assert.False(t, Open(err), "underlying call failures should not be reported as breaker-open")
	}

	// The 3rd consecutive failure should have tripped the breaker open, so
	// this call must be rejected without invoking failing again.
	called := false
	_, err := b.Execute(func() (string, error) {
		called = true
		return "", errors.New("should not run")
	})

	require.Error(t, err)
	assert.False(t, called, "breaker should short-circuit and never invoke the wrapped call")
	assert.True(t, Open(err), "rejection while open should be reported as breaker-open")
	assert.True(t, errors.Is(err, gobreaker.ErrOpenState))
}

func TestOpen_FalseForOrdinaryErrors(t *testing.T) {
	assert.False(t, Open(errors.New("some ordinary failure")))
	assert.False(t, Open(nil))
}

func TestNew_SuccessfulCallsPassThrough(t *testing.T) {
	b := New[int]("test", Settings{
		ConsecutiveFailures: 5,
		OpenTimeout:         time.Minute,
		HalfOpenMaxRequests: 1,
	}, discardLogger())

	v, err := b.Execute(func() (int, error) { return 42, nil })
	require.NoError(t, err)
	assert.Equal(t, 42, v)
}
