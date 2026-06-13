//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	trackingPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/port"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	trackingDomainService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/service"
	trackingDomainRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/repository"
	trackingRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/broker/rabbitmq"
	githubClient "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/clients/github"
	trackingPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/repository/postgres"
	trackingUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/usecase"

	subDomainRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	subPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/repository/postgres"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/cache/redis"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/broker/rabbitmq"
	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/cache"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/storage/postgres"
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
	redis.NewRedisClient,
	redis.NewCache,
	wire.Bind(new(sharedcache.Cache), new(*redis.Cache)),
)

var RepositorySet = wire.NewSet(
	sharedPostgres.New,
	trackingPostgres.NewRepositoryRepository,
	subPostgres.NewSubscriptionRepository,
	wire.Bind(new(sharedPostgres.PgxInterface), new(*pgxpool.Pool)),
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
