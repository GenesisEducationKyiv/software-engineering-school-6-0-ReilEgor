package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"
	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"

	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/usecase"
)

type TrackerSubscriptionDeleter interface {
	DeleteByEmailAndRepo(ctx context.Context, email, repoName string) error
	HasSubscriptions(ctx context.Context, repoName string) (bool, error)
}

type UnsubscriptionActivatedConsumer struct {
	conn    *sharedRabbitmq.Connection
	repoUC  trackingDomainUsecase.RepositoryUseCase
	subRepo TrackerSubscriptionDeleter
	logger  *slog.Logger
}

func NewUnsubscriptionActivatedConsumer(
	conn *sharedRabbitmq.Connection,
	repoUC trackingDomainUsecase.RepositoryUseCase,
	subRepo TrackerSubscriptionDeleter,
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
	c.logger.DebugContext(ctx, "called", slog.String("op", "UnsubscriptionActivatedConsumer.consume"))

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
		contracts.QueueUnsubscriptionActivated,
		true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	msgs, err := ch.Consume(contracts.QueueUnsubscriptionActivated, "", false, false, false, false, nil)
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
	var event contracts.UnsubscriptionActivatedEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		c.logger.Error("rabbitmq: unmarshal unsubscription activated event", slog.Any("error", err))
		if nackErr := d.Nack(false, false); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if event.RequestID != "" {
		ctx = ctxlog.WithRequestID(ctx, event.RequestID)
		ctx = ctxlog.WithLogger(ctx, c.logger.With(slog.String("request_id", event.RequestID)))
	}
	log := ctxlog.FromCtx(ctx)
	log.DebugContext(ctx, "called", slog.String("op", "UnsubscriptionActivatedConsumer.handle"))

	if err := c.subRepo.DeleteByEmailAndRepo(ctx, event.Email, event.RepoName); err != nil {
		log.Error("unsubscription: delete tracker subscription failed",
			slog.String("repo", event.RepoName),
			slog.String("email", event.Email),
			slog.Any("error", err),
		)
		if nackErr := d.Nack(false, true); nackErr != nil {
			log.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	hasMore, err := c.subRepo.HasSubscriptions(ctx, event.RepoName)
	if err != nil {
		log.Error("unsubscription: check subscriptions failed", slog.Any("error", err))
		if nackErr := d.Nack(false, true); nackErr != nil {
			log.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if !hasMore {
		if err := c.repoUC.Delete(ctx, event.RepoName); err != nil {
			log.Error("unsubscription: delete repository failed",
				slog.String("repo", event.RepoName),
				slog.Any("error", err),
			)
			if nackErr := d.Nack(false, true); nackErr != nil {
				log.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
			}
			return
		}
	}

	if ackErr := d.Ack(false); ackErr != nil {
		log.Error("rabbitmq: ack failed", slog.Any("error", ackErr))
	}
}
