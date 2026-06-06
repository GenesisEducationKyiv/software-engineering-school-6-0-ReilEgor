package grpc

import (
	"google.golang.org/grpc"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/transport/grpc/middleware"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/transport/grpc/proto/v1"
)

func NewGrpcServer(h *SubscriptionHandler, appCfg config.AppConfig) *grpc.Server {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthInterceptor(appCfg.APIKey)),
	)
	pb.RegisterSubscriptionServiceServer(srv, h)

	return srv
}
