package grpc

import (
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	middleware2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/middleware"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/transport/grpc/middleware"
)

func NewGrpcServer(h *SubscriptionHandler, appCfg config.AppConfig) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware2.RequestIDInterceptor(),
			middleware.AuthInterceptor(appCfg.APIKey),
		),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:              30 * time.Second,
			Timeout:           5 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	pb.RegisterSubscriptionServiceServer(srv, h)
	reflection.Register(srv)

	return srv
}
