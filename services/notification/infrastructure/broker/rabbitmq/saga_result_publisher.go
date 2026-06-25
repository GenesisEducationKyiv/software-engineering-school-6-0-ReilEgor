package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	rabbitmq2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SagaResultPublisher struct {
	conn *rabbitmq2.Connection
}

func NewSagaResultPublisher(conn *rabbitmq2.Connection) (*SagaResultPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	_, err = ch.QueueDeclare(rabbitmq2.QueueSagaConfirmationResults, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	return &SagaResultPublisher{conn: conn}, nil
}

func (p *SagaResultPublisher) Publish(ctx context.Context, event sharedModel.ConfirmationResultEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("saga result publisher: marshal: %w", err)
	}

	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("saga result publisher: open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if err := ch.PublishWithContext(ctx, "", rabbitmq2.QueueSagaConfirmationResults, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("saga result publisher: publish: %w", err)
	}
	return nil
}
