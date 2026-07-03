package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"
	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/service"
)

//go:generate mockery --name SagaResultPublisher --output ../../mocks --case underscore --outpkg mocks
type SagaResultPublisher interface {
	Publish(ctx context.Context, event contracts.ConfirmationResultEvent) error
}

type ConfirmationConsumer struct {
	conn          *sharedRabbitmq.Connection
	emailSvc      service.EmailService
	sendTimeout   time.Duration
	sagaResultPub SagaResultPublisher
	logger        *slog.Logger
}

func NewConfirmationConsumer(
	conn *sharedRabbitmq.Connection,
	emailSvc service.EmailService,
	sendTimeout time.Duration,
	sagaResultPub SagaResultPublisher,
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
	c.logger.DebugContext(ctx, "called", slog.String("op", "ConfirmationConsumer.consume"))

	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			c.logger.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	_, err = ch.QueueDeclare(contracts.QueueConfirmations, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	msgs, err := ch.Consume(contracts.QueueConfirmations, "", false, false, false, false, nil)
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
	var cmd contracts.SendConfirmationCommand
	if err := json.Unmarshal(d.Body, &cmd); err != nil {
		c.logger.Error("rabbitmq: unmarshal confirmation message", slog.Any("error", err))
		if nackErr := d.Nack(false, false); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if cmd.RequestID != "" {
		ctx = ctxlog.WithRequestID(ctx, cmd.RequestID)
		ctx = ctxlog.WithLogger(ctx, c.logger.With(slog.String("request_id", cmd.RequestID)))
	}
	log := ctxlog.FromCtx(ctx)
	log.DebugContext(ctx, "called", slog.String("op", "ConfirmationConsumer.handle"))

	sendCtx, cancel := context.WithTimeout(ctx, c.sendTimeout)
	defer cancel()

	if err := c.emailSvc.SendConfirmation(sendCtx, cmd.Email, cmd.RepoName, cmd.Token); err != nil {
		log.Error("rabbitmq: send confirmation email failed",
			slog.String("to", cmd.Email),
			slog.Any("error", err),
		)
		c.handleEmailError(ctx, d, cmd, err, log)
		return
	}

	if pubErr := c.sagaResultPub.Publish(ctx, contracts.ConfirmationResultEvent{
		SagaID:         cmd.SagaID,
		SubscriptionID: cmd.SubscriptionID,
		Success:        true,
		Email:          cmd.Email,
		RepoName:       cmd.RepoName,
		Token:          cmd.Token,
		RequestID:      cmd.RequestID,
	}); pubErr != nil {
		log.Error("rabbitmq: publish saga result failed, will retry", slog.Any("error", pubErr))
		if nackErr := d.Nack(false, true); nackErr != nil {
			log.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if ackErr := d.Ack(false); ackErr != nil {
		log.Error("rabbitmq: ack failed", slog.Any("error", ackErr))
	}
}

func (c *ConfirmationConsumer) handleEmailError(
	ctx context.Context,
	d amqp.Delivery,
	cmd contracts.SendConfirmationCommand,
	err error,
	log *slog.Logger,
) {
	if errors.Is(err, service.ErrSMTPUnavailable) {
		if nackErr := d.Nack(false, true); nackErr != nil {
			log.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if pubErr := c.sagaResultPub.Publish(ctx, contracts.ConfirmationResultEvent{
		SagaID:         cmd.SagaID,
		SubscriptionID: cmd.SubscriptionID,
		Success:        false,
		Email:          cmd.Email,
		RepoName:       cmd.RepoName,
		Token:          cmd.Token,
		RequestID:      cmd.RequestID,
	}); pubErr != nil {
		log.Error("rabbitmq: publish saga result failed, requeuing", slog.Any("error", pubErr))
		if nackErr := d.Nack(false, true); nackErr != nil {
			log.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}
	if nackErr := d.Nack(false, false); nackErr != nil {
		log.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
	}
}
