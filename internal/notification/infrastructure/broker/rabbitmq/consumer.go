package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/broker/rabbitmq"
)

const queueName = "notifications"

type Consumer struct {
	conn        *rabbitmq.Connection
	emailSender service.EmailService
	sendTimeout time.Duration
	logger      *slog.Logger
}

func NewConsumer(conn *rabbitmq.Connection, emailSender service.EmailService, sendTimeout time.Duration) *Consumer {
	return &Consumer{
		conn:        conn,
		emailSender: emailSender,
		sendTimeout: sendTimeout,
		logger:      slog.With(slog.String("component", "RabbitMQConsumer")),
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	for {
		if err := c.consume(ctx); err != nil {
			c.logger.Error("rabbitmq: consumer error, restarting", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func (c *Consumer) consume(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	msgs, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("deliveries channel closed")
			}
			c.handle(ctx, d)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, d amqp.Delivery) {
	var cmd service.SendNotificationCommand
	if err := json.Unmarshal(d.Body, &cmd); err != nil {
		c.logger.Error("rabbitmq: unmarshal message", slog.Any("error", err))
		_ = d.Nack(false, false)
		return
	}

	sendCtx, cancel := context.WithTimeout(ctx, c.sendTimeout)
	defer cancel()

	if err := c.emailSender.SendNotification(sendCtx, cmd.Email, cmd.RepoName, cmd.Tag, cmd.Token); err != nil {
		c.logger.Error("rabbitmq: send notification email",
			slog.String("to", cmd.Email),
			slog.Any("error", err),
		)
		_ = d.Nack(false, true)
		return
	}

	_ = d.Ack(false)
}
