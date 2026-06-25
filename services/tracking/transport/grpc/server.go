package grpc

import (
	"google.golang.org/grpc"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/transport/grpc/middleware"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"
)

func NewGrpcServer(h *TrackingHandler, appCfg config.AppConfig) *grpc.Server {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthInterceptor(appCfg.APIKey)),
	)
	pb.RegisterTrackingServiceServer(srv, h)
	return srv
}
