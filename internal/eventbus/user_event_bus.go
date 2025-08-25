package eventbus

import (
	"context"
	"encoding/json"
)

// UserEventBus provides a type-safe API for user events.
type UserEventBus struct {
	bus EventBus
}

// NewUserEventBus creates a new UserEventBus instance.
func NewUserEventBus(bus EventBus) *UserEventBus {
	return &UserEventBus{bus: bus}
}

// PublishUserCreated publishes a user creation event.
// It abstracts the RabbitMQ routing key and JSON marshaling.
func (b *UserEventBus) PublishUserCreated(ctx context.Context, userID, username, email string) error {
	event := UserEvent{}

	// Use a specific routing key for user creation events
	routingKey := "user.created"
	return b.bus.Publish(ctx, routingKey, event)
}

// SubscribeUserCreated listens for user creation events.
// It handles unmarshaling and passes a typed struct to the handler.
func (b *UserEventBus) SubscribeUserCreated(handler func(event UserEvent)) error {
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
		handler(event)
	})
}

// SubscribeUserUpdated listens for user updated events.
// It handles unmarshaling and passes a typed struct to the handler.
func (b *UserEventBus) SubscribeUserUpdated(handler func(event UserEvent)) error {
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
		handler(event)
	})
}

// SubscribeUserDeleted listens for user deletion events.
// It handles unmarshaling and passes a typed struct to the handler.
func (b *UserEventBus) SubscribeUserDeleted(handler func(event UserEvent)) error {
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
		handler(event)
	})
}
