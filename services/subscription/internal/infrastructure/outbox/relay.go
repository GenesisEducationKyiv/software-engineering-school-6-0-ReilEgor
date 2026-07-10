package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/repository"
)

const (
	defaultBatchSize   = 100
	defaultMaxAttempts = 5
)

type Publisher interface {
	Publish(ctx context.Context, queue string, body []byte) error
}

type Relay struct {
	outboxRepo  repository.OutboxRepository
	transactor  repository.Transactor
	publisher   Publisher
	interval    time.Duration
	batchSize   int
	maxAttempts int
	logger      *slog.Logger
}

func NewRelay(
	outboxRepo repository.OutboxRepository,
	conn *sharedRabbitmq.Connection,
	interval time.Duration,
	transactor repository.Transactor,
) *Relay {
	return NewRelayWithPublisher(outboxRepo, NewRabbitPublisher(conn), interval, transactor)
}

func NewRelayWithPublisher(
	outboxRepo repository.OutboxRepository,
	publisher Publisher,
	interval time.Duration,
	transactor repository.Transactor,
) *Relay {
	return &Relay{
		outboxRepo:  outboxRepo,
		transactor:  transactor,
		publisher:   publisher,
		interval:    interval,
		batchSize:   defaultBatchSize,
		maxAttempts: defaultMaxAttempts,
		logger:      slog.With(slog.String("component", "OutboxRelay")),
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
	err := r.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		msgs, err := r.outboxRepo.FetchPending(txCtx, r.batchSize)
		if err != nil {
			return fmt.Errorf("fetch pending: %w", err)
		}

		for _, msg := range msgs {
			r.processMessage(txCtx, msg)
		}
		return nil
	})
	if err != nil {
		r.logger.Error("outbox: process batch failed", slog.Any("error", err))
	}
}

func (r *Relay) processMessage(ctx context.Context, msg sharedModel.OutboxMessage) {
	if err := r.publisher.Publish(ctx, msg.Queue, msg.Payload); err != nil {
		r.logger.Error("outbox: publish failed",
			slog.String("queue", msg.Queue),
			slog.Int64("id", msg.ID),
			slog.Any("error", err),
		)
		if failErr := r.outboxRepo.Fail(ctx, msg.ID, r.maxAttempts, err.Error()); failErr != nil {
			r.logger.Error("outbox: mark failed",
				slog.Int64("id", msg.ID),
				slog.Any("error", failErr),
			)
		}
		return
	}
	if err := r.outboxRepo.Delete(ctx, msg.ID); err != nil {
		r.logger.Error("outbox: delete after publish failed",
			slog.Int64("id", msg.ID),
			slog.Any("error", err),
		)
	}
}

type rabbitPublisher struct {
	conn *sharedRabbitmq.Connection
}

func NewRabbitPublisher(conn *sharedRabbitmq.Connection) Publisher {
	return &rabbitPublisher{conn: conn}
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

	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("%s: enable confirms: %w", op, err)
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	if err := ch.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	select {
	case confirm := <-confirms:
		if !confirm.Ack {
			return fmt.Errorf("%s: broker nacked message for queue %q", op, queue)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%s: %w", op, ctx.Err())
	}
}
