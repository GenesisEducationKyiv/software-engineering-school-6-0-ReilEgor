package main

import (
	"github.com/google/wire"

	subscriptionRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	subscriptionUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/usecase"
	subRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/infrastructure/broker/rabbitmq"
	subPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/repository/postgres"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/saga"
	subGrpc "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/transport/grpc"
	subUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/usecase"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/storage/postgres"
)

var SubscriptionRepositorySet = wire.NewSet(
	subPostgres.NewSubscriptionRepository,
	subPostgres.NewUserRepository,
	subPostgres.NewSagaRepository,
	subPostgres.NewRepositoryRepository,
	subPostgres.NewOutboxRepository,
	sharedPostgres.NewTransactor,
	wire.Bind(new(subscriptionRepo.SubscriptionRepository), new(*subPostgres.SubscriptionRepository)),
	wire.Bind(new(subscriptionRepo.SubscriptionWriter), new(*subPostgres.SubscriptionRepository)),
	wire.Bind(new(subscriptionRepo.UserRepository), new(*subPostgres.UserRepository)),
	wire.Bind(new(subscriptionRepo.SagaRepository), new(*subPostgres.SagaRepository)),
	wire.Bind(new(subscriptionRepo.RepositoryUpdater), new(*subPostgres.RepositoryRepository)),
	wire.Bind(new(subscriptionRepo.OutboxRepository), new(*subPostgres.OutboxRepository)),
	wire.Bind(new(subscriptionRepo.Transactor), new(*sharedPostgres.Transactor)),
)

var SubscriptionUseCaseSet = wire.NewSet(
	saga.NewOrchestrator,
	subUsecase.NewUserUseCase,
	subUsecase.NewRepositoryUseCase,
	wire.Bind(new(subscriptionUsecase.UserUseCase), new(*subUsecase.UserUseCase)),
	wire.Bind(new(subscriptionUsecase.RepositoryUseCase), new(*subUsecase.RepositoryUseCase)),
)

var BrokerSet = wire.NewSet(
	ProvideRabbitMQConnection,
	subRabbitmq.NewSagaResultConsumer,
)

var GrpcSet = wire.NewSet(
	subGrpc.NewSubscriptionHandler,
	subGrpc.NewGrpcServer,
)
