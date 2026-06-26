//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	"github.com/google/wire"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/infrastructure/adapter"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/infrastructure/outbox"
	subRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/transport/broker/rabbitmq"
	subHttp "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/transport/http"
	subUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/usecase"
)

func ProvideSubscriptionDBConfig(cfg Config) config.SubscriptionDBConfig { return cfg.SubscriptionDB }
func ProvideRedisConfig(cfg Config) config.RedisConfig                   { return cfg.Redis }
func ProvideHTTPConfig(cfg Config) config.HTTPConfig                     { return cfg.HTTP }
func ProvideAppConfig(cfg Config) config.AppConfig                       { return cfg.App }
func ProvideRabbitMQConfig(cfg Config) config.RabbitMQConfig             { return cfg.RabbitMQ }
func ProvideTrackingClientConfig(cfg Config) config.TrackingClientConfig { return cfg.TrackingClient }

func ProvideRabbitMQConnection(cfg config.RabbitMQConfig) (*sharedRabbitmq.Connection, func(), error) {
	return sharedRabbitmq.NewConnection(cfg.URL)
}

func ProvideOutboxInterval() time.Duration {
	return 5 * time.Second
}

func ProvideTrackingGRPCConn(cfg config.TrackingClientConfig) (*grpc.ClientConn, func(), error) {
	conn, err := grpc.NewClient(cfg.TrackingGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("dial tracking grpc: %w", err)
	}
	return conn, func() { conn.Close() }, nil
}

func ProvideTrackingServiceClient(conn *grpc.ClientConn) pb.TrackingServiceClient {
	return pb.NewTrackingServiceClient(conn)
}

type App struct {
	HTTPServer         *subHttp.GinServer
	GrpcServer         *grpc.Server
	SagaResultConsumer *subRabbitmq.SagaResultConsumer
	OutboxRelay        *outbox.Relay
}

func InitializeApp(ctx context.Context, cfg Config) (*App, func(), error) {
	wire.Build(
		ProvideSubscriptionDBConfig,
		ProvideRedisConfig,
		ProvideHTTPConfig,
		ProvideAppConfig,
		ProvideRabbitMQConfig,
		ProvideTrackingClientConfig,
		ProvideOutboxInterval,
		ProvideTrackingGRPCConn,
		ProvideTrackingServiceClient,
		SharedSet,
		CacheSet,
		SubscriptionRepositorySet,
		SubscriptionUseCaseSet,
		BrokerSet,
		GrpcSet,
		outbox.NewRelay,
		subHttp.NewGinServer,
		adapter.NewRepositoryUseCaseAdapter,
		wire.Bind(new(subUsecase.TrackingRepository), new(*adapter.RepositoryUseCaseAdapter)),
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
