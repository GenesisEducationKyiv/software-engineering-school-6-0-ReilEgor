//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/config"
	repositoryInterface "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/repository"
	servicesInterface "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
	usecaseInterface "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/usecase"
	cacheRealization "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/infrastructure/cache/redis"
	servicesRealizationEmail "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/infrastructure/clients/email"
	servicesRealizationGitHub "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/infrastructure/clients/github"
	repository "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/infrastructure/storage/postgres"
	repositoryRealization "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/repository/postgres"
	grpcTransport "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/http"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/http/handlers"
	usecaseRealization "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/usecase"
)

var UseCaseSet = wire.NewSet(
	usecaseRealization.NewRepositoryUseCase,
	usecaseRealization.NewNotificationUseCase,
	usecaseRealization.NewUserUseCase,
	wire.Bind(new(usecaseInterface.RepositoryUseCase), new(*usecaseRealization.RepositoryUseCase)),
	wire.Bind(new(usecaseInterface.NotificationUseCase), new(*usecaseRealization.NotificationUseCase)),
	wire.Bind(new(usecaseInterface.UserUseCase), new(*usecaseRealization.UserUseCase)),
)

var RepositorySet = wire.NewSet(
	repository.New,
	repositoryRealization.NewRepositoryRepository,
	repositoryRealization.NewSubscriptionRepository,
	repositoryRealization.NewUserRepository,
	wire.Bind(new(repositoryRealization.PgxInterface), new(*pgxpool.Pool)),
	wire.Bind(new(repositoryInterface.RepositoryRepository), new(*repositoryRealization.RepositoryRepository)),
	wire.Bind(new(repositoryInterface.SubscriptionRepository), new(*repositoryRealization.SubscriptionRepository)),
	wire.Bind(new(repositoryInterface.UserRepository), new(*repositoryRealization.UserRepository)),
)

func ProvideCachedClient(
	c *servicesRealizationGitHub.GitHubClient,
	cache servicesInterface.Cache,
) servicesInterface.GitHubClient {
	return servicesRealizationGitHub.NewCachedGitHubClient(c, cache)
}

var GitHubSet = wire.NewSet(
	servicesRealizationGitHub.NewGitHubClient,
	ProvideCachedClient,
)

var RestSet = wire.NewSet(
	http.NewGinServer,
	handlers.NewHandler,
)

var CacheSet = wire.NewSet(
	cacheRealization.NewRedisClient,
	cacheRealization.NewCache,
	wire.Bind(new(servicesInterface.Cache), new(*cacheRealization.Cache)),
)

var EmailSet = wire.NewSet(
	servicesRealizationEmail.NewSMTPClient,
	servicesRealizationEmail.NewEmailManager,
	wire.Bind(new(servicesInterface.EmailService), new(*servicesRealizationEmail.EmailManager)),
	wire.Bind(new(servicesInterface.EmailSender), new(*servicesRealizationEmail.SMTPClient)),
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
	NotificationUseCase usecaseInterface.NotificationUseCase
}

func InitializeApp(
	ctx context.Context,
	redisHost config.RedisHostType,
	redisPort config.RedisPortType,
	redisPassword config.RedisPasswordType,
	redisDB int,
	dsn config.DSNType,
	emailHost config.EmailHostType,
	emailPort config.EmailPortType,
	emailPassword config.EmailPasswordType,
	emailFrom config.EmailFromType,
	emailUser config.EmailUserType,
	apiKey config.APIKeyType,
	githubToken config.GitHubTokenType,
	baseURL config.AppBaseURLType,
) (*App, func(), error) {
	wire.Build(
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
