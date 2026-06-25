package main

import (
	"github.com/google/wire"
	"google.golang.org/grpc"

	nethttp "net/http"

	trackingPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/port"
	trackingRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/repository"
	trackingService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/service"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	trackingRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/broker/rabbitmq"
	githubClient "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/clients/github"
	trackingGrpc "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/grpc"
	trackingHttp "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/http"
	trackingPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/repository/postgres"
	trackingUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/cache"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/storage/postgres"
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
	trackingPostgres.NewOutboxRepository,
	sharedPostgres.NewTransactor,
	wire.Bind(new(trackingRepo.RepositoryRepository), new(*trackingPostgres.RepositoryRepository)),
	wire.Bind(new(trackingPort.RepositoryReader), new(*trackingPostgres.RepositoryRepository)),
	wire.Bind(new(trackingRepo.OutboxRepository), new(*trackingPostgres.OutboxRepository)),
	wire.Bind(new(trackingRepo.Transactor), new(*sharedPostgres.Transactor)),
)

func ProvideTagUpdatedGRPCPublisher(
	conn *grpc.ClientConn,
	cfg config.SubscriptionClientConfig,
) *trackingGrpc.TagUpdatedPublisher {
	return trackingGrpc.NewTagUpdatedPublisher(conn, cfg.APIKey)
}

func ProvideTagUpdatedHTTPPublisher(
	client *nethttp.Client,
	cfg config.SubscriptionClientConfig,
) *trackingHttp.TagUpdatedPublisher {
	return trackingHttp.NewTagUpdatedPublisher(client, cfg.SubscriptionHTTPAddr, cfg.APIKey)
}

func ProvideTagUpdatedPublisher(
	cfg config.SubscriptionClientConfig,
	grpcPub *trackingGrpc.TagUpdatedPublisher,
	httpPub *trackingHttp.TagUpdatedPublisher,
) trackingPort.TagUpdatedPublisher {
	if cfg.TagPublisherType == "http" {
		return httpPub
	}
	return grpcPub
}

var BrokerSet = wire.NewSet(
	ProvideRabbitMQConnection,
	trackingRabbitmq.NewPublisher,
	trackingRabbitmq.NewSubscriptionActivatedConsumer,
	trackingRabbitmq.NewUnsubscriptionActivatedConsumer,
	ProvideTagUpdatedGRPCPublisher,
	ProvideTagUpdatedHTTPPublisher,
	ProvideTagUpdatedPublisher,
	wire.Bind(new(trackingPort.NotificationPublisher), new(*trackingRabbitmq.Publisher)),
)

var UseCaseSet = wire.NewSet(
	trackingUsecase.NewRepositoryUseCase,
	trackingUsecase.NewReleaseProcessor,
	wire.Bind(new(trackingDomainUsecase.RepositoryUseCase), new(*trackingUsecase.RepositoryUseCase)),
	wire.Bind(new(trackingDomainUsecase.ReleaseProcessorUseCase), new(*trackingUsecase.ReleaseProcessor)),
)
