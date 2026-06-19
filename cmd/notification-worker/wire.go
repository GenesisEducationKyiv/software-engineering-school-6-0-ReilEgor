//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"

	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	trackingRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/broker/rabbitmq"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
)

func ProvideTrackingDBConfig(cfg Config) config.TrackingDBConfig { return cfg.TrackingDB }
func ProvideRedisConfig(cfg Config) config.RedisConfig           { return cfg.Redis }
func ProvideGitHubConfig(cfg Config) config.GitHubConfig         { return cfg.GitHub }
func ProvideRabbitMQConfig(cfg Config) config.RabbitMQConfig     { return cfg.RabbitMQ }

func ProvideRabbitMQConnection(cfg config.RabbitMQConfig) (*sharedRabbitmq.Connection, func(), error) {
	return sharedRabbitmq.NewConnection(cfg.URL)
}

type App struct {
	ReleaseProcessor              trackingDomainUsecase.ReleaseProcessorUseCase
	SubscriptionActivatedConsumer *trackingRabbitmq.SubscriptionActivatedConsumer
	UnsubscriptionActivatedConsumer *trackingRabbitmq.UnsubscriptionActivatedConsumer
}

func InitializeApp(ctx context.Context, cfg Config) (*App, func(), error) {
	wire.Build(
		ProvideTrackingDBConfig,
		ProvideRedisConfig,
		ProvideGitHubConfig,
		ProvideRabbitMQConfig,
		SharedSet,
		CacheSet,
		GitHubSet,
		TrackingRepositorySet,
		SubscriptionRepositorySet,
		BrokerSet,
		UseCaseSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
