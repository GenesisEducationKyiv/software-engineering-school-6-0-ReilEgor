package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	sharedRabbit "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

type subscriptionActivatedPublisher interface {
	Publish(ctx context.Context, event sharedModel.SubscriptionActivatedEvent) error
}

type SagaResultConsumer struct {
	conn         *sharedRabbit.Connection
	subsRepo     repository.SubscriptionWriter
	sagaRepo     repository.SagaRepository
	activatedPub subscriptionActivatedPublisher
	transactor   repository.Transactor
	outboxRepo   repository.OutboxRepository
	logger       *slog.Logger
}

func NewSagaResultConsumer(
	conn *sharedRabbit.Connection,
	subsRepo repository.SubscriptionWriter,
	sagaRepo repository.SagaRepository,
	activatedPub *SubscriptionActivatedPublisher,
	transactor repository.Transactor,
	outboxRepo repository.OutboxRepository,
) *SagaResultConsumer {
	return &SagaResultConsumer{
		conn:         conn,
		subsRepo:     subsRepo,
		sagaRepo:     sagaRepo,
		activatedPub: activatedPub,
		transactor:   transactor,
		outboxRepo:   outboxRepo,
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

	_, err = ch.QueueDeclare(sharedRabbit.QueueSagaConfirmationResults, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	msgs, err := ch.Consume(sharedRabbit.QueueSagaConfirmationResults, "", false, false, false, false, nil)
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
	err := c.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if txErr := c.sagaRepo.UpdateStatus(txCtx, event.SagaID, sharedModel.SagaStatusCompleted); txErr != nil {
			return fmt.Errorf("updateStatus: %w", txErr)
		}
		payload, txErr := json.Marshal(sharedModel.SubscriptionActivatedEvent{
			FullName: event.RepoName,
			Email:    event.Email,
			Token:    event.Token,
		})
		if txErr != nil {
			return fmt.Errorf("marshal: %w", txErr)
		}
		return c.outboxRepo.Insert(txCtx, sharedRabbit.QueueSubscriptionActivated, payload)
	})
	if err != nil {
		c.logger.Error("saga: handleSuccess failed", slog.Any("error", err))
		c.nack(d, true)
		return
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
