package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	rabbitmq3 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	conn *rabbitmq3.Connection
}

func NewPublisher(conn *rabbitmq3.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	_, err = ch.QueueDeclare(rabbitmq3.QueueConfirmations, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	return &Publisher{conn: conn}, nil
}

func (p *Publisher) SendConfirmation(
	ctx context.Context,
	to, repoName, token string,
	sagaID, subscriptionID int64,
) error {
	return p.Publish(ctx, sharedModel.SendConfirmationCommand{
		Email:          to,
		RepoName:       repoName,
		Token:          token,
		SagaID:         sagaID,
		SubscriptionID: subscriptionID,
	})
}

func (p *Publisher) Publish(ctx context.Context, cmd sharedModel.SendConfirmationCommand) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			slog.Warn("rabbitmq: close channel", slog.Any("error", closeErr))
		}
	}()

	body, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal command: %w", err)
	}

	if err := ch.PublishWithContext(ctx, "", rabbitmq3.QueueConfirmations, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}
