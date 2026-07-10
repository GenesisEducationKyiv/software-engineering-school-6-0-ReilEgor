package main

import (
	"github.com/google/wire"

	trackingPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/repository/postgres"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/transport/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/usecase"
)

var SubscriptionRepositorySet = wire.NewSet(
	trackingPostgres.NewTrackerSubscriptionRepository,
	wire.Bind(new(usecase.SubscriberReader), new(*trackingPostgres.SubscriptionRepository)),
	wire.Bind(new(rabbitmq.TrackerSubscriptionWriter), new(*trackingPostgres.SubscriptionRepository)),
	wire.Bind(new(rabbitmq.TrackerSubscriptionDeleter), new(*trackingPostgres.SubscriptionRepository)),
)
