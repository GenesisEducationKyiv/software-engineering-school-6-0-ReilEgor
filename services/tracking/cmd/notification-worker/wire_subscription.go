package main

import (
	"github.com/google/wire"

	trackingPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/repository/postgres"
	trackingRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/transport/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/usecase"
)

var SubscriptionRepositorySet = wire.NewSet(
	trackingPostgres.NewTrackerSubscriptionRepository,
	wire.Bind(new(usecase.SubscriberReader), new(*trackingPostgres.SubscriptionRepository)),
	wire.Bind(new(trackingRabbitmq.TrackerSubscriptionWriter), new(*trackingPostgres.SubscriptionRepository)),
	wire.Bind(new(trackingRabbitmq.TrackerSubscriptionDeleter), new(*trackingPostgres.SubscriptionRepository)),
)
