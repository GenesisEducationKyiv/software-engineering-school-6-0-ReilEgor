package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/port"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

type ConfirmationConsumer struct {
	conn          *rabbitmq.Connection
	emailSvc      service.EmailService
	sendTimeout   time.Duration
	sagaResultPub port.SagaResultPublisher
	logger        *slog.Logger
}

func NewConfirmationConsumer(
	conn *rabbitmq.Connection,
	emailSvc service.EmailService,
	sendTimeout time.Duration,
	sagaResultPub port.SagaResultPublisher,
) *ConfirmationConsumer {
	return &ConfirmationConsumer{
		conn:          conn,
		emailSvc:      emailSvc,
		sendTimeout:   sendTimeout,
		sagaResultPub: sagaResultPub,
		logger:        slog.With(slog.String("component", "ConfirmationConsumer")),
	}
}

func (c *ConfirmationConsumer) Start(ctx context.Context) error {
	for {
		if err := c.consume(ctx); err != nil {
			c.logger.Error("rabbitmq: confirmation consumer error, restarting", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func (c *ConfirmationConsumer) consume(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			c.logger.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	_, err = ch.QueueDeclare(rabbitmq.QueueConfirmations, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	msgs, err := ch.Consume(rabbitmq.QueueConfirmations, "", false, false, false, false, nil)
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

func (c *ConfirmationConsumer) handle(ctx context.Context, d amqp.Delivery) {
	var cmd model.SendConfirmationCommand
	if err := json.Unmarshal(d.Body, &cmd); err != nil {
		c.logger.Error("rabbitmq: unmarshal confirmation message", slog.Any("error", err))
		if nackErr := d.Nack(false, false); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	sendCtx, cancel := context.WithTimeout(ctx, c.sendTimeout)
	defer cancel()

	if err := c.emailSvc.SendConfirmation(sendCtx, cmd.Email, cmd.RepoName, cmd.Token); err != nil {
		c.logger.Error("rabbitmq: send confirmation email failed",
			slog.String("to", cmd.Email),
			slog.Any("error", err),
		)
		c.handleEmailError(ctx, d, cmd, err)
		return
	}

	if pubErr := c.sagaResultPub.Publish(ctx, model.ConfirmationResultEvent{
		SagaID:         cmd.SagaID,
		SubscriptionID: cmd.SubscriptionID,
		Success:        true,
		Email:          cmd.Email,
		RepoName:       cmd.RepoName,
		Token:          cmd.Token,
	}); pubErr != nil {
		c.logger.Error("rabbitmq: publish saga result failed, will retry", slog.Any("error", pubErr))
		if nackErr := d.Nack(false, true); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if ackErr := d.Ack(false); ackErr != nil {
		c.logger.Error("rabbitmq: ack failed", slog.Any("error", ackErr))
	}
}

func (c *ConfirmationConsumer) handleEmailError(
	ctx context.Context,
	d amqp.Delivery,
	cmd model.SendConfirmationCommand,
	err error,
) {
	if errors.Is(err, service.ErrSMTPUnavailable) {
		if nackErr := d.Nack(false, true); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if pubErr := c.sagaResultPub.Publish(ctx, model.ConfirmationResultEvent{
		SagaID:         cmd.SagaID,
		SubscriptionID: cmd.SubscriptionID,
		Success:        false,
		Email:          cmd.Email,
		RepoName:       cmd.RepoName,
		Token:          cmd.Token,
	}); pubErr != nil {
		c.logger.Error("rabbitmq: publish saga result failed", slog.Any("error", pubErr))
	}
	if nackErr := d.Nack(false, false); nackErr != nil {
		c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
	}
}
