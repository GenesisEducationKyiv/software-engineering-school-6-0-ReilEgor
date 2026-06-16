//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"google.golang.org/grpc"

	subscriptionPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/port"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/transport/http"
	trackingUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/usecase"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
)

func ProvideSubscriptionDBConfig(cfg Config) config.SubscriptionDBConfig { return cfg.SubscriptionDB }
func ProvideRedisConfig(cfg Config) config.RedisConfig                   { return cfg.Redis }
func ProvideGitHubConfig(cfg Config) config.GitHubConfig                 { return cfg.GitHub }
func ProvideHTTPConfig(cfg Config) config.HTTPConfig                     { return cfg.HTTP }
func ProvideAppConfig(cfg Config) config.AppConfig                       { return cfg.App }
func ProvideRabbitMQConfig(cfg Config) config.RabbitMQConfig             { return cfg.RabbitMQ }

func ProvideRabbitMQConnection(cfg config.RabbitMQConfig) (*sharedRabbitmq.Connection, func(), error) {
	return sharedRabbitmq.NewConnection(cfg.URL)
}

type App struct {
	HTTPServer *http.GinServer
	GrpcServer *grpc.Server
}

func InitializeApp(ctx context.Context, cfg Config) (*App, func(), error) {
	wire.Build(
		ProvideSubscriptionDBConfig,
		ProvideRedisConfig,
		ProvideGitHubConfig,
		ProvideHTTPConfig,
		ProvideAppConfig,
		ProvideRabbitMQConfig,
		SharedSet,
		CacheSet,
		GitHubSet,
		TrackingRepositorySet,
		TrackingUseCaseSet,
		SubscriptionRepositorySet,
		SubscriptionUseCaseSet,
		BrokerSet,
		GrpcSet,
		http.NewGinServer,
		wire.Bind(new(subscriptionPort.RepositoryUseCase), new(*trackingUsecase.RepositoryUseCase)),
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
