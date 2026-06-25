package main

import (
	"time"

	"github.com/google/wire"

	notifPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/domain/port"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/domain/usecase"
	infraRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/infrastructure/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/transport/broker/rabbitmq"
	email2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/infrastructure/clients/email"
	notifUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
)

func ProvideEmailService(sender service.EmailSender, cfg Config) *email2.EmailService {
	return email2.NewEmailService(sender, cfg.App.BaseURL)
}

func ProvideSendTimeout(cfg config.SenderConfig) time.Duration {
	return cfg.SendTimeout
}

var EmailSet = wire.NewSet(
	email2.NewSMTPClient,
	ProvideEmailService,
	wire.Bind(new(service.EmailSender), new(*email2.SMTPClient)),
	wire.Bind(new(service.EmailService), new(*email2.EmailService)),
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
	wire.Bind(new(notifPort.SagaResultPublisher), new(*infraRabbitmq.SagaResultPublisher)),
)
