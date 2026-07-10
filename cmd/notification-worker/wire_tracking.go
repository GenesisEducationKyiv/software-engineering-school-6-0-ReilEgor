package main

import (
	"github.com/google/wire"

	trackingPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/port"
	trackingRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/repository"
	trackingService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/service"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	trackingRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/broker/rabbitmq"
	githubClient "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/clients/github"
	trackingPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/repository/postgres"
	trackingUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/usecase"
	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/cache"
)

func ProvideCachedClient(c *githubClient.GitHubClient, cache sharedcache.Cache) trackingService.GitHubClient {
	return githubClient.NewCachedGitHubClient(c, cache)
}

var GitHubSet = wire.NewSet(
	githubClient.NewGitHubClient,
	ProvideCachedClient,
)

var TrackingRepositorySet = wire.NewSet(
	trackingPostgres.NewRepositoryRepository,
	wire.Bind(new(trackingRepo.RepositoryRepository), new(*trackingPostgres.RepositoryRepository)),
	wire.Bind(new(trackingPort.RepositoryReader), new(*trackingPostgres.RepositoryRepository)),
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
