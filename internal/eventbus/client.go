package eventbus

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

const VERISAFE = "hello"

// EventBus is an interface that defines the contract for any event bus implementation.
// The Publish method now accepts a routing key.
type EventBus interface {
	Publish(ctx context.Context, routingKey string, event interface{}) error
	Subscribe(routingKey string, handler func(event []byte)) error
	Close()
}

// RabbitMQEventBus is a concrete implementation of EventBus that uses RabbitMQ.
type RabbitMQEventBus struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

// NewRabbitMQEventBus creates and returns a new RabbitMQEventBus instance.
// It connects to the RabbitMQ server and declares a durable exchange.
func NewRabbitMQEventBus(amqpURI, exchange string) (*RabbitMQEventBus, error) {
	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declare a durable direct exchange
	err = ch.ExchangeDeclare(
		exchange, // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return nil, err
	}

	return &RabbitMQEventBus{
		conn:     conn,
		channel:  ch,
		exchange: exchange,
	}, nil
}

// Publish serializes the event and sends it to the RabbitMQ exchange.
func (eb *RabbitMQEventBus) Publish(ctx context.Context, routingKey string, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	publishing := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent, // Make message persistent
	}

	return eb.channel.PublishWithContext(
		ctx,
		eb.exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		publishing,
	)
}

// Subscribe declares a queue, binds it to the exchange, and consumes messages.
func (eb *RabbitMQEventBus) Subscribe(routingKey string, handler func(event []byte)) error {
	// Declare a durable, non-exclusive queue
	q, err := eb.channel.QueueDeclare(
		"",    // Name, RabbitMQ will generate a unique name
		true,  // Durable
		false, // Delete when unused
		false, // Not exclusive to this consumer
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return err
	}

	// Bind the queue to the exchange with a specific routing key
	err = eb.channel.QueueBind(
		q.Name,      // queue name
		routingKey,  // routing key
		eb.exchange, // exchange
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := eb.channel.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack (set to false for manual acknowledgment)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return err
	}

	// Start a goroutine to process messages
	go func() {
		for d := range msgs {
			// Process the message
			handler(d.Body)
			
			// Manually acknowledge the message
			if err := d.Ack(false); err != nil {
				// Log error but continue processing
				// In a production environment, you might want to implement retry logic
			}
		}
	}()

	return nil
}

// Close closes the RabbitMQ channel and connection.
func (eb *RabbitMQEventBus) Close() {
	eb.channel.Close()
	eb.conn.Close()
}
