package main

import (
	"github.com/google/wire"

	trackingRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/repository"
	trackingService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/service"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	githubInfra "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/clients/github"
	trackingPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/repository/postgres"
	trackingUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/usecase"
	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/cache"
)

func ProvideCachedClient(c *githubInfra.GitHubClient, cache sharedcache.Cache) trackingService.GitHubClient {
	return githubInfra.NewCachedGitHubClient(c, cache)
}

var GitHubSet = wire.NewSet(
	githubInfra.NewGitHubClient,
	ProvideCachedClient,
)

var TrackingRepositorySet = wire.NewSet(
	trackingPostgres.NewRepositoryRepository,
	wire.Bind(new(trackingRepo.RepositoryRepository), new(*trackingPostgres.RepositoryRepository)),
)

var TrackingUseCaseSet = wire.NewSet(
	trackingUsecase.NewRepositoryUseCase,
	wire.Bind(new(trackingDomainUsecase.RepositoryUseCase), new(*trackingUsecase.RepositoryUseCase)),
)
