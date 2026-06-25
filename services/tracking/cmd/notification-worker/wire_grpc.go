package main

import (
	"github.com/google/wire"

	trackingGrpc "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/transport/grpc"
)

var GrpcSet = wire.NewSet(
	trackingGrpc.NewTrackingHandler,
	trackingGrpc.NewGrpcServer,
)
