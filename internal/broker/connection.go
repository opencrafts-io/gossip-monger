package broker

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection interface {
	// Channel opens a new AMQP channel. amqp.Channel is not safe for
	// concurrent RPC calls (Declare/Bind/Qos/Consume), so callers must not
	// share a channel across goroutines — each consumer/publish operation
	// should get its own via a fresh call to Channel().
	Channel() (*amqp.Channel, error)
	Close() error
	IsClosed() bool
}

type RabbitMQConnection struct {
	conn *amqp.Connection
}

func NewRabbitMQConnection(
	ctx context.Context,
	dsn string,
) (*RabbitMQConnection, error) {
	conn, err := amqp.Dial(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	return &RabbitMQConnection{
		conn: conn,
	}, nil
}

// Channel opens a new AMQP channel on the underlying connection. Each call
// returns a distinct channel — callers own its lifecycle and must close it
// when done.
func (rc *RabbitMQConnection) Channel() (*amqp.Channel, error) {
	ch, err := rc.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}
	return ch, nil
}

// Close closes the connection
func (rc *RabbitMQConnection) Close() error {
	if rc.conn != nil {
		return rc.conn.Close()
	}
	return nil
}

// IsClosed checks if the connection is closed
func (rc *RabbitMQConnection) IsClosed() bool {
	return rc.conn == nil || rc.conn.IsClosed()
}
