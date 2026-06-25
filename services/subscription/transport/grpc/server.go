package grpc

import (
	"google.golang.org/grpc"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/transport/grpc/middleware"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"
)

func NewGrpcServer(h *SubscriptionHandler, appCfg config.AppConfig) *grpc.Server {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthInterceptor(appCfg.APIKey)),
	)
	pb.RegisterSubscriptionServiceServer(srv, h)

	return srv
}
