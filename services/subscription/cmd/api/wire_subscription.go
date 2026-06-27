package main

import (
	"github.com/google/wire"

	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"

	repository2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/repository"
	usecase4 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/usecase"
	postgres2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/repository/postgres"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/saga"
	subRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/transport/broker/rabbitmq"
	grpc2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/transport/grpc"
	usecase3 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/usecase"
)

var SubscriptionRepositorySet = wire.NewSet(
	postgres2.NewSubscriptionRepository,
	postgres2.NewUserRepository,
	postgres2.NewSagaRepository,
	postgres2.NewRepositoryRepository,
	postgres2.NewOutboxRepository,
	sharedPostgres.NewTransactor,
	wire.Bind(new(repository2.SubscriptionRepository), new(*postgres2.SubscriptionRepository)),
	wire.Bind(new(repository2.SubscriptionWriter), new(*postgres2.SubscriptionRepository)),
	wire.Bind(new(repository2.UserRepository), new(*postgres2.UserRepository)),
	wire.Bind(new(repository2.SagaRepository), new(*postgres2.SagaRepository)),
	wire.Bind(new(repository2.RepositoryUpdater), new(*postgres2.RepositoryRepository)),
	wire.Bind(new(repository2.RepositoryRepository), new(*postgres2.RepositoryRepository)),
	wire.Bind(new(repository2.OutboxRepository), new(*postgres2.OutboxRepository)),
	wire.Bind(new(repository2.Transactor), new(*sharedPostgres.Transactor)),
)

var SubscriptionUseCaseSet = wire.NewSet(
	saga.NewOrchestrator,
	usecase3.NewUserUseCase,
	usecase3.NewRepositoryUseCase,
	wire.Bind(new(usecase4.UserUseCase), new(*usecase3.UserUseCase)),
	wire.Bind(new(usecase4.RepositoryUseCase), new(*usecase3.RepositoryUseCase)),
)

var BrokerSet = wire.NewSet(
	ProvideRabbitMQConnection,
	subRabbitmq.NewSagaResultConsumer,
)

var GrpcSet = wire.NewSet(
	grpc2.NewSubscriptionHandler,
	grpc2.NewGrpcServer,
)
