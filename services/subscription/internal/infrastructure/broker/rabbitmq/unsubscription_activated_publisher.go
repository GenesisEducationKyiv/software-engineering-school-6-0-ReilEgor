package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type UnsubscriptionActivatedPublisher struct {
	conn *sharedRabbitmq.Connection
}

func NewUnsubscriptionActivatedPublisher(conn *sharedRabbitmq.Connection) (*UnsubscriptionActivatedPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if _, err = ch.QueueDeclare(contracts.QueueUnsubscriptionActivated, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	return &UnsubscriptionActivatedPublisher{conn: conn}, nil
}

func (p *UnsubscriptionActivatedPublisher) Publish(
	ctx context.Context,
	event contracts.UnsubscriptionActivatedEvent,
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

	if err := ch.PublishWithContext(ctx, "", contracts.QueueUnsubscriptionActivated, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("unsubscription activated publisher: publish: %w", err)
	}
	return nil
}
