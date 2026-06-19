package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

type subscriptionActivatedPublisher interface {
	Publish(ctx context.Context, event sharedModel.SubscriptionActivatedEvent) error
}

type SagaResultConsumer struct {
	conn        *rabbitmq.Connection
	subsRepo    repository.SubscriptionWriter
	sagaRepo    repository.SagaRepository
	activatedPub subscriptionActivatedPublisher
	logger      *slog.Logger
}

func NewSagaResultConsumer(
	conn *rabbitmq.Connection,
	subsRepo repository.SubscriptionWriter,
	sagaRepo repository.SagaRepository,
	activatedPub *SubscriptionActivatedPublisher,
) *SagaResultConsumer {
	return &SagaResultConsumer{
		conn:         conn,
		subsRepo:     subsRepo,
		sagaRepo:     sagaRepo,
		activatedPub: activatedPub,
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

	_, err = ch.QueueDeclare(rabbitmq.QueueSagaConfirmationResults, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	msgs, err := ch.Consume(rabbitmq.QueueSagaConfirmationResults, "", false, false, false, false, nil)
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
	var event sharedModel.ConfirmationResultEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		c.logger.Error("rabbitmq: unmarshal saga result", slog.Any("error", err))
		c.nack(d, false)
		return
	}

	if event.Success {
		c.handleSuccess(ctx, d, event)
	} else {
		c.handleCompensation(ctx, d, event)
	}
}

func (c *SagaResultConsumer) handleSuccess(
	ctx context.Context,
	d amqp.Delivery,
	event sharedModel.ConfirmationResultEvent,
) {
	if err := c.sagaRepo.UpdateStatus(ctx, event.SagaID, sharedModel.SagaStatusCompleted); err != nil {
		c.logger.Error("saga: update status to COMPLETED failed", slog.Any("error", err))
		c.nack(d, true)
		return
	}

	if err := c.activatedPub.Publish(ctx, sharedModel.SubscriptionActivatedEvent{
		FullName: event.RepoName,
		Email:    event.Email,
		Token:    event.Token,
	}); err != nil {
		c.logger.Warn("saga: publish subscription.activated failed", slog.Any("error", err))
	}

	c.ack(d)
}

func (c *SagaResultConsumer) handleCompensation(
	ctx context.Context,
	d amqp.Delivery,
	event sharedModel.ConfirmationResultEvent,
) {
	c.logger.Warn("saga: confirmation failed, compensating",
		slog.Int64("saga_id", event.SagaID),
		slog.Int64("subscription_id", event.SubscriptionID),
	)
	if err := c.subsRepo.DeleteByID(ctx, event.SubscriptionID); err != nil {
		c.logger.Error("saga: compensation failed (delete subscription)", slog.Any("error", err))
		c.nack(d, true)
		return
	}
	if err := c.sagaRepo.UpdateStatus(ctx, event.SagaID, sharedModel.SagaStatusCompensated); err != nil {
		c.logger.Error("saga: update status to COMPENSATED failed", slog.Any("error", err))
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
