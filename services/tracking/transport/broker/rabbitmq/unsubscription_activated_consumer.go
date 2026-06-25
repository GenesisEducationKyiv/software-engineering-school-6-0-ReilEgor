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

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/port"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/usecase"
)

type UnsubscriptionActivatedConsumer struct {
	conn    *sharedRabbitmq.Connection
	repoUC  trackingDomainUsecase.RepositoryUseCase
	subRepo port.TrackerSubscriptionDeleter
	logger  *slog.Logger
}

func NewUnsubscriptionActivatedConsumer(
	conn *sharedRabbitmq.Connection,
	repoUC trackingDomainUsecase.RepositoryUseCase,
	subRepo port.TrackerSubscriptionDeleter,
) *UnsubscriptionActivatedConsumer {
	return &UnsubscriptionActivatedConsumer{
		conn:    conn,
		repoUC:  repoUC,
		subRepo: subRepo,
		logger:  slog.With(slog.String("component", "UnsubscriptionActivatedConsumer")),
	}
}

func (c *UnsubscriptionActivatedConsumer) Start(ctx context.Context) error {
	for {
		if err := c.consume(ctx); err != nil {
			c.logger.Error("rabbitmq: unsubscription activated consumer error, restarting", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func (c *UnsubscriptionActivatedConsumer) consume(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			c.logger.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if _, err = ch.QueueDeclare(
		sharedRabbitmq.QueueUnsubscriptionActivated,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	msgs, err := ch.Consume(sharedRabbitmq.QueueUnsubscriptionActivated, "", false, false, false, false, nil)
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

func (c *UnsubscriptionActivatedConsumer) handle(ctx context.Context, d amqp.Delivery) {
	var event sharedModel.UnsubscriptionActivatedEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		c.logger.Error("rabbitmq: unmarshal unsubscription activated event", slog.Any("error", err))
		if nackErr := d.Nack(false, false); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if err := c.subRepo.DeleteByEmailAndRepo(ctx, event.Email, event.RepoName); err != nil {
		c.logger.Error("unsubscription: delete tracker subscription failed",
			slog.String("repo", event.RepoName),
			slog.String("email", event.Email),
			slog.Any("error", err),
		)
		if nackErr := d.Nack(false, true); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	hasMore, err := c.subRepo.HasSubscriptions(ctx, event.RepoName)
	if err != nil {
		c.logger.Error("unsubscription: check subscriptions failed", slog.Any("error", err))
		if nackErr := d.Nack(false, true); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if !hasMore {
		if err := c.repoUC.Delete(ctx, event.RepoName); err != nil {
			c.logger.Error("unsubscription: delete repository failed",
				slog.String("repo", event.RepoName),
				slog.Any("error", err),
			)
			if nackErr := d.Nack(false, true); nackErr != nil {
				c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
			}
			return
		}
	}

	if ackErr := d.Ack(false); ackErr != nil {
		c.logger.Error("rabbitmq: ack failed", slog.Any("error", ackErr))
	}
}
