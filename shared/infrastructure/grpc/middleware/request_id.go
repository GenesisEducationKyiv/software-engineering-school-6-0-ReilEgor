package middleware

import (
	"context"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func RequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		reqID := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("x-request-id"); len(vals) > 0 {
				reqID = vals[0]
			}
		}
		if reqID == "" {
			reqID = uuid.NewString()
		}

		ctx = ctxlog.WithLogger(ctx, slog.Default().With(slog.String("request_id", reqID)))
		ctx = ctxlog.WithRequestID(ctx, reqID)
		return handler(ctx, req)
	}
}
