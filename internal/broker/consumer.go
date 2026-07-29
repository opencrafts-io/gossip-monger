package broker

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ExchangeType string

const (
	DirectExchangeType ExchangeType = "direct"
	FanoutExchangeType ExchangeType = "fanout"
	TopicExchangeType  ExchangeType = "topic"
)

// MessageHandler is the callback function that processes consumed messages
type MessageHandler func(ctx context.Context, message []byte) error

// MessageConsumer defines the contract for consuming messages from RabbitMQ
type MessageConsumer interface {
	Consume(
		ctx context.Context,
		exchange string,
		exchangeType ExchangeType,
		queue string,
		bindingKey string,
		deadLetterExchange string,
		handler MessageHandler,
	) error
}

// Consumer implements MessageConsumer
type Consumer struct {
	conn             Connection
	prefetchCount    int
	maxRetryAttempts int
	logger           slog.Logger
}

// NewConsumer creates a new Consumer instance with configurable prefetch count
// prefetchCount limits the number of unacknowledged messages delivered to this consumer
// Recommended: 1 for serial processing, higher for parallel processing
//
// maxRetryAttempts only matters for queues declared with a deadLetterExchange
// (see Consume) — it's ignored otherwise.
func NewConsumer(
	conn Connection,
	prefetchCount int,
	maxRetryAttempts int,
	logger slog.Logger,
) *Consumer {
	return &Consumer{
		conn:             conn,
		prefetchCount:    prefetchCount,
		maxRetryAttempts: maxRetryAttempts,
		logger:           logger,
	}
}

// Consume starts consuming messages from a queue and passes them to the
// handler. If deadLetterExchange is non-empty, the queue is declared with
// it as its x-dead-letter-exchange, and a handler error nacks the message
// there (instead of discarding it) for automatic delayed redelivery — up to
// maxRetryAttempts, after which it is routed to the parked queue instead of
// being nacked again. Pass "" to keep the original discard-on-error
// behavior (no DLX configured).
func (c *Consumer) Consume(
	ctx context.Context,
	exchange string,
	exchangeType ExchangeType,
	queue string,
	bindingKey string,
	deadLetterExchange string,
	handler MessageHandler,
) error {
	ch, err := c.conn.Channel()
	if err != nil {
		c.logger.Error("cannot consume messages", "error", err)
		return err
	}
	defer ch.Close()
	err = ch.ExchangeDeclare(
		exchange,             // name
		string(exchangeType), // type
		true,                 // durable (survives restart)
		false,                // auto-deleted
		false,                // internal
		false,                // no-wait
		nil,                  // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	var queueArgs amqp.Table
	if deadLetterExchange != "" {
		queueArgs = amqp.Table{"x-dead-letter-exchange": deadLetterExchange}
	}

	_, err = ch.QueueDeclare(
		queue,     // name
		true,      // durable (recommended)
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		queueArgs, // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	err = ch.QueueBind(
		queue,      // queue name
		bindingKey, // routing pattern (e.g., "gossip.#")
		exchange,   // exchange name
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	err = ch.Qos(
		c.prefetchCount, // prefetch count
		0,               // prefetch size (0 = no size limit)
		false,           // global (false = apply only to this consumer)
	)
	if err != nil {
		c.logger.Error("failed to set QoS",
			"queue", queue,
			"prefetch_count", c.prefetchCount,
			"error", err,
		)
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming messages
	msgs, err := ch.Consume(
		queue, // queue name
		"",    // consumer tag (auto-generated)
		false, // auto-acknowledge
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		c.logger.Error("failed to start consuming",
			"queue", queue,
			"error", err,
		)
		return fmt.Errorf("failed to consume from queue: %w", err)
	}

	c.logger.Info("consumer started", "queue", queue)

	// Listen for messages
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer stopped", "queue", queue)
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				c.logger.Error("message channel closed", "queue", queue)
				return fmt.Errorf("message channel closed for queue: %s", queue)
			}

			// Call the handler with the message
			if err := handler(ctx, msg.Body); err != nil {
				c.logger.Error("handler error",
					"queue", queue,
					"error", err,
				)

				if deadLetterExchange != "" && deathCount(msg.Headers) >= c.maxRetryAttempts {
					if parkErr := c.park(ctx, msg.Body); parkErr != nil {
						c.logger.Error("failed to park exhausted message, retrying instead",
							"queue", queue,
							"error", parkErr,
						)
						msg.Nack(false, false)
					} else {
						c.logger.Warn("message exhausted retry attempts, parked for manual triage",
							"queue", queue,
							"attempts", c.maxRetryAttempts,
						)
						msg.Ack(false)
					}
				} else {
					msg.Nack(false, false)
					// With a dead-letter-exchange configured, this is not a
					// discard — RabbitMQ routes it into the retry flow.
				}
				// Continue consuming, don't stop on handler error
			} else {
				msg.Ack(false)
			}
		}
	}
}

// park republishes body verbatim to the parked queue, for a message that
// has exhausted its retry attempts and needs manual triage rather than
// another automatic redelivery.
func (c *Consumer) park(ctx context.Context, body []byte) error {
	publisher := NewPublisher(c.conn, &c.logger)
	return publisher.PublishRaw(ctx, "", ParkedQueue, body)
}

// deathCount returns how many times this message has previously been
// dead-lettered for handler rejection (RabbitMQ's "rejected" reason),
// derived from the standard x-death header. Redelivery cycles caused by the
// retry queue's TTL expiry ("expired" reason) are not counted here — those
// happen once per rejection and would otherwise double the count.
func deathCount(headers amqp.Table) int {
	raw, ok := headers["x-death"]
	if !ok {
		return 0
	}

	deaths, ok := raw.([]any)
	if !ok {
		return 0
	}

	total := 0
	for _, d := range deaths {
		entry, ok := d.(amqp.Table)
		if !ok {
			continue
		}
		if reason, _ := entry["reason"].(string); reason != "rejected" {
			continue
		}
		switch count := entry["count"].(type) {
		case int64:
			total += int(count)
		case int32:
			total += int(count)
		}
	}
	return total
}
