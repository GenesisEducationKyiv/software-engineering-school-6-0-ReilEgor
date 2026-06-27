package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"

	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/usecase"
)

type TrackerSubscriptionWriter interface {
	Upsert(ctx context.Context, repoID int64, email, token string) error
}

type SubscriptionActivatedConsumer struct {
	conn    *sharedRabbitmq.Connection
	repoUC  trackingDomainUsecase.RepositoryUseCase
	subRepo TrackerSubscriptionWriter
	logger  *slog.Logger
}

func NewSubscriptionActivatedConsumer(
	conn *sharedRabbitmq.Connection,
	repoUC trackingDomainUsecase.RepositoryUseCase,
	subRepo TrackerSubscriptionWriter,
) *SubscriptionActivatedConsumer {
	return &SubscriptionActivatedConsumer{
		conn:    conn,
		repoUC:  repoUC,
		subRepo: subRepo,
		logger:  slog.With(slog.String("component", "SubscriptionActivatedConsumer")),
	}
}

func (c *SubscriptionActivatedConsumer) Start(ctx context.Context) error {
	for {
		if err := c.consume(ctx); err != nil {
			c.logger.Error("rabbitmq: subscription activated consumer error, restarting", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func (c *SubscriptionActivatedConsumer) consume(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			c.logger.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if _, err = ch.QueueDeclare(contracts.QueueSubscriptionActivated, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	msgs, err := ch.Consume(contracts.QueueSubscriptionActivated, "", false, false, false, false, nil)
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

func (c *SubscriptionActivatedConsumer) handle(ctx context.Context, d amqp.Delivery) {
	var event contracts.SubscriptionActivatedEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		c.logger.Error("rabbitmq: unmarshal subscription activated event", slog.Any("error", err))
		if nackErr := d.Nack(false, false); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	repo, err := c.repoUC.GetOrCreate(ctx, event.FullName)
	if err != nil {
		c.logger.Error("subscription activated: get or create repo failed",
			slog.String("repo", event.FullName),
			slog.Any("error", err),
		)
		if nackErr := d.Nack(false, true); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if err := c.subRepo.Upsert(ctx, repo.ID, event.Email, event.Token); err != nil {
		c.logger.Error("subscription activated: upsert tracker subscription failed",
			slog.String("repo", event.FullName),
			slog.String("email", event.Email),
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
