package main

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	"github.com/google/wire"
	"google.golang.org/grpc"

	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/cache"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"
	nethttp "net/http"

	repository2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/repository"
	trackingService "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/service"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/usecase"
	trackingGrpc "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/infrastructure/adapter/grpc"
	trackingHttp "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/infrastructure/adapter/http"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/infrastructure/broker/rabbitmq"
	github2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/infrastructure/clients/github"
	postgres2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/repository/postgres"
	rabbitmq2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/transport/broker/rabbitmq"
	usecase2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/usecase"
)

func ProvideCachedClient(c *github2.GitHubClient, cache sharedcache.Cache) trackingService.GitHubClient {
	return github2.NewCachedGitHubClient(c, cache)
}

var GitHubSet = wire.NewSet(
	github2.NewGitHubClient,
	ProvideCachedClient,
)

var TrackingRepositorySet = wire.NewSet(
	postgres2.NewRepositoryRepository,
	postgres2.NewOutboxRepository,
	sharedPostgres.NewTransactor,
	wire.Bind(new(repository2.RepositoryRepository), new(*postgres2.RepositoryRepository)),
	wire.Bind(new(usecase2.RepositoryReader), new(*postgres2.RepositoryRepository)),
	wire.Bind(new(repository2.OutboxRepository), new(*postgres2.OutboxRepository)),
	wire.Bind(new(repository2.Transactor), new(*sharedPostgres.Transactor)),
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
) usecase2.TagUpdatedPublisher {
	if cfg.TagPublisherType == "http" {
		return httpPub
	}
	return grpcPub
}

var BrokerSet = wire.NewSet(
	ProvideRabbitMQConnection,
	rabbitmq.NewPublisher,
	rabbitmq2.NewSubscriptionActivatedConsumer,
	rabbitmq2.NewUnsubscriptionActivatedConsumer,
	ProvideTagUpdatedGRPCPublisher,
	ProvideTagUpdatedHTTPPublisher,
	ProvideTagUpdatedPublisher,
	wire.Bind(new(usecase2.NotificationPublisher), new(*rabbitmq.Publisher)),
)

var UseCaseSet = wire.NewSet(
	usecase2.NewRepositoryUseCase,
	usecase2.NewReleaseProcessor,
	wire.Bind(new(trackingDomainUsecase.RepositoryUseCase), new(*usecase2.RepositoryUseCase)),
	wire.Bind(new(trackingDomainUsecase.ReleaseProcessorUseCase), new(*usecase2.ReleaseProcessor)),
)
