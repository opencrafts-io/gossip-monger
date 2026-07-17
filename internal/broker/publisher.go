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
	// PublishRaw publishes body as-is, skipping JSON marshaling. Used to
	// forward an already-serialized message body verbatim, e.g. re-publishing
	// a dead-lettered delivery's raw bytes to the parked queue.
	PublishRaw(
		ctx context.Context,
		exchange, routingKey string,
		body []byte,
	) error
}

// Publisher implements MessagePublisher
type Publisher struct {
	conn   Connection
	logger *slog.Logger
}

// NewPublisher creates a new Publisher instance
func NewPublisher(conn Connection, logger *slog.Logger) *Publisher {
	return &Publisher{
		conn:   conn,
		logger: logger,
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
		p.logger.Error("failed to marshal message", "error", err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ch, err := p.conn.Channel()
	if err != nil {
		p.logger.Error("cannot publish message", "error", err)
		return err
	}
	defer ch.Close()

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
		p.logger.Error("failed to publish message",
			"exchange", exchange,
			"routing_key", routingKey,
			"error", err,
		)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.Debug("message published",
		"exchange", exchange,
		"routing_key", routingKey,
	)
	return nil
}

// PublishRaw publishes body as-is, skipping JSON marshaling.
func (p *Publisher) PublishRaw(
	ctx context.Context,
	exchange, routingKey string,
	body []byte,
) error {
	ch, err := p.conn.Channel()
	if err != nil {
		p.logger.Error("cannot publish message", "error", err)
		return err
	}
	defer ch.Close()

	err = ch.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		p.logger.Error("failed to publish raw message",
			"exchange", exchange,
			"routing_key", routingKey,
			"error", err,
		)
		return fmt.Errorf("failed to publish raw message: %w", err)
	}

	p.logger.Debug("raw message published",
		"exchange", exchange,
		"routing_key", routingKey,
	)
	return nil
}
