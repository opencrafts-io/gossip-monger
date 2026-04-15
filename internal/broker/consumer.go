package broker

import (
	"context"
	"fmt"
	"log/slog"
)

// MessageHandler is the callback function that processes consumed messages
type MessageHandler func(ctx context.Context, message []byte) error

// MessageConsumer defines the contract for consuming messages from RabbitMQ
type MessageConsumer interface {
	Consume(ctx context.Context, queue string, handler MessageHandler) error
}

// Consumer implements MessageConsumer
type Consumer struct {
	conn Connection
}

// NewConsumer creates a new Consumer instance
func NewConsumer(conn Connection) *Consumer {
	return &Consumer{
		conn: conn,
	}
}

// Consume starts consuming messages from a queue and passes them to the handler
func (c *Consumer) Consume(
	ctx context.Context,
	queue string,
	handler MessageHandler,
) error {
	ch := c.conn.Channel()
	if ch == nil {
		err := fmt.Errorf("channel is nil")
		slog.Error("cannot consume messages", "error", err)
		return err
	}

	// Start consuming messages
	msgs, err := ch.Consume(
		queue, // queue name
		"",    // consumer tag (auto-generated)
		true,  // auto-acknowledge
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		slog.Error("failed to start consuming",
			"queue", queue,
			"error", err,
		)
		return fmt.Errorf("failed to consume from queue: %w", err)
	}

	slog.Info("consumer started", "queue", queue)

	// Listen for messages
	for {
		select {
		case <-ctx.Done():
			slog.Info("consumer stopped", "queue", queue)
			return ctx.Err()
		case msg := <-msgs:
			if msg.Body == nil {
				err := fmt.Errorf("channel closed")
				slog.Error("consuming failed", "queue", queue, "error", err)
				return err
			}

			// Call the handler with the message
			if err := handler(ctx, msg.Body); err != nil {
				slog.Error("handler error",
					"queue", queue,
					"error", err,
				)
				// Continue consuming, don't stop on handler error
			}
		}
	}
}
