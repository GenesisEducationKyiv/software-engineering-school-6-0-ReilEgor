package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/broker/rabbitmq"
)

const queueName = "notifications"

type Publisher struct {
	conn *rabbitmq.Connection
}

func NewPublisher(conn *rabbitmq.Connection) (*Publisher, error) {
	p := &Publisher{conn: conn}
	if err := p.declareQueue(); err != nil {
		return nil, fmt.Errorf("rabbitmq: declare queue: %w", err)
	}
	return p, nil
}

func (p *Publisher) Publish(ctx context.Context, cmd service.SendNotificationCommand) error {
	body, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("rabbitmq: marshal: %w", err)
	}

	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: open channel: %w", err)
	}
	defer ch.Close()

	return ch.PublishWithContext(ctx, "", queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Publisher) declareQueue() error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	return err
}
