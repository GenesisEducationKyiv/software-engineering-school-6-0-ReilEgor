package grpc

import (
	"github.com/ReilEgor/RepoNotifier/internal/config"
	"github.com/ReilEgor/RepoNotifier/internal/transport/grpc/middleware"
	pb "github.com/ReilEgor/RepoNotifier/internal/transport/grpc/proto/v1"
	"google.golang.org/grpc"
)

func NewGrpcServer(h *SubscriptionHandler, apiKey config.ApiKeyType) *grpc.Server {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthInterceptor(string(apiKey))),
	)
	pb.RegisterSubscriptionServiceServer(srv, h)

	return srv
}
