package main

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	"github.com/google/wire"
	"google.golang.org/grpc"

	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/cache"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"
	nethttp "net/http"

	trackingPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/port"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/repository"
	trackingService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/service"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/infrastructure/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/infrastructure/clients/github"
	trackingGrpc "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/infrastructure/grpc"
	trackingHttp "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/infrastructure/http"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/repository/postgres"
	trackingRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/transport/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/usecase"
)

func ProvideCachedClient(c *github.GitHubClient, cache sharedcache.Cache) trackingService.GitHubClient {
	return github.NewCachedGitHubClient(c, cache)
}

var GitHubSet = wire.NewSet(
	github.NewGitHubClient,
	ProvideCachedClient,
)

var TrackingRepositorySet = wire.NewSet(
	postgres.NewRepositoryRepository,
	postgres.NewOutboxRepository,
	sharedPostgres.NewTransactor,
	wire.Bind(new(repository.RepositoryRepository), new(*postgres.RepositoryRepository)),
	wire.Bind(new(trackingPort.RepositoryReader), new(*postgres.RepositoryRepository)),
	wire.Bind(new(repository.OutboxRepository), new(*postgres.OutboxRepository)),
	wire.Bind(new(repository.Transactor), new(*sharedPostgres.Transactor)),
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
	rabbitmq.NewPublisher,
	trackingRabbitmq.NewSubscriptionActivatedConsumer,
	trackingRabbitmq.NewUnsubscriptionActivatedConsumer,
	ProvideTagUpdatedGRPCPublisher,
	ProvideTagUpdatedHTTPPublisher,
	ProvideTagUpdatedPublisher,
	wire.Bind(new(trackingPort.NotificationPublisher), new(*rabbitmq.Publisher)),
)

var UseCaseSet = wire.NewSet(
	usecase.NewRepositoryUseCase,
	usecase.NewReleaseProcessor,
	wire.Bind(new(trackingDomainUsecase.RepositoryUseCase), new(*usecase.RepositoryUseCase)),
	wire.Bind(new(trackingDomainUsecase.ReleaseProcessorUseCase), new(*usecase.ReleaseProcessor)),
)
