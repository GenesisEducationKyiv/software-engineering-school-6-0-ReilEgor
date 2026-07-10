package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	rabbitmq2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

type TagUpdatedConsumer struct {
	conn    *rabbitmq2.Connection
	repoUpd repository.RepositoryUpdater
	logger  *slog.Logger
}

func NewTagUpdatedConsumer(
	conn *rabbitmq2.Connection,
	repoUpd repository.RepositoryUpdater,
) *TagUpdatedConsumer {
	return &TagUpdatedConsumer{
		conn:    conn,
		repoUpd: repoUpd,
		logger:  slog.With(slog.String("component", "TagUpdatedConsumer")),
	}
}

func (c *TagUpdatedConsumer) Start(ctx context.Context) error {
	for {
		if err := c.consume(ctx); err != nil {
			c.logger.Error("rabbitmq: tag updated consumer error, restarting", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func (c *TagUpdatedConsumer) consume(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			c.logger.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if _, err = ch.QueueDeclare(rabbitmq2.QueueTagUpdated, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	msgs, err := ch.Consume(rabbitmq2.QueueTagUpdated, "", false, false, false, false, nil)
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

func (c *TagUpdatedConsumer) handle(ctx context.Context, d amqp.Delivery) {
	var event sharedModel.TagUpdatedEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		c.logger.Error("rabbitmq: unmarshal tag updated event", slog.Any("error", err))
		if nackErr := d.Nack(false, false); nackErr != nil {
			c.logger.Error("rabbitmq: nack failed", slog.Any("error", nackErr))
		}
		return
	}

	if err := c.repoUpd.UpdateTag(ctx, event.FullName, event.Tag); err != nil {
		c.logger.Error("tag updated consumer: update tag failed",
			slog.String("repo", event.FullName),
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
