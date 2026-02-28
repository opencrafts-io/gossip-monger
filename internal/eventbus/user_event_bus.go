package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

const userEventsRoutingKey = "verisafe.user.events"
const expectedSourceServiceID = "io.opencrafts.verisafe"

// UserEventBus provides a type-safe API for user events.
type UserEventBus struct {
	bus              EventBus
	logger           *slog.Logger
	pool             *pgxpool.Pool
	userEventHandler *UserEventHandler

	// ctx is a long-lived context owned by UserEventBus, used by subscriber
	// handlers. It is independent of any caller context so that subscription
	// handlers remain active for the lifetime of the bus.
	ctx    context.Context
	cancel context.CancelFunc
}

// NewUserEventBus creates a new UserEventBus instance.
func NewUserEventBus(bus EventBus, pool *pgxpool.Pool, logger *slog.Logger) *UserEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &UserEventBus{
		bus:              bus,
		pool:             pool,
		logger:           logger,
		userEventHandler: NewUserEventHandler(pool, logger),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// SetupEventSubscriptions registers all event subscriptions for the specific event bus.
func (b *UserEventBus) SetupEventSubscriptions() error {
	if err := b.subscribeForUserEvents(); err != nil {
		return fmt.Errorf("failed to subscribe to user events: %w", err)
	}
	b.logger.Info("user event subscriptions set up successfully")
	return nil
}

// Close cancels the internal context, signalling all active handlers to stop.
func (b *UserEventBus) Close() {
	b.cancel()
}

func (b *UserEventBus) subscribeForUserEvents() error {
	return b.bus.Subscribe(userEventsRoutingKey, func(data []byte) {
		var event UserEvent
		if err := json.Unmarshal(data, &event); err != nil {
			b.logger.Error("failed to unmarshal user event",
				slog.String("routing_key", userEventsRoutingKey),
				slog.Any("error", err),
			)
			return
		}

		if event.Metadata.SourceServiceID != expectedSourceServiceID {
			b.logger.Error("user event rejected: unexpected source service ID",
				slog.String("routing_key", userEventsRoutingKey),
				slog.String("expected_source", expectedSourceServiceID),
				slog.String("actual_source", event.Metadata.SourceServiceID),
				slog.String("user_id", event.User.ID.String()),
				slog.Bool("abort", true),
			)
			return
		}

		switch event.Metadata.EventType {
		case "user.created":
			b.userEventHandler.HandleUserCreated(b.ctx, event)
		case "user.updated":
			b.userEventHandler.HandleUserUpdated(b.ctx, event)
		case "user.deleted":
			b.userEventHandler.HandleUserDeleted(b.ctx, event)
		default:
			b.logger.Error("user event rejected: unrecognised event type",
				slog.String("routing_key", userEventsRoutingKey),
				slog.String("event_type", event.Metadata.EventType),
				slog.String("user_id", event.User.ID.String()),
				slog.Bool("abort", true),
			)
			return
		}

		b.logger.Info("user event handled successfully",
			slog.String("event_type", event.Metadata.EventType),
			slog.String("user_id", event.User.ID.String()),
		)
	})
}
