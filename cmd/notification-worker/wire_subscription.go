package main

import (
	"github.com/google/wire"

	trackingPort "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/port"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/infrastructure/noop"
)

var SubscriptionRepositorySet = wire.NewSet(
	noop.NewSubscriberReader,
	wire.Bind(new(trackingPort.SubscriberReader), new(*noop.SubscriberReader)),
)
