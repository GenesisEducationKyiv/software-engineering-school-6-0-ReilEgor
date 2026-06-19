package main

import (
	"github.com/google/wire"

	subscriptionPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/port"
	subscriptionRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	subscriptionUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/usecase"
	subRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/infrastructure/broker/rabbitmq"
	subPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/repository/postgres"
	subGrpc "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/transport/grpc"
	subUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/usecase"
)

var SubscriptionRepositorySet = wire.NewSet(
	subPostgres.NewSubscriptionRepository,
	subPostgres.NewUserRepository,
	subPostgres.NewSagaRepository,
	wire.Bind(new(subscriptionRepo.SubscriptionRepository), new(*subPostgres.SubscriptionRepository)),
	wire.Bind(new(subscriptionRepo.SubscriptionWriter), new(*subPostgres.SubscriptionRepository)),
	wire.Bind(new(subscriptionRepo.UserRepository), new(*subPostgres.UserRepository)),
	wire.Bind(new(subscriptionRepo.SagaRepository), new(*subPostgres.SagaRepository)),
)

var SubscriptionUseCaseSet = wire.NewSet(
	subUsecase.NewUserUseCase,
	wire.Bind(new(subscriptionUsecase.UserUseCase), new(*subUsecase.UserUseCase)),
)

var BrokerSet = wire.NewSet(
	ProvideRabbitMQConnection,
	subRabbitmq.NewPublisher,
	subRabbitmq.NewSagaResultConsumer,
	wire.Bind(new(subscriptionPort.ConfirmationSender), new(*subRabbitmq.Publisher)),
)

var GrpcSet = wire.NewSet(
	subGrpc.NewSubscriptionHandler,
	subGrpc.NewGrpcServer,
)
