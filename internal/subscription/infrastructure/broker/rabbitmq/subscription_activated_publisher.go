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

type SubscriptionActivatedPublisher struct {
	conn *rabbitmq2.Connection
}

func NewSubscriptionActivatedPublisher(conn *rabbitmq2.Connection) (*SubscriptionActivatedPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if _, err = ch.QueueDeclare(rabbitmq2.QueueSubscriptionActivated, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	return &SubscriptionActivatedPublisher{conn: conn}, nil
}

func (p *SubscriptionActivatedPublisher) Publish(
	ctx context.Context,
	event sharedModel.SubscriptionActivatedEvent,
) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("subscription activated publisher: marshal: %w", err)
	}

	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("subscription activated publisher: open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if err := ch.PublishWithContext(ctx, "", rabbitmq2.QueueSubscriptionActivated, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("subscription activated publisher: publish: %w", err)
	}
	return nil
}
