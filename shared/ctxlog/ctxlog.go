package ctxlog

import (
	"context"
	"log/slog"
)

type (
	logKey   struct{}
	reqIDKey struct{}
)

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, logKey{}, l)
}

func FromCtx(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(logKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, reqIDKey{}, id)
}

func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(reqIDKey{}).(string); ok {
		return id
	}
	return ""
}
