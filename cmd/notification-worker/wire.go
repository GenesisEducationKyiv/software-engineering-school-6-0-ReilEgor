//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	subDomainRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	subPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/repository/postgres"
	trackingPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/port"
	trackingDomainRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/repository"
	trackingDomainService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/service"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	trackingRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/broker/rabbitmq"
	githubClient "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/clients/github"
	trackingPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/repository/postgres"
	trackingUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/usecase"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	redis2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/cache/redis"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/cache"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/storage/postgres"
)

func ProvideDBConfig(cfg Config) config.DBConfig             { return cfg.DB }
func ProvideRedisConfig(cfg Config) config.RedisConfig       { return cfg.Redis }
func ProvideGitHubConfig(cfg Config) config.GitHubConfig     { return cfg.GitHub }
func ProvideRabbitMQConfig(cfg Config) config.RabbitMQConfig { return cfg.RabbitMQ }

func ProvideRabbitMQConnection(cfg config.RabbitMQConfig) (*sharedRabbitmq.Connection, func(), error) {
	return sharedRabbitmq.NewConnection(cfg.URL)
}

func ProvideCachedClient(c *githubClient.GitHubClient, cache sharedcache.Cache) trackingDomainService.GitHubClient {
	return githubClient.NewCachedGitHubClient(c, cache)
}

var GitHubSet = wire.NewSet(
	githubClient.NewGitHubClient,
	ProvideCachedClient,
)

var CacheSet = wire.NewSet(
	redis2.NewRedisClient,
	redis2.NewCache,
	wire.Bind(new(sharedcache.Cache), new(*redis2.Cache)),
)

var RepositorySet = wire.NewSet(
	postgres.New,
	trackingPostgres.NewRepositoryRepository,
	subPostgres.NewSubscriptionRepository,
	wire.Bind(new(postgres.PgxInterface), new(*pgxpool.Pool)),
	wire.Bind(new(trackingDomainRepo.RepositoryRepository), new(*trackingPostgres.RepositoryRepository)),
	wire.Bind(new(subDomainRepo.SubscriptionRepository), new(*subPostgres.SubscriptionRepository)),
	wire.Bind(new(trackingPort.RepositoryReader), new(*trackingPostgres.RepositoryRepository)),
	wire.Bind(new(trackingPort.SubscriberReader), new(*subPostgres.SubscriptionRepository)),
)

var BrokerSet = wire.NewSet(
	ProvideRabbitMQConnection,
	trackingRabbitmq.NewPublisher,
	wire.Bind(new(trackingPort.NotificationPublisher), new(*trackingRabbitmq.Publisher)),
)

var UseCaseSet = wire.NewSet(
	trackingUsecase.NewRepositoryUseCase,
	trackingUsecase.NewReleaseProcessor,
	wire.Bind(new(trackingDomainUsecase.RepositoryUseCase), new(*trackingUsecase.RepositoryUseCase)),
	wire.Bind(new(trackingDomainUsecase.ReleaseProcessorUseCase), new(*trackingUsecase.ReleaseProcessor)),
)

type App struct {
	ReleaseProcessor trackingDomainUsecase.ReleaseProcessorUseCase
}

func InitializeApp(ctx context.Context, cfg Config) (*App, func(), error) {
	wire.Build(
		ProvideDBConfig,
		ProvideRedisConfig,
		ProvideGitHubConfig,
		ProvideRabbitMQConfig,
		GitHubSet,
		CacheSet,
		RepositorySet,
		BrokerSet,
		UseCaseSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
