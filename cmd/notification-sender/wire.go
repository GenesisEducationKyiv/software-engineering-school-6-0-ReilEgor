//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"

	notifRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/infrastructure/broker/rabbitmq"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
)

func ProvideEmailConfig(cfg Config) config.EmailConfig       { return cfg.Email }
func ProvideSenderConfig(cfg Config) config.SenderConfig     { return cfg.Sender }
func ProvideRabbitMQConfig(cfg Config) config.RabbitMQConfig { return cfg.RabbitMQ }

func ProvideRabbitMQConnection(cfg config.RabbitMQConfig) (*sharedRabbitmq.Connection, func(), error) {
	return sharedRabbitmq.NewConnection(cfg.URL)
}

type App struct {
	NotificationConsumer *notifRabbitmq.NotificationConsumer
	ConfirmationConsumer *notifRabbitmq.ConfirmationConsumer
}

func InitializeApp(ctx context.Context, cfg Config) (*App, func(), error) {
	wire.Build(
		ProvideEmailConfig,
		ProvideSenderConfig,
		ProvideRabbitMQConfig,
		ProvideSendTimeout,
		EmailSet,
		NotificationSet,
		BrokerSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
