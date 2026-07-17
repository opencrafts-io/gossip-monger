package broker

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// RetryExchange is the shared dead-letter target for queues that want
	// automatic delayed redelivery instead of losing a failed message.
	RetryExchange = "gossip.retry.exchange"
	// RetryQueue holds a dead-lettered message for its configured delay,
	// then RabbitMQ redelivers it (via its own x-dead-letter-exchange) back
	// to whichever queue its original routing key maps to.
	RetryQueue = "gossip.retry.queue"
	// ParkedQueue is the terminal destination for messages that exhausted
	// their retry attempts. Never auto-consumed — for manual triage.
	ParkedQueue = "gossip.parked.queue"
)

// DeclareRetryTopology declares the shared retry (delayed-requeue) queue and
// the terminal parked queue used by consumers that opt into automatic
// retry-with-backoff instead of dropping a failed message.
//
// sourceExchange is where a message is redelivered once its retry delay
// elapses — the queues that dead-letter into RetryExchange must already be
// bound to sourceExchange under the same routing keys passed here, or the
// redelivered message has nowhere to land.
func DeclareRetryTopology(
	conn Connection,
	retryDelay time.Duration,
	sourceExchange string,
	routingKeys []string,
) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(
		RetryExchange,
		string(DirectExchangeType),
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("failed to declare retry exchange: %w", err)
	}

	_, err = ch.QueueDeclare(
		RetryQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-message-ttl":          int64(retryDelay / time.Millisecond),
			"x-dead-letter-exchange": sourceExchange,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to declare retry queue: %w", err)
	}

	for _, key := range routingKeys {
		if err := ch.QueueBind(RetryQueue, key, RetryExchange, false, nil); err != nil {
			return fmt.Errorf("failed to bind retry queue for %q: %w", key, err)
		}
	}

	if _, err := ch.QueueDeclare(
		ParkedQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("failed to declare parked queue: %w", err)
	}

	return nil
}
