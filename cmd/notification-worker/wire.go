//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"

	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
)

func ProvideDBConfig(cfg Config) config.DBConfig             { return cfg.DB }
func ProvideRedisConfig(cfg Config) config.RedisConfig       { return cfg.Redis }
func ProvideGitHubConfig(cfg Config) config.GitHubConfig     { return cfg.GitHub }
func ProvideRabbitMQConfig(cfg Config) config.RabbitMQConfig { return cfg.RabbitMQ }

func ProvideRabbitMQConnection(cfg config.RabbitMQConfig) (*sharedRabbitmq.Connection, func(), error) {
	return sharedRabbitmq.NewConnection(cfg.URL)
}

type App struct {
	ReleaseProcessor trackingDomainUsecase.ReleaseProcessorUseCase
}

func InitializeApp(ctx context.Context, cfg Config) (*App, func(), error) {
	wire.Build(
		ProvideDBConfig,
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
