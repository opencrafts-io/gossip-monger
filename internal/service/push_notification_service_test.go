package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/opencrafts-io/gossip-monger/internal/repository"
	"github.com/opencrafts-io/gossip-monger/internal/resilience"
	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeQuerier embeds the (large) repository.Querier interface as a nil
// value so tests only need to implement the one method they exercise;
// calling any other method panics on the nil embedded interface, which is
// the point — it surfaces an unexpected call immediately.
type fakeQuerier struct {
	repository.Querier
	upsertNotification        func(ctx context.Context, arg repository.UpsertNotificationParams) (repository.Notification, error)
	getNotificationByQueueMsg func(ctx context.Context, id *string) (repository.Notification, error)
}

func (f *fakeQuerier) UpsertNotification(
	ctx context.Context,
	arg repository.UpsertNotificationParams,
) (repository.Notification, error) {
	return f.upsertNotification(ctx, arg)
}

func (f *fakeQuerier) GetNotificationByQueueMessageID(
	ctx context.Context,
	id *string,
) (repository.Notification, error) {
	if f.getNotificationByQueueMsg != nil {
		return f.getNotificationByQueueMsg(ctx, id)
	}
	// Default: no existing row, i.e. not a duplicate.
	return repository.Notification{}, pgx.ErrNoRows
}

// fakeBreaker lets a test force the breaker-open path without waiting on
// real consecutive failures/timeouts, and tracks whether it was invoked at
// all so a test can assert the provider was never reached.
type fakeBreaker[T any] struct {
	forcedErr error
	calls     *int
}

func (f fakeBreaker[T]) Execute(req func() (T, error)) (T, error) {
	if f.calls != nil {
		*f.calls++
	}
	if f.forcedErr != nil {
		var zero T
		return zero, f.forcedErr
	}
	return req()
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func validPushNotification() repository.Notification {
	return repository.Notification{
		Headings:         json.RawMessage(`{"en":"hi"}`),
		Contents:         json.RawMessage(`{"en":"body"}`),
		IncludedSegments: []string{"Active Users"},
	}
}

func TestSend_BreakerOpen_RecordsCircuitOpenAndReturnsError(t *testing.T) {
	var captured repository.UpsertNotificationParams
	repo := &fakeQuerier{
		upsertNotification: func(_ context.Context, arg repository.UpsertNotificationParams) (repository.Notification, error) {
			captured = arg
			return repository.Notification{}, nil
		},
	}

	pns := &pushNotificationService{
		repo:    repo,
		logger:  testLogger(),
		breaker: fakeBreaker[*onesignalCallResult]{forcedErr: gobreaker.ErrOpenState},
	}

	err := pns.Send(context.Background(), validPushNotification(), "req-123")

	require.Error(t, err)
	assert.True(t, resilience.Open(err))
	require.NotNil(t, captured.Status)
	assert.Equal(t, "circuit_open", *captured.Status)
	require.NotNil(t, captured.QueueMessageID)
	assert.Equal(t, "req-123", *captured.QueueMessageID)
}

func TestSend_ValidationError_PersistsFailedStatusWithoutCallingProvider(t *testing.T) {
	var captured repository.UpsertNotificationParams
	calls := 0
	repo := &fakeQuerier{
		upsertNotification: func(_ context.Context, arg repository.UpsertNotificationParams) (repository.Notification, error) {
			captured = arg
			return repository.Notification{}, nil
		},
	}

	pns := &pushNotificationService{
		repo:    repo,
		logger:  testLogger(),
		breaker: fakeBreaker[*onesignalCallResult]{calls: &calls},
	}

	// No targeting mechanism specified at all -> preparePushPayload fails
	// before the breaker/provider is ever reached.
	invalid := repository.Notification{
		Headings: json.RawMessage(`{"en":"hi"}`),
		Contents: json.RawMessage(`{"en":"body"}`),
	}

	err := pns.Send(context.Background(), invalid, "req-456")

	require.Error(t, err)
	assert.Equal(t, 0, calls, "breaker/provider must not be invoked when payload validation fails")
	require.NotNil(t, captured.Status)
	assert.Equal(t, "failed", *captured.Status)
	require.NotNil(t, captured.QueueMessageID)
	assert.Equal(t, "req-456", *captured.QueueMessageID)
}

func TestSend_QueueMessageIDThreadedThroughOnEveryAttempt(t *testing.T) {
	var seen []string
	repo := &fakeQuerier{
		upsertNotification: func(_ context.Context, arg repository.UpsertNotificationParams) (repository.Notification, error) {
			if arg.QueueMessageID != nil {
				seen = append(seen, *arg.QueueMessageID)
			}
			return repository.Notification{}, nil
		},
	}

	pns := &pushNotificationService{
		repo:    repo,
		logger:  testLogger(),
		breaker: fakeBreaker[*onesignalCallResult]{forcedErr: gobreaker.ErrOpenState},
	}

	// Same queueMessageID sent twice, simulating a dead-lettered redelivery
	// of the same logical message. Both attempts must record against the
	// same id so the DB-level upsert (ON CONFLICT (queue_message_id)) dedupes
	// them into one row instead of erroring on a second insert.
	_ = pns.Send(context.Background(), validPushNotification(), "req-same")
	_ = pns.Send(context.Background(), validPushNotification(), "req-same")

	require.Len(t, seen, 2)
	assert.Equal(t, "req-same", seen[0])
	assert.Equal(t, "req-same", seen[1])
}

func TestSend_DuplicateAlreadySent_SkipsResendWithoutCallingProvider(t *testing.T) {
	sentStatus := "sent"
	calls := 0
	repo := &fakeQuerier{
		getNotificationByQueueMsg: func(_ context.Context, id *string) (repository.Notification, error) {
			return repository.Notification{Status: &sentStatus}, nil
		},
	}

	pns := &pushNotificationService{
		repo:    repo,
		logger:  testLogger(),
		breaker: fakeBreaker[*onesignalCallResult]{calls: &calls},
	}

	err := pns.Send(context.Background(), validPushNotification(), "req-already-sent")

	require.NoError(t, err)
	assert.Equal(t, 0, calls, "an already-sent queue_message_id must not trigger a second real send")
}
