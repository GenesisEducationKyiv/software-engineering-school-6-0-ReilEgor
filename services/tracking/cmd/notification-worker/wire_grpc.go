package main

import (
	"github.com/google/wire"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/transport/grpc"
)

var GrpcSet = wire.NewSet(
	grpc.NewTrackingHandler,
	grpc.NewGrpcServer,
)
