package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/OneSignal/onesignal-go-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	pushNotificationRoutingKey    = "gossip-monger.notification.requested"
	expectedNotificationEventType = "notification.requested"
)

// NotificationEventBus provides a type-safe API for notification events.
type NotificationEventBus struct {
	bus                      EventBus
	logger                   *slog.Logger
	notificationEventHandler *NotificationEventHandler

	// ctx is a long-lived context owned by NotificationEventBus, used by
	// subscriber handlers. It is independent of any caller context so that
	// handlers remain active for the lifetime of the bus.
	ctx    context.Context
	cancel context.CancelFunc
}

// NewNotificationEventBus creates a new NotificationEventBus instance.
func NewNotificationEventBus(
	bus EventBus,
	pool *pgxpool.Pool,
	onesignalClient *onesignal.APIClient,
	logger *slog.Logger,
) *NotificationEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &NotificationEventBus{
		bus:                      bus,
		logger:                   logger,
		notificationEventHandler: NewNotificationEventHandler(pool, onesignalClient, logger),
		ctx:                      ctx,
		cancel:                   cancel,
	}
}

// SetupEventSubscriptions registers all event subscriptions for the application.
func (b *NotificationEventBus) SetupEventSubscriptions() error {
	if err := b.subscribePushNotificationRequested(); err != nil {
		return fmt.Errorf("failed to subscribe to notification push events: %w", err)
	}
	b.logger.Info("notification event subscriptions set up successfully")
	return nil
}

// Close cancels the internal context, signalling all active handlers to stop.
func (b *NotificationEventBus) Close() {
	b.cancel()
}

// subscribePushNotificationRequested listens for services requesting push
// notifications to be sent to users.
func (b *NotificationEventBus) subscribePushNotificationRequested() error {
	return b.bus.Subscribe(pushNotificationRoutingKey, func(data []byte) {
		var event NotificationEvent
		if err := json.Unmarshal(data, &event); err != nil {
			b.logger.Error("failed to unmarshal notification event",
				slog.String("routing_key", pushNotificationRoutingKey),
				slog.Any("error", err),
			)
			return
		}

		if event.Metadata.EventType != expectedNotificationEventType {
			b.logger.Error("notification event rejected: unexpected event type",
				slog.String("routing_key", pushNotificationRoutingKey),
				slog.String("expected_event_type", expectedNotificationEventType),
				slog.String("actual_event_type", event.Metadata.EventType),
				slog.Bool("abort", true),
			)
			return
		}

		b.notificationEventHandler.HandlerPushNotificationSendRequested(b.ctx, event)

		b.logger.Info("notification event handled successfully",
			slog.String("event_type", event.Metadata.EventType),
		)
	})
}

