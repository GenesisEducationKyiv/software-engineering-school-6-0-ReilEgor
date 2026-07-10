package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"

	rabbitmq2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

type TagUpdatedPublisher struct {
	conn *rabbitmq2.Connection
}

func NewTagUpdatedPublisher(conn *rabbitmq2.Connection) (*TagUpdatedPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if _, err = ch.QueueDeclare(rabbitmq2.QueueTagUpdated, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	return &TagUpdatedPublisher{conn: conn}, nil
}

func (p *TagUpdatedPublisher) Publish(ctx context.Context, event sharedModel.TagUpdatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("tag updated publisher: marshal: %w", err)
	}

	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("tag updated publisher: open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if err := ch.PublishWithContext(ctx, "", rabbitmq2.QueueTagUpdated, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("tag updated publisher: publish: %w", err)
	}
	return nil
}
