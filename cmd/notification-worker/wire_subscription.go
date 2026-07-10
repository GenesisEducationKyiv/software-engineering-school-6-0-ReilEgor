package main

import (
	"github.com/google/wire"

	subDomainRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	subPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/repository/postgres"
	trackingPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/port"
)

var SubscriptionRepositorySet = wire.NewSet(
	subPostgres.NewSubscriptionRepository,
	wire.Bind(new(subDomainRepo.SubscriptionRepository), new(*subPostgres.SubscriptionRepository)),
	wire.Bind(new(trackingPort.SubscriberReader), new(*subPostgres.SubscriptionRepository)),
)
