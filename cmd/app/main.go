package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/caarlos0/env/v11"

	_ "context"

	"github.com/ReilEgor/RepoNotifier/internal/config"
)

// Swagger Metadata for API Documentation
//
//	@title						RepoNotifier API
//	@version					1.0    	      1.0
//	@description				Service for tracking GitHub releases.
//	@securityDefinitions.apiKey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key
//
//	@host						localhost:8080
//	@BasePath					/api/v1.
func main() {
	myLogger := setupLogger()
	cfg, err := loadConfig(myLogger)
	if err != nil {
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, cleanup, err := InitializeApp(ctx, cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword, 0, cfg.DSN,
		cfg.EmailHost, cfg.EmailPort, cfg.EmailPassword, cfg.EmailFrom, cfg.EmailUser,
		cfg.ApiKey, cfg.GitHubToken, cfg.AppBaseURL)
	if err != nil {
		myLogger.Error("application initialization failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer cleanup()

	errCh := make(chan error, 2)

	go startHTTPServer(ctx, app, cfg, myLogger, errCh)
	go startGRPCServer(ctx, app, cfg, myLogger, errCh)
	go startNotificationWorker(ctx, app, myLogger)

	wait(ctx, myLogger, errCh)
}

func setupLogger() *slog.Logger {
	myLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(myLogger)
	return myLogger
}

func loadConfig(l *slog.Logger) (config.Config, error) {
	var cfg config.Config
	if err := env.Parse(&cfg); err != nil {
		wrapErr := fmt.Errorf("failed to parse environment variables: %w", err)
		l.Error("config load error", slog.Any("error", wrapErr))
		return cfg, wrapErr
	}
	return cfg, nil
}

func startHTTPServer(ctx context.Context, app *App, cfg config.Config, l *slog.Logger, errCh chan error) {
	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	l.Info("HTTP server starting", slog.String("addr", addr))
	if err := app.HTTPServer.Run(ctx, addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errCh <- fmt.Errorf("http server error: %w", err)
	}
}

func startGRPCServer(ctx context.Context, app *App, cfg config.Config, l *slog.Logger, errCh chan error) {
	addr := fmt.Sprintf(":%s", cfg.GRPCPort)
	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		errCh <- fmt.Errorf("gRPC listen error: %w", err)
		return
	}
	go func() {
		<-ctx.Done()
		l.Info("gRPC server shutting down")
		app.GrpcServer.GracefulStop()
	}()

	l.Info("gRPC server starting", slog.String("addr", addr))
	if err := app.GrpcServer.Serve(lis); err != nil {
		errCh <- fmt.Errorf("gRPC server error: %w", err)
	}
}

func startNotificationWorker(ctx context.Context, app *App, l *slog.Logger) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			l.Info("notification worker stopped")
			return
		case <-ticker.C:
			if err := app.SubscriptionUseCase.ProcessNotifications(ctx); err != nil {
				l.Error("worker check failed", slog.Any("error", err))
			}
		}
	}
}

func wait(ctx context.Context, l *slog.Logger, errCh chan error) {
	select {
	case <-ctx.Done():
		l.Info("shutting down gracefully")
		if err := <-errCh; err != nil {
			logger.Error("server shutdown error", slog.Any("error", err))
		}
	case err := <-errCh:
		l.Error("server stopped unexpectedly", slog.Any("error", err))
	}
}
