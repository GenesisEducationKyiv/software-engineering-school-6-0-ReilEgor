//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/wire"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	nethttp "net/http"

	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/usecase"
	trackingRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/transport/broker/rabbitmq"
	trackingOutbox "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/infrastructure/outbox"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"
)

func ProvideTrackingDBConfig(cfg Config) config.TrackingDBConfig { return cfg.TrackingDB }
func ProvideRedisConfig(cfg Config) config.RedisConfig           { return cfg.Redis }
func ProvideGitHubConfig(cfg Config) config.GitHubConfig         { return cfg.GitHub }
func ProvideRabbitMQConfig(cfg Config) config.RabbitMQConfig     { return cfg.RabbitMQ }
func ProvideAppConfig(cfg Config) config.AppConfig               { return cfg.App }
func ProvideSubscriptionClientConfig(cfg Config) config.SubscriptionClientConfig {
	return cfg.SubscriptionClient
}

func ProvideRabbitMQConnection(cfg config.RabbitMQConfig) (*sharedRabbitmq.Connection, func(), error) {
	return sharedRabbitmq.NewConnection(cfg.URL)
}

func ProvideSubscriptionGRPCConn(cfg config.SubscriptionClientConfig) (*grpc.ClientConn, func(), error) {
	conn, err := grpc.NewClient(cfg.SubscriptionGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("dial subscription grpc: %w", err)
	}
	return conn, func() { conn.Close() }, nil
}

func ProvideHTTPClient() *nethttp.Client {
	return &nethttp.Client{Timeout: 10 * time.Second}
}

func ProvideOutboxInterval() time.Duration {
	return 5 * time.Second
}

type App struct {
	GrpcServer                      *grpc.Server
	ReleaseProcessor                trackingDomainUsecase.ReleaseProcessorUseCase
	SubscriptionActivatedConsumer   *trackingRabbitmq.SubscriptionActivatedConsumer
	UnsubscriptionActivatedConsumer *trackingRabbitmq.UnsubscriptionActivatedConsumer
	OutboxRelay                     *trackingOutbox.Relay
}

func InitializeApp(ctx context.Context, cfg Config) (*App, func(), error) {
	wire.Build(
		ProvideTrackingDBConfig,
		ProvideRedisConfig,
		ProvideGitHubConfig,
		ProvideRabbitMQConfig,
		ProvideAppConfig,
		ProvideSubscriptionClientConfig,
		ProvideSubscriptionGRPCConn,
		ProvideHTTPClient,
		ProvideOutboxInterval,
		SharedSet,
		CacheSet,
		GitHubSet,
		TrackingRepositorySet,
		SubscriptionRepositorySet,
		BrokerSet,
		UseCaseSet,
		GrpcSet,
		trackingOutbox.NewRelay,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
