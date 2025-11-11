package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserEventBus provides a type-safe API for user events.
type UserEventBus struct {
	bus              EventBus
	logger           *slog.Logger
	pool             *pgxpool.Pool
	userEventHandler *UserEventHandler
}

// NewUserEventBus creates a new UserEventBus instance.
func NewUserEventBus(bus EventBus, pool *pgxpool.Pool, logger *slog.Logger) *UserEventBus {

	userEventHandler := NewUserEventHandler(pool, logger)

	b := &UserEventBus{
		bus:              bus,
		pool:             pool,
		logger:           logger,
		userEventHandler: userEventHandler,
	}

	return b
}

// SetupEventSubscriptions sets up all event subscriptions for the application
func (b *UserEventBus) SetupEventSubscriptions(ctx context.Context) error {
	if err := b.SubscribeForUserEvents(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to user events: %w", err)
	}

	b.logger.Info("Verisafe user event subscriptions set up successfully")
	return nil
}

func (b *UserEventBus) SubscribeForUserEvents(ctx context.Context) error {
	routingKey := "verisafe.user.events"
	return b.bus.Subscribe(routingKey, func(data []byte) {
		var event UserEvent
		if err := json.Unmarshal(data, &event); err != nil {
			// Log the error but don't stop the consumer
			// A dead-letter queue or logging service could be used here
			return
		}
		if event.Metadata.SourceServiceID != "io.opencrafts.verisafe" {
			b.logger.Error(fmt.Sprintf(
				"Wrong metadata event service source id expected 'io.opencrafts.verisafe' instead got %s",
				event.Metadata.SourceServiceID,
			),
				slog.String("requested", "user.created"),
				slog.String("user_id", event.User.ID.String()),
				/// Log that the operation was aborted by the service to maintain consistency
				slog.Bool("abort", true),
			)
			return
		}

		switch event.Metadata.EventType {
		case "user.created":
			b.userEventHandler.HandleUserCreated(ctx, event)
			break

		case "user.updated":
			b.userEventHandler.HandleUserUpdated(ctx, event)
			break

		case "user.deleted":
			b.userEventHandler.HandleUserDeleted(ctx, event)
			break
		default:
			b.logger.Error(fmt.Sprintf(
				"Wrong metadata event type expected user.created instead got %s",
				event.Metadata.EventType,
			),
				slog.String("requested", "user.created"),
				slog.String("user_id", event.User.ID.String()),
				/// Log that the operation was aborted by the service to maintain consistency
				slog.Bool("abort", true),
			)
			return

		}

		b.logger.Info("User event handled successfully", slog.Any("event", event))
	})
}
