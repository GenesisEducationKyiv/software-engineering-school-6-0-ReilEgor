package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	trackingRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/repository"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
)

const defaultBatchSize = 100

type rawPublisher interface {
	Publish(ctx context.Context, queue string, body []byte) error
}

type Relay struct {
	outboxRepo trackingRepo.OutboxRepository
	publisher  rawPublisher
	interval   time.Duration
	batchSize  int
	logger     *slog.Logger
}

func NewRelay(
	outboxRepo trackingRepo.OutboxRepository,
	conn *sharedRabbitmq.Connection,
	interval time.Duration,
) *Relay {
	return &Relay{
		outboxRepo: outboxRepo,
		publisher:  &rabbitPublisher{conn: conn},
		interval:   interval,
		batchSize:  defaultBatchSize,
		logger:     slog.With(slog.String("component", "OutboxRelay")),
	}
}

func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			r.process(ctx)
		}
	}
}

func (r *Relay) process(ctx context.Context) {
	msgs, err := r.outboxRepo.FetchPending(ctx, r.batchSize)
	if err != nil {
		r.logger.Error("outbox: fetch pending", slog.Any("error", err))
		return
	}

	for _, msg := range msgs {
		if err := r.publisher.Publish(ctx, msg.Queue, msg.Payload); err != nil {
			r.logger.Error("outbox: publish failed",
				slog.String("queue", msg.Queue),
				slog.Int64("id", msg.ID),
				slog.Any("error", err),
			)
			continue
		}
		if err := r.outboxRepo.Delete(ctx, msg.ID); err != nil {
			r.logger.Error("outbox: delete after publish failed",
				slog.Int64("id", msg.ID),
				slog.Any("error", err),
			)
		}
	}
}

type rabbitPublisher struct {
	conn *sharedRabbitmq.Connection
}

func (p *rabbitPublisher) Publish(ctx context.Context, queue string, body []byte) error {
	const op = "rabbitPublisher.Publish"

	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("%s: open channel: %w", op, err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("outbox: close channel", slog.Any("error", closeErr))
		}
	}()

	if err := ch.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
