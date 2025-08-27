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
	// Subscribe to user created events
	if err := b.SubscribeUserCreated(ctx, b.userEventHandler.HandleUserCreated); err != nil {
		return fmt.Errorf("failed to subscribe to user created events: %w", err)
	}

	// Subscribe to user updated events
	if err := b.SubscribeUserUpdated(ctx, b.userEventHandler.HandleUserUpdated); err != nil {
		return fmt.Errorf("failed to subscribe to user updated events: %w", err)
	}

	// Subscribe to user deleted events
	if err := b.SubscribeUserDeleted(ctx, b.userEventHandler.HandleUserDeleted); err != nil {
		return fmt.Errorf("failed to subscribe to user deleted events: %w", err)
	}

	b.logger.Info("Verisafe user event subscriptions set up successfully")
	return nil
}

// SubscribeUserCreated listens for user creation events.
// It handles unmarshaling and passes a typed struct to the handler.
func (b *UserEventBus) SubscribeUserCreated(ctx context.Context,
	handler func(context context.Context, event UserEvent),
) error {
	// Use the same routing key as the publisher
	routingKey := "verisafe.user.created"

	// Subscribe to the generic event bus, and wrap the handler to unmarshal the data
	return b.bus.Subscribe(routingKey, func(data []byte) {
		var event UserEvent
		if err := json.Unmarshal(data, &event); err != nil {
			// Log the error but don't stop the consumer
			// A dead-letter queue or logging service could be used here
			return
		}
		b.logger.Info("User created", slog.Any("user", event))
		handler(ctx, event)
	})
}

// SubscribeUserUpdated listens for user updated events.
// It handles unmarshaling and passes a typed struct to the handler.
func (b *UserEventBus) SubscribeUserUpdated(
	ctx context.Context,
	handler func(ctx context.Context, event UserEvent),
) error {
	// Use the same routing key as the publisher
	routingKey := "verisafe.user.updated"

	// Subscribe to the generic event bus, and wrap the handler to unmarshal the data
	return b.bus.Subscribe(routingKey, func(data []byte) {
		var event UserEvent
		if err := json.Unmarshal(data, &event); err != nil {
			// Log the error but don't stop the consumer
			// A dead-letter queue or logging service could be used here
			return
		}
		handler(ctx, event)
	})
}

// SubscribeUserDeleted listens for user deletion events.
// It handles unmarshaling and passes a typed struct to the handler.
func (b *UserEventBus) SubscribeUserDeleted(
	ctx context.Context,
	handler func(ctx context.Context, event UserEvent),
) error {
	// Use the same routing key as the publisher
	routingKey := "verisafe.user.deleted"

	// Subscribe to the generic event bus, and wrap the handler to unmarshal the data
	return b.bus.Subscribe(routingKey, func(data []byte) {
		var event UserEvent
		if err := json.Unmarshal(data, &event); err != nil {
			// Log the error but don't stop the consumer
			// A dead-letter queue or logging service could be used here
			return
		}
		handler(ctx, event)
	})
}

// PublishUserCreated publishes a user created event to the event bus
func (b *UserEventBus) PublishUserCreated(ctx context.Context, event UserEvent) error {
	routingKey := "verisafe.user.created"
	b.logger.Info("Publishing user created event", slog.String("routing_key", routingKey))
	return b.bus.Publish(ctx, routingKey, event)
}

// PublishUserUpdated publishes a user updated event to the event bus
func (b *UserEventBus) PublishUserUpdated(ctx context.Context, event UserEvent) error {
	routingKey := "verisafe.user.updated"
	b.logger.Info("Publishing user updated event", slog.String("routing_key", routingKey))
	return b.bus.Publish(ctx, routingKey, event)
}

// PublishUserDeleted publishes a user deleted event to the event bus
func (b *UserEventBus) PublishUserDeleted(ctx context.Context, event UserEvent) error {
	routingKey := "verisafe.user.deleted"
	b.logger.Info("Publishing user deleted event", slog.String("routing_key", routingKey))
	return b.bus.Publish(ctx, routingKey, event)
}
