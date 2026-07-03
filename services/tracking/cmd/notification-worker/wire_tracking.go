package main

import (
	"net/http"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	"github.com/google/wire"
	"google.golang.org/grpc"

	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/cache"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/repository"
	trackingService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/service"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/usecase"
	trackingGrpc "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/infrastructure/adapter/grpc"
	trackingHttp "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/infrastructure/adapter/http"
	rabbitmqPublisher "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/infrastructure/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/infrastructure/clients/github"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/repository/postgres"
	rabbitmqConsumer "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/transport/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/usecase"
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
	wire.Bind(new(usecase.RepositoryReader), new(*postgres.RepositoryRepository)),
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
	client *http.Client,
	cfg config.SubscriptionClientConfig,
) *trackingHttp.TagUpdatedPublisher {
	return trackingHttp.NewTagUpdatedPublisher(client, cfg.SubscriptionHTTPAddr, cfg.APIKey)
}

func ProvideTagUpdatedPublisher(
	cfg config.SubscriptionClientConfig,
	grpcPub *trackingGrpc.TagUpdatedPublisher,
	httpPub *trackingHttp.TagUpdatedPublisher,
) usecase.TagUpdatedPublisher {
	if cfg.TagPublisherType == "http" {
		return httpPub
	}
	return grpcPub
}

var BrokerSet = wire.NewSet(
	ProvideRabbitMQConnection,
	rabbitmqPublisher.NewPublisher,
	rabbitmqConsumer.NewSubscriptionActivatedConsumer,
	rabbitmqConsumer.NewUnsubscriptionActivatedConsumer,
	ProvideTagUpdatedGRPCPublisher,
	ProvideTagUpdatedHTTPPublisher,
	ProvideTagUpdatedPublisher,
	wire.Bind(new(usecase.NotificationPublisher), new(*rabbitmqPublisher.Publisher)),
)

var UseCaseSet = wire.NewSet(
	usecase.NewRepositoryUseCase,
	usecase.NewReleaseProcessor,
	wire.Bind(new(trackingDomainUsecase.RepositoryUseCase), new(*usecase.RepositoryUseCase)),
	wire.Bind(new(trackingDomainUsecase.ReleaseProcessorUseCase), new(*usecase.ReleaseProcessor)),
)
