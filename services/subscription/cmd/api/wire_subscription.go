package main

import (
	"github.com/google/wire"

	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/domain/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/repository/postgres"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/saga"
	subRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/transport/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/transport/grpc"
	usecase2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/usecase"
)

var SubscriptionRepositorySet = wire.NewSet(
	postgres.NewSubscriptionRepository,
	postgres.NewUserRepository,
	postgres.NewSagaRepository,
	postgres.NewRepositoryRepository,
	postgres.NewOutboxRepository,
	sharedPostgres.NewTransactor,
	wire.Bind(new(repository.SubscriptionRepository), new(*postgres.SubscriptionRepository)),
	wire.Bind(new(repository.SubscriptionWriter), new(*postgres.SubscriptionRepository)),
	wire.Bind(new(repository.UserRepository), new(*postgres.UserRepository)),
	wire.Bind(new(repository.SagaRepository), new(*postgres.SagaRepository)),
	wire.Bind(new(repository.RepositoryUpdater), new(*postgres.RepositoryRepository)),
	wire.Bind(new(repository.RepositoryRepository), new(*postgres.RepositoryRepository)),
	wire.Bind(new(repository.OutboxRepository), new(*postgres.OutboxRepository)),
	wire.Bind(new(repository.Transactor), new(*sharedPostgres.Transactor)),
)

var SubscriptionUseCaseSet = wire.NewSet(
	saga.NewOrchestrator,
	usecase2.NewUserUseCase,
	usecase2.NewRepositoryUseCase,
	wire.Bind(new(usecase.UserUseCase), new(*usecase2.UserUseCase)),
	wire.Bind(new(usecase.RepositoryUseCase), new(*usecase2.RepositoryUseCase)),
)

var BrokerSet = wire.NewSet(
	ProvideRabbitMQConnection,
	subRabbitmq.NewSagaResultConsumer,
)

var GrpcSet = wire.NewSet(
	grpc.NewSubscriptionHandler,
	grpc.NewGrpcServer,
)
