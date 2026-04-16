package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

// MessagePublisher defines the contract for publishing messages to RabbitMQ
type MessagePublisher interface {
	Publish(
		ctx context.Context,
		exchange, routingKey string,
		message any,
	) error
}

// Publisher implements MessagePublisher
type Publisher struct {
	conn Connection
}

// NewPublisher creates a new Publisher instance
func NewPublisher(conn Connection) *Publisher {
	return &Publisher{
		conn: conn,
	}
}

// Publish marshals the message to JSON and publishes it to the exchange
func (p *Publisher) Publish(
	ctx context.Context,
	exchange, routingKey string,
	message any,
) error {
	// Marshal the message to JSON
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		slog.Error("failed to marshal message", "error", err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ch := p.conn.Channel()
	if ch == nil {
		err := fmt.Errorf("channel is nil")
		slog.Error("cannot publish message", "error", err)
		return err
	}

	// Publish the message
	err = ch.PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonBytes,
		},
	)
	if err != nil {
		slog.Error("failed to publish message",
			"exchange", exchange,
			"routing_key", routingKey,
			"error", err,
		)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	slog.Debug("message published",
		"exchange", exchange,
		"routing_key", routingKey,
	)
	return nil
}
