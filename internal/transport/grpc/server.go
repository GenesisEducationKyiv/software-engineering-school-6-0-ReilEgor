package grpc

import (
	"google.golang.org/grpc"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/grpc/middleware"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/grpc/proto/v1"
)

func NewGrpcServer(h *SubscriptionHandler, apiKey config.APIKeyType) *grpc.Server {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthInterceptor(string(apiKey))),
	)
	pb.RegisterSubscriptionServiceServer(srv, h)

	return srv
}
