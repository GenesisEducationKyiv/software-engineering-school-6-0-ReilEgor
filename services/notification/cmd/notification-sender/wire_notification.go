package main

import (
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	"github.com/google/wire"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/usecase"
	infraRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/infrastructure/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/infrastructure/clients/email"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/transport/broker/rabbitmq"
	notifUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/usecase"
)

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
	rabbitmq.NewNotificationConsumer,
	rabbitmq.NewConfirmationConsumer,
	infraRabbitmq.NewSagaResultPublisher,
	wire.Bind(new(rabbitmq.SagaResultPublisher), new(*infraRabbitmq.SagaResultPublisher)),
)
