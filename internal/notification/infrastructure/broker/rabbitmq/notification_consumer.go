package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/usecase"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/model"
)

type Consumer struct {
	conn           *sharedRabbitmq.Connection
	notificationUC usecase.NotificationUseCase
	sendTimeout    time.Duration
	logger         *slog.Logger
}

func NewConsumer(
	conn *sharedRabbitmq.Connection,
	notificationUC usecase.NotificationUseCase,
	sendTimeout time.Duration,
) *Consumer {
	return &Consumer{
		conn:           conn,
		notificationUC: notificationUC,
		sendTimeout:    sendTimeout,
		logger:         slog.With(slog.String("component", "RabbitMQConsumer")),
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
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			c.logger.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	_, err = ch.QueueDeclare(sharedRabbitmq.QueueNotifications, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	msgs, err := ch.Consume(sharedRabbitmq.QueueNotifications, "", false, false, false, false, nil)
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
	var cmd model.SendNotificationCommand
	if err := json.Unmarshal(d.Body, &cmd); err != nil {
		c.logger.Error("rabbitmq: unmarshal message", slog.Any("error", err))
		if nackErr := d.Nack(false, false); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	sendCtx, cancel := context.WithTimeout(ctx, c.sendTimeout)
	defer cancel()

	if err := c.notificationUC.Send(sendCtx, cmd); err != nil {
		c.logger.Error("rabbitmq: send notification failed",
			slog.String("to", cmd.Email),
			slog.Any("error", err),
		)
		if nackErr := d.Nack(false, true); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if ackErr := d.Ack(false); ackErr != nil {
		c.logger.Error("rabbitmq: ack failed", slog.Any("error", ackErr))
	}
}
