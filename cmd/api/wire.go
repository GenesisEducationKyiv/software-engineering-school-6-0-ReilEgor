//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	notifService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/service"
	email2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/infrastructure/clients/email"
	subscriptionService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/cache/redis"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/storage/postgres"
	subscriptionRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	subscriptionUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/usecase"
	postgres2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/repository/postgres"
	grpc2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/transport/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/transport/http"
	usecase5 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/usecase"
	trackingRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/repository"
	trackingService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/service"
	trackingUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	github2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/clients/github"
	trackingPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/repository/postgres"
	usecase2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/usecase"
)

func ProvideDBConfig(cfg Config) config.DBConfig         { return cfg.DB }
func ProvideRedisConfig(cfg Config) config.RedisConfig   { return cfg.Redis }
func ProvideEmailConfig(cfg Config) config.EmailConfig   { return cfg.Email }
func ProvideGitHubConfig(cfg Config) config.GitHubConfig { return cfg.GitHub }
func ProvideHTTPConfig(cfg Config) config.HTTPConfig     { return cfg.HTTP }
func ProvideAppConfig(cfg Config) config.AppConfig       { return cfg.App }
func ProvideBaseURL(cfg config.AppConfig) string         { return cfg.BaseURL }

var UseCaseSet = wire.NewSet(
	usecase2.NewRepositoryUseCase,
	usecase5.NewUserUseCase,
	wire.Bind(new(trackingUsecase.RepositoryUseCase), new(*usecase2.RepositoryUseCase)),
	wire.Bind(new(subscriptionUsecase.UserUseCase), new(*usecase5.UserUseCase)),
)

var RepositorySet = wire.NewSet(
	sharedPostgres.New,
	trackingPostgres.NewRepositoryRepository,
	postgres2.NewSubscriptionRepository,
	postgres2.NewUserRepository,
	wire.Bind(new(sharedPostgres.PgxInterface), new(*pgxpool.Pool)),
	wire.Bind(new(trackingRepo.RepositoryRepository), new(*trackingPostgres.RepositoryRepository)),
	wire.Bind(new(subscriptionRepo.SubscriptionRepository), new(*postgres2.SubscriptionRepository)),
	wire.Bind(new(subscriptionRepo.UserRepository), new(*postgres2.UserRepository)),
)

func ProvideCachedClient(
	c *github2.GitHubClient,
	cache trackingService.Cache,
) trackingService.GitHubClient {
	return github2.NewCachedGitHubClient(c, cache)
}

var GitHubSet = wire.NewSet(
	github2.NewGitHubClient,
	ProvideCachedClient,
)

var CacheSet = wire.NewSet(
	redis.NewRedisClient,
	redis.NewCache,
	wire.Bind(new(trackingService.Cache), new(*redis.Cache)),
)

var EmailSet = wire.NewSet(
	email2.NewSMTPClient,
	email2.NewEmailManager,
	wire.Bind(new(notifService.EmailService), new(*email2.EmailManager)),
	wire.Bind(new(notifService.EmailSender), new(*email2.SMTPClient)),
	wire.Bind(new(subscriptionService.ConfirmationSender), new(*email2.EmailManager)),
)

var GrpcSet = wire.NewSet(
	grpc2.NewSubscriptionHandler,
	grpc2.NewGrpcServer,
)

type App struct {
	HTTPServer *http.GinServer
	GrpcServer *grpc.Server
}

func InitializeApp(ctx context.Context, cfg Config) (*App, func(), error) {
	wire.Build(
		ProvideDBConfig,
		ProvideRedisConfig,
		ProvideEmailConfig,
		ProvideGitHubConfig,
		ProvideHTTPConfig,
		ProvideAppConfig,
		ProvideBaseURL,
		GitHubSet,
		EmailSet,
		RepositorySet,
		UseCaseSet,
		CacheSet,
		http.NewGinServer,
		GrpcSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
