package rabbitmq

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	initialBackoff = time.Second
	maxBackoff     = 30 * time.Second
)

type Connection struct {
	url    string
	mu     sync.Mutex
	conn   *amqp.Connection
	logger *slog.Logger
}

func NewConnection(url string) (*Connection, func(), error) {
	c := &Connection{
		url:    url,
		logger: slog.With(slog.String("component", "RabbitMQConnection")),
	}
	if err := c.connect(); err != nil {
		return nil, nil, fmt.Errorf("rabbitmq: initial connect: %w", err)
	}
	cleanup := func() { _ = c.conn.Close() }
	return c, cleanup, nil
}

func (c *Connection) Channel() (*amqp.Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || c.conn.IsClosed() {
		c.reconnect()
	}
	return c.conn.Channel()
}

func (c *Connection) connect() error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Connection) reconnect() {
	backoff := initialBackoff
	for {
		c.logger.Warn("rabbitmq: reconnecting", slog.Duration("backoff", backoff))
		time.Sleep(backoff)
		if err := c.connect(); err == nil {
			c.logger.Info("rabbitmq: reconnected")
			return
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
