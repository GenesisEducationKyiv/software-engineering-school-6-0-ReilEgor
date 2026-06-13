//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/cache/redis"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/cache"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/storage/postgres"
	subscriptionRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	subscriptionPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/port"
	subscriptionUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/usecase"
	subRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/infrastructure/broker/rabbitmq"
	subPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/repository/postgres"
	subGrpc "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/transport/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/transport/http"
	subUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/usecase"
	trackingRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/repository"
	trackingService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/service"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	githubInfra "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/clients/github"
	trackingPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/repository/postgres"
	trackingUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/usecase"
)

func ProvideDBConfig(cfg Config) config.DBConfig         { return cfg.DB }
func ProvideRedisConfig(cfg Config) config.RedisConfig   { return cfg.Redis }
func ProvideGitHubConfig(cfg Config) config.GitHubConfig { return cfg.GitHub }
func ProvideHTTPConfig(cfg Config) config.HTTPConfig     { return cfg.HTTP }
func ProvideAppConfig(cfg Config) config.AppConfig       { return cfg.App }
func ProvideRabbitMQConfig(cfg Config) config.RabbitMQConfig { return cfg.RabbitMQ }

func ProvideRabbitMQConnection(cfg config.RabbitMQConfig) (*sharedRabbitmq.Connection, func(), error) {
	return sharedRabbitmq.NewConnection(cfg.URL)
}

var UseCaseSet = wire.NewSet(
	trackingUsecase.NewRepositoryUseCase,
	subUsecase.NewUserUseCase,
	wire.Bind(new(trackingDomainUsecase.RepositoryUseCase), new(*trackingUsecase.RepositoryUseCase)),
	wire.Bind(new(subscriptionPort.RepositoryUseCase), new(*trackingUsecase.RepositoryUseCase)),
	wire.Bind(new(subscriptionUsecase.UserUseCase), new(*subUsecase.UserUseCase)),
)

var RepositorySet = wire.NewSet(
	sharedPostgres.New,
	trackingPostgres.NewRepositoryRepository,
	subPostgres.NewSubscriptionRepository,
	subPostgres.NewUserRepository,
	wire.Bind(new(sharedPostgres.PgxInterface), new(*pgxpool.Pool)),
	wire.Bind(new(trackingRepo.RepositoryRepository), new(*trackingPostgres.RepositoryRepository)),
	wire.Bind(new(subscriptionRepo.SubscriptionRepository), new(*subPostgres.SubscriptionRepository)),
	wire.Bind(new(subscriptionRepo.UserRepository), new(*subPostgres.UserRepository)),
)

func ProvideCachedClient(
	c *githubInfra.GitHubClient,
	cache sharedcache.Cache,
) trackingService.GitHubClient {
	return githubInfra.NewCachedGitHubClient(c, cache)
}

var GitHubSet = wire.NewSet(
	githubInfra.NewGitHubClient,
	ProvideCachedClient,
)

var CacheSet = wire.NewSet(
	redis.NewRedisClient,
	redis.NewCache,
	wire.Bind(new(sharedcache.Cache), new(*redis.Cache)),
)

var BrokerSet = wire.NewSet(
	ProvideRabbitMQConnection,
	subRabbitmq.NewPublisher,
	wire.Bind(new(subscriptionPort.ConfirmationSender), new(*subRabbitmq.Publisher)),
)

var GrpcSet = wire.NewSet(
	subGrpc.NewSubscriptionHandler,
	subGrpc.NewGrpcServer,
)

type App struct {
	HTTPServer *http.GinServer
	GrpcServer *grpc.Server
}

func InitializeApp(ctx context.Context, cfg Config) (*App, func(), error) {
	wire.Build(
		ProvideDBConfig,
		ProvideRedisConfig,
		ProvideGitHubConfig,
		ProvideHTTPConfig,
		ProvideAppConfig,
		ProvideRabbitMQConfig,
		GitHubSet,
		BrokerSet,
		RepositorySet,
		UseCaseSet,
		CacheSet,
		http.NewGinServer,
		GrpcSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
