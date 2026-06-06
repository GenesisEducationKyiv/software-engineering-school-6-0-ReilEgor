//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/usecase"
	email2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/infrastructure/clients/email"
	usecaseRealization "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/cache/redis"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
	repository "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/storage/postgres"
	repository4 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	usecase3 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/usecase"
	postgres2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/repository/postgres"
	grpc2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/transport/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/transport/http"
	usecase5 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/usecase"
	repository3 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/repository"
	service2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/service"
	usecase4 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	github2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/clients/github"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/repository/postgres"
	usecase2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/usecase"
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
	usecase2.NewRepositoryUseCase,
	usecaseRealization.NewNotificationUseCase,
	usecase5.NewUserUseCase,
	wire.Bind(new(usecase4.RepositoryUseCase), new(*usecase2.RepositoryUseCase)),
	wire.Bind(new(usecase.NotificationUseCase), new(*usecaseRealization.NotificationUseCase)),
	wire.Bind(new(usecase3.UserUseCase), new(*usecase5.UserUseCase)),
)

var RepositorySet = wire.NewSet(
	repository.New,
	postgres.NewRepositoryRepository,
	postgres2.NewSubscriptionRepository,
	postgres2.NewUserRepository,
	wire.Bind(new(repository.PgxInterface), new(*pgxpool.Pool)),
	wire.Bind(new(repository3.RepositoryRepository), new(*postgres.RepositoryRepository)),
	wire.Bind(new(repository4.SubscriptionRepository), new(*postgres2.SubscriptionRepository)),
	wire.Bind(new(repository4.UserRepository), new(*postgres2.UserRepository)),
)

func ProvideCachedClient(
	c *github2.GitHubClient,
	cache service2.Cache,
) service2.GitHubClient {
	return github2.NewCachedGitHubClient(c, cache)
}

var GitHubSet = wire.NewSet(
	github2.NewGitHubClient,
	ProvideCachedClient,
)

var RestSet = wire.NewSet(
	http.NewGinServer,
)

var CacheSet = wire.NewSet(
	redis.NewRedisClient,
	redis.NewCache,
	wire.Bind(new(service2.Cache), new(*redis.Cache)),
)

var EmailSet = wire.NewSet(
	email2.NewSMTPClient,
	email2.NewEmailManager,
	wire.Bind(new(service.EmailService), new(*email2.EmailManager)),
	wire.Bind(new(service.EmailSender), new(*email2.SMTPClient)),
)

var ServicesSet = wire.NewSet(
	GitHubSet,
	EmailSet,
)

var GrpcSet = wire.NewSet(
	grpc2.NewSubscriptionHandler,
	grpc2.NewGrpcServer,
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
