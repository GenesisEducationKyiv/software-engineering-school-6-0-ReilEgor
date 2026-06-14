package main

import (
	"time"

	"github.com/google/wire"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/usecase"
	notifRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/infrastructure/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/infrastructure/clients/email"
	notifUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
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
	notifRabbitmq.NewConsumer,
	notifRabbitmq.NewConfirmationConsumer,
)
