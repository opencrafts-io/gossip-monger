package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationEventBus provides a type-safe API for notification events.
type NotificationEventBus struct {
	bus                      EventBus
	logger                   *slog.Logger
	pool                     *pgxpool.Pool
	notificationEventHandler *NotificationEventHandler
}

// NewUserEventBus creates a new UserEventBus instance.
func NewNotificationEventBus(bus EventBus, pool *pgxpool.Pool, logger *slog.Logger) *NotificationEventBus {

	notificationEventHandler := NewNotificationEventHandler(pool, logger)

	b := &NotificationEventBus{
		bus:                      bus,
		pool:                     pool,
		logger:                   logger,
		notificationEventHandler: notificationEventHandler,
	}

	return b
}

// SetupEventSubscriptions sets up all event subscriptions for the application
func (b *NotificationEventBus) SetupEventSubscriptions(ctx context.Context) error {
	// Subscribe to user created events
	if err := b.SubscribePushNotificationRequested(
		ctx,
		b.notificationEventHandler.HandlerPushNotificationSendRequested,
	); err != nil {
		return fmt.Errorf("failed to subscribe to notification push  events: %w", err)
	}
	b.logger.Info("Gossip notification subscriptions set up successfully")
	return nil
}

// SubscribePushNotificationRequested listens for services requesting push notifications
// to users
// It handles unmarshaling and passes a typed struct to the handler.
func (b *NotificationEventBus) SubscribePushNotificationRequested(ctx context.Context,
	handler func(context context.Context, event NotificationEvent),
) error {
	// Use the same routing key as the publisher
	routingKey := "gossip-monger.notification.requested"

	// Subscribe to the generic event bus, and wrap the handler to unmarshal the data
	return b.bus.Subscribe(routingKey, func(data []byte) {
		var event NotificationEvent
		if err := json.Unmarshal(data, &event); err != nil {
			// Log the error but don't stop the consumer
			// A dead-letter queue or logging service could be used here
			return
		}

		if event.Metadata.EventType != "notification.requested" {
			b.logger.Error(fmt.Sprintf(
				"Wrong metadata event type expected notification.requested instead got %s",
				event.Metadata.EventType,
			),
				slog.String("requested", "notification.requested"),
				// slog.String("hello", event.User.ID.String()),
				/// Log that the operation was aborted by the service to maintain consistency
				slog.Bool("abort", true),
			)
			return
		}
		handler(ctx, event)
	})
}
