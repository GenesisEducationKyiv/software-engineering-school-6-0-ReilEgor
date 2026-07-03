package grpc

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	sharedMiddleware "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/middleware"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"
	"google.golang.org/grpc"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/transport/grpc/middleware"
)

func NewGrpcServer(h *TrackingHandler, appCfg config.AppConfig) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			sharedMiddleware.RequestIDInterceptor(),
			middleware.AuthInterceptor(appCfg.APIKey),
		),
	)
	pb.RegisterTrackingServiceServer(srv, h)
	return srv
}
