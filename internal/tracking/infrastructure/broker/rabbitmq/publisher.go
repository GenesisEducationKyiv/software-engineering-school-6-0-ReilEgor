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

type Publisher struct {
	conn *rabbitmq2.Connection
}

func NewPublisher(conn *rabbitmq2.Connection) (*Publisher, error) {
	p := &Publisher{conn: conn}
	if err := p.declareQueue(); err != nil {
		return nil, fmt.Errorf("rabbitmq: declare queue: %w", err)
	}
	return p, nil
}

func (p *Publisher) Publish(ctx context.Context, cmd sharedModel.SendNotificationCommand) error {
	body, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("rabbitmq: marshal: %w", err)
	}

	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if err := ch.PublishWithContext(ctx, "", rabbitmq2.QueueNotifications, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("rabbitmq: publish: %w", err)
	}
	return nil
}

func (p *Publisher) declareQueue() error {
	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	if _, err = ch.QueueDeclare(rabbitmq2.QueueNotifications, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	return nil
}
