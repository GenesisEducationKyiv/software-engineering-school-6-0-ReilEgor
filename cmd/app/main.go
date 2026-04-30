package main

import (
	"context"
	_ "context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ReilEgor/RepoNotifier/internal/config"
	"github.com/caarlos0/env/v11"
)

// Swagger Metadata for API Documentation
// @title RepoNotifier API
// @version 1.0    	      1.0
// @description Service for tracking GitHub releases.

// @securityDefinitions.apiKey ApiKeyAuth
// @in header
// @name X-API-Key

// @host      localhost:8080
// @BasePath  /api/v1
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	var cfg config.Config
	err := env.Parse(&cfg)
	if err != nil {
		logger.Error("failed to load config",
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app, cleanup, err := InitializeApp(
		ctx, cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword, 0, cfg.DSN,
		cfg.EmailHost, cfg.EmailPort, cfg.EmailPassword, cfg.EmailFrom, cfg.EmailUser,
		cfg.ApiKey, cfg.GitHubToken, cfg.AppBaseURL,
	)
	if err != nil {
		logger.Error("application initialization failed",
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	defer cleanup()

	errCh := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%s", cfg.HTTPPort)
		logger.Info("HTTP server starting", slog.String("addr", addr))
		if err := app.HTTPServer.Run(ctx, addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server error: %w", err)
		}
	}()
	go func() {
		addr := fmt.Sprintf(":%s", cfg.GRPCPort)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			errCh <- fmt.Errorf("gRPC listen error: %w", err)
			return
		}
		logger.Info("gRPC server starting", slog.String("addr", addr))
		if err := app.GrpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		logger.Info("initial notification check started")
		if err := app.SubscriptionUseCase.ProcessNotifications(ctx); err != nil {
			logger.Error("initial notification check failed", slog.Any("error", err))
		}

		for {
			select {
			case <-ctx.Done():
				logger.Info("notification worker stopped")
				return
			case <-ticker.C:
				logger.Info("running scheduled notification check")
				if err := app.SubscriptionUseCase.ProcessNotifications(ctx); err != nil {
					logger.Error("worker check failed", slog.Any("error", err))
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down gracefully")
		if err := <-errCh; err != nil {
			logger.Error("server shutdown error", slog.Any("error", err))
		}
		logger.Info("server stopped")
	case err := <-errCh:
		logger.Error("server stopped unexpectedly", slog.Any("error", err))
	}
}
