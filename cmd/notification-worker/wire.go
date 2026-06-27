//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/port"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/usecase"
	email2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/infrastructure/clients/email"
	usecaseRealization "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/cache/redis"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/storage/postgres"
	repository4 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	postgres2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/repository/postgres"
	repository3 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/repository"
	service2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/service"
	usecase4 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	github2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/clients/github"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/repository/postgres"
	usecase2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/usecase"
)

func ProvideDBConfig(cfg Config) config.DBConfig         { return cfg.DB }
func ProvideRedisConfig(cfg Config) config.RedisConfig   { return cfg.Redis }
func ProvideEmailConfig(cfg Config) config.EmailConfig   { return cfg.Email }
func ProvideGitHubConfig(cfg Config) config.GitHubConfig { return cfg.GitHub }
func ProvideAppConfig(cfg Config) config.AppConfig       { return cfg.App }
func ProvideWorkerConfig(cfg Config) config.WorkerConfig { return cfg.Worker }
func ProvideBaseURL(cfg config.AppConfig) string         { return cfg.BaseURL }

var UseCaseSet = wire.NewSet(
	usecase2.NewRepositoryUseCase,
	usecaseRealization.NewNotificationUseCase,
	wire.Bind(new(usecase4.RepositoryUseCase), new(*usecase2.RepositoryUseCase)),
	wire.Bind(new(usecase.NotificationUseCase), new(*usecaseRealization.NotificationUseCase)),
	wire.Bind(new(port.UpdateChecker), new(*usecase2.RepositoryUseCase)),
)

var RepositorySet = wire.NewSet(
	sharedPostgres.New,
	postgres.NewRepositoryRepository,
	postgres2.NewSubscriptionRepository,
	wire.Bind(new(sharedPostgres.PgxInterface), new(*pgxpool.Pool)),
	wire.Bind(new(repository3.RepositoryRepository), new(*postgres.RepositoryRepository)),
	wire.Bind(new(repository4.SubscriptionRepository), new(*postgres2.SubscriptionRepository)),
	wire.Bind(new(port.RepositoryReader), new(*postgres.RepositoryRepository)),
	wire.Bind(new(port.SubscriberReader), new(*postgres2.SubscriptionRepository)),
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

type App struct {
	NotificationUseCase usecase.NotificationUseCase
}

func InitializeApp(ctx context.Context, cfg Config) (*App, func(), error) {
	wire.Build(
		ProvideDBConfig,
		ProvideRedisConfig,
		ProvideEmailConfig,
		ProvideGitHubConfig,
		ProvideAppConfig,
		ProvideWorkerConfig,
		ProvideBaseURL,
		GitHubSet,
		EmailSet,
		RepositorySet,
		UseCaseSet,
		CacheSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
