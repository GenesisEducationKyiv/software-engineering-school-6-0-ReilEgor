package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

type UnsubscriptionActivatedPublisher struct {
	conn *rabbitmq.Connection
}

func NewUnsubscriptionActivatedPublisher(conn *rabbitmq.Connection) (*UnsubscriptionActivatedPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if _, err = ch.QueueDeclare(rabbitmq.QueueUnsubscriptionActivated, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	return &UnsubscriptionActivatedPublisher{conn: conn}, nil
}

func (p *UnsubscriptionActivatedPublisher) Publish(
	ctx context.Context,
	event sharedModel.SubscriptionActivatedEvent,
) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("unsubscription activated publisher: marshal: %w", err)
	}

	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("unsubscription activated publisher: open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if err := ch.PublishWithContext(ctx, "", rabbitmq.QueueUnsubscriptionActivated, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("unsubscription activated publisher: publish: %w", err)
	}
	return nil
}
