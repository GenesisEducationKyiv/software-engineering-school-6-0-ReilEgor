package grpc

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	"google.golang.org/grpc"

	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/transport/grpc/middleware"
)

func NewGrpcServer(h *TrackingHandler, appCfg config.AppConfig) *grpc.Server {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthInterceptor(appCfg.APIKey)),
	)
	pb.RegisterTrackingServiceServer(srv, h)
	return srv
}
