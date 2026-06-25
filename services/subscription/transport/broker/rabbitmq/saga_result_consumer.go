package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/saga"
)

type SagaResultConsumer struct {
	conn         *sharedRabbitmq.Connection
	orchestrator *saga.Orchestrator
	logger       *slog.Logger
}

func NewSagaResultConsumer(
	conn *sharedRabbitmq.Connection,
	orchestrator *saga.Orchestrator,
) *SagaResultConsumer {
	return &SagaResultConsumer{
		conn:         conn,
		orchestrator: orchestrator,
		logger:       slog.With(slog.String("component", "SagaResultConsumer")),
	}
}

func (c *SagaResultConsumer) Start(ctx context.Context) error {
	for {
		if err := c.consume(ctx); err != nil {
			c.logger.Error("rabbitmq: saga result consumer error, restarting", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func (c *SagaResultConsumer) consume(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			c.logger.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	_, err = ch.QueueDeclare(sharedRabbitmq.QueueSagaConfirmationResults, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	msgs, err := ch.Consume(sharedRabbitmq.QueueSagaConfirmationResults, "", false, false, false, false, nil)
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

func (c *SagaResultConsumer) handle(ctx context.Context, d amqp.Delivery) {
	var reply sharedModel.ConfirmationResultEvent
	if err := json.Unmarshal(d.Body, &reply); err != nil {
		c.logger.Error("rabbitmq: unmarshal saga result", slog.Any("error", err))
		c.nack(d, false)
		return
	}

	if err := c.orchestrator.HandleConfirmationReply(ctx, reply); err != nil {
		c.logger.Error("rabbitmq: saga orchestrator error", slog.Any("error", err))
		c.nack(d, true)
		return
	}

	c.ack(d)
}

func (c *SagaResultConsumer) ack(d amqp.Delivery) {
	if err := d.Ack(false); err != nil {
		c.logger.Error("rabbitmq: ack failed", slog.Any("error", err))
	}
}

func (c *SagaResultConsumer) nack(d amqp.Delivery, requeue bool) {
	if err := d.Nack(false, requeue); err != nil {
		c.logger.Error("rabbitmq: nack failed", slog.Any("error", err))
	}
}
