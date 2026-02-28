package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/resend/resend-go/v2"
)

const (
	emailRoutingKey        = "gossip-monger.email"
	expectedEmailEventType = "email.send"
)

// EmailEventBus provides a type-safe API for email events.
type EmailEventBus struct {
	bus          EventBus
	logger       *slog.Logger
	emailHandler *EmailEventHandler

	// ctx is a long-lived context owned by EmailEventBus, used by subscriber
	// handlers. It is independent of any caller context so that handlers
	// remain active for the lifetime of the bus.
	ctx    context.Context
	cancel context.CancelFunc
}

// NewEmailEventBus creates a new EmailEventBus for performing email related tasks.
func NewEmailEventBus(
	bus EventBus,
	pool *pgxpool.Pool,
	client *resend.Client,
	logger *slog.Logger,
) *EmailEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &EmailEventBus{
		bus:          bus,
		logger:       logger,
		emailHandler: NewEmailEventHandler(pool, client, logger),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// SetupEventSubscriptions registers all event subscriptions for the application.
func (b *EmailEventBus) SetupEventSubscriptions() error {
	if err := b.subscribeEmailSendRequested(); err != nil {
		return fmt.Errorf("failed to subscribe to email send events: %w", err)
	}
	b.logger.Info("email event subscriptions set up successfully")
	return nil
}

// Close cancels the internal context, signalling all active handlers to stop.
func (b *EmailEventBus) Close() {
	b.cancel()
}

// subscribeEmailSendRequested listens for services requesting emails to be sent.
func (b *EmailEventBus) subscribeEmailSendRequested() error {
	return b.bus.Subscribe(emailRoutingKey, func(data []byte) {
		var event EmailEvent
		if err := json.Unmarshal(data, &event); err != nil {
			b.logger.Error("failed to unmarshal email event",
				slog.String("routing_key", emailRoutingKey),
				slog.Any("error", err),
			)
			return
		}

		if event.Metadata.EventType != expectedEmailEventType {
			b.logger.Error("email event rejected: unexpected event type",
				slog.String("routing_key", emailRoutingKey),
				slog.String("expected_event_type", expectedEmailEventType),
				slog.String("actual_event_type", event.Metadata.EventType),
				slog.Bool("abort", true),
			)
			return
		}

		b.emailHandler.HandleEmailSendRequested(b.ctx, event)

		b.logger.Info("email event handled successfully",
			slog.String("event_type", event.Metadata.EventType),
		)
	})
}

