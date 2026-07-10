package main

import (
	"github.com/google/wire"

	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/repository"
	subscriptionDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/repository/postgres"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/saga"
	subRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/transport/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/transport/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/usecase"
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
	usecase.NewUserUseCase,
	usecase.NewRepositoryUseCase,
	wire.Bind(new(subscriptionDomainUsecase.UserUseCase), new(*usecase.UserUseCase)),
	wire.Bind(new(subscriptionDomainUsecase.RepositoryUseCase), new(*usecase.RepositoryUseCase)),
	wire.Bind(new(subRabbitmq.SagaResultHandler), new(*saga.Orchestrator)),
)

var BrokerSet = wire.NewSet(
	ProvideRabbitMQConnection,
	subRabbitmq.NewSagaResultConsumer,
)

var GrpcSet = wire.NewSet(
	grpc.NewSubscriptionHandler,
	grpc.NewGrpcServer,
)
