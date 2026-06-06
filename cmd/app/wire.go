//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/infrastructure/clients/github"
	repositoryRealization "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/repository/postgres"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/cache/redis"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
	repository2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/email"
	repository "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/storage/postgres"
	grpcTransport "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/http"
	usecaseRealization "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/usecase"
)

func ProvideDBConfig(cfg config.Config) config.DBConfig         { return cfg.DB }
func ProvideRedisConfig(cfg config.Config) config.RedisConfig   { return cfg.Redis }
func ProvideEmailConfig(cfg config.Config) config.EmailConfig   { return cfg.Email }
func ProvideGitHubConfig(cfg config.Config) config.GitHubConfig { return cfg.GitHub }
func ProvideHTTPConfig(cfg config.Config) config.HTTPConfig     { return cfg.HTTP }
func ProvideAppConfig(cfg config.Config) config.AppConfig       { return cfg.App }
func ProvideWorkerConfig(cfg config.Config) config.WorkerConfig { return cfg.Worker }
func ProvideBaseURL(cfg config.AppConfig) string                { return cfg.BaseURL }

var UseCaseSet = wire.NewSet(
	usecaseRealization.NewRepositoryUseCase,
	usecaseRealization.NewNotificationUseCase,
	usecaseRealization.NewUserUseCase,
	wire.Bind(new(usecase.RepositoryUseCase), new(*usecaseRealization.RepositoryUseCase)),
	wire.Bind(new(usecase.NotificationUseCase), new(*usecaseRealization.NotificationUseCase)),
	wire.Bind(new(usecase.UserUseCase), new(*usecaseRealization.UserUseCase)),
)

var RepositorySet = wire.NewSet(
	repository.New,
	repositoryRealization.NewRepositoryRepository,
	repositoryRealization.NewSubscriptionRepository,
	repositoryRealization.NewUserRepository,
	wire.Bind(new(repository.PgxInterface), new(*pgxpool.Pool)),
	wire.Bind(new(repository2.RepositoryRepository), new(*repositoryRealization.RepositoryRepository)),
	wire.Bind(new(repository2.SubscriptionRepository), new(*repositoryRealization.SubscriptionRepository)),
	wire.Bind(new(repository2.UserRepository), new(*repositoryRealization.UserRepository)),
)

func ProvideCachedClient(
	c *github.GitHubClient,
	cache service.Cache,
) service.GitHubClient {
	return github.NewCachedGitHubClient(c, cache)
}

var GitHubSet = wire.NewSet(
	github.NewGitHubClient,
	ProvideCachedClient,
)

var RestSet = wire.NewSet(
	http.NewGinServer,
)

var CacheSet = wire.NewSet(
	redis.NewRedisClient,
	redis.NewCache,
	wire.Bind(new(service.Cache), new(*redis.Cache)),
)

var EmailSet = wire.NewSet(
	email.NewSMTPClient,
	email.NewEmailManager,
	wire.Bind(new(service.EmailService), new(*email.EmailManager)),
	wire.Bind(new(service.EmailSender), new(*email.SMTPClient)),
)

var ServicesSet = wire.NewSet(
	GitHubSet,
	EmailSet,
)

var GrpcSet = wire.NewSet(
	grpcTransport.NewSubscriptionHandler,
	grpcTransport.NewGrpcServer,
)

type App struct {
	HTTPServer          *http.GinServer
	GrpcServer          *grpc.Server
	NotificationUseCase usecase.NotificationUseCase
}

func InitializeApp(ctx context.Context, cfg config.Config) (*App, func(), error) {
	wire.Build(
		ProvideDBConfig,
		ProvideRedisConfig,
		ProvideEmailConfig,
		ProvideGitHubConfig,
		ProvideHTTPConfig,
		ProvideAppConfig,
		ProvideWorkerConfig,
		ProvideBaseURL,
		ServicesSet,
		RepositorySet,
		UseCaseSet,
		CacheSet,
		RestSet,
		GrpcSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
