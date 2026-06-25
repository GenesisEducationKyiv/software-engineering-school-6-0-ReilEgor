package main

import (
	"github.com/google/wire"

	trackingPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/port"
	trackingPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/repository/postgres"
)

var SubscriptionRepositorySet = wire.NewSet(
	trackingPostgres.NewTrackerSubscriptionRepository,
	wire.Bind(new(trackingPort.SubscriberReader), new(*trackingPostgres.SubscriptionRepository)),
	wire.Bind(new(trackingPort.TrackerSubscriptionWriter), new(*trackingPostgres.SubscriptionRepository)),
	wire.Bind(new(trackingPort.TrackerSubscriptionDeleter), new(*trackingPostgres.SubscriptionRepository)),
)
