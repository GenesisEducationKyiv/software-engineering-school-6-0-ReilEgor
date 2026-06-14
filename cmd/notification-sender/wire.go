//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"time"

	"github.com/google/wire"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/usecase"
	notifRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/infrastructure/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/infrastructure/clients/email"
	notifUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/usecase"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
)

func ProvideEmailConfig(cfg Config) config.EmailConfig       { return cfg.Email }
func ProvideSenderConfig(cfg Config) config.SenderConfig     { return cfg.Sender }
func ProvideRabbitMQConfig(cfg Config) config.RabbitMQConfig { return cfg.RabbitMQ }

func ProvideRabbitMQConnection(cfg config.RabbitMQConfig) (*sharedRabbitmq.Connection, func(), error) {
	return sharedRabbitmq.NewConnection(cfg.URL)
}

func ProvideEmailService(sender service.EmailSender, cfg Config) *email.EmailService {
	return email.NewEmailService(sender, cfg.App.BaseURL)
}

func ProvideSendTimeout(cfg config.SenderConfig) time.Duration {
	return cfg.SendTimeout
}

var EmailSet = wire.NewSet(
	email.NewSMTPClient,
	ProvideEmailService,
	wire.Bind(new(service.EmailSender), new(*email.SMTPClient)),
	wire.Bind(new(service.EmailService), new(*email.EmailService)),
)

var NotificationSet = wire.NewSet(
	notifUsecase.NewNotificationUseCase,
	wire.Bind(new(usecase.NotificationUseCase), new(*notifUsecase.NotificationUseCase)),
)

var BrokerSet = wire.NewSet(
	ProvideRabbitMQConnection,
	notifRabbitmq.NewConsumer,
	notifRabbitmq.NewConfirmationConsumer,
)

type App struct {
	NotificationConsumer *notifRabbitmq.Consumer
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
