package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/resend/resend-go/v2"
)

type EmailEventBus struct {
	bus          EventBus
	logger       *slog.Logger
	pool         *pgxpool.Pool
	client       *resend.Client
	emailHandler *EmailEventHandler
}

// Returns a new EmailEventBus for performing email related tasks
func NewEmailEventBus(
	bus EventBus, pool *pgxpool.Pool,
	client *resend.Client,
	logger *slog.Logger,
) *EmailEventBus {
	return &EmailEventBus{
		bus:          bus,
		pool:         pool,
		client:       client,
		logger:       logger,
		emailHandler: NewEmailEventHandler(pool, client, logger),
	}
}

func (b *EmailEventBus) SetupEventSubscriptions(ctx context.Context) error {
	if err := b.SubscribeEmailSendRequested(ctx, b.emailHandler.HandleEmailSendRequested); err != nil {
		return fmt.Errorf("Failed to subscribe to email sending event %w", err)
	}

	b.logger.Info("Gossip email subscriptions set up successfully")

	return nil
}

func (b *EmailEventBus) SubscribeEmailSendRequested(
	ctx context.Context,
	handler func(context context.Context, event EmailEvent),
) error {
	// Use the same routing key as the publisher
	routingKey := "gossip-monger.email"

	// Subscribe to the generic event bus, and wrap the handler to unmarshal the data
	return b.bus.Subscribe(routingKey, func(data []byte) {
		var event EmailEvent
		if err := json.Unmarshal(data, &event); err != nil {
			// Log the error but don't stop the consumer
			// A dead-letter queue or logging service could be used here
			b.logger.Error("Error unmarshalling data", slog.Any("error", err))
			return
		}

		if event.Metadata.EventType != "email.send" {
			b.logger.Error(fmt.Sprintf(
				"Wrong metadata event type expected email.send instead got %s",
				event.Metadata.EventType,
			),
				slog.String("requested", "email.send"),
				/// Log that the operation was aborted by the service to maintain consistency
				slog.Bool("abort", true),
			)
			return
		}
		handler(ctx, event)
	})

}
