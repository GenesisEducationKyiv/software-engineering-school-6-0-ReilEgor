package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"golang.org/x/sync/errgroup"

	sharedConfig "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
)

type Config struct {
	DB     sharedConfig.DBConfig
	Redis  sharedConfig.RedisConfig
	Email  sharedConfig.EmailConfig
	GitHub sharedConfig.GitHubConfig
	Worker sharedConfig.WorkerConfig
	App    sharedConfig.AppConfig
}

func main() {
	myLogger := setupLogger()
	cfg, err := loadConfig(myLogger)
	if err != nil {
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, cleanup, err := InitializeApp(ctx, cfg)
	if err != nil {
		myLogger.Error("application initialization failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer cleanup()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		myLogger.Info("notification worker starting")
		startNotificationWorker(ctx, app, cfg, myLogger)
		return nil
	})

	if err := g.Wait(); err != nil {
		myLogger.Error("worker stopped", slog.Any("error", err))
	}
}

func setupLogger() *slog.Logger {
	myLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(myLogger)
	return myLogger
}

func loadConfig(l *slog.Logger) (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		wrapErr := fmt.Errorf("failed to parse environment variables: %w", err)
		l.Error("config load error", slog.Any("error", wrapErr))
		return cfg, wrapErr
	}
	return cfg, nil
}

func startNotificationWorker(ctx context.Context, app *App, cfg Config, l *slog.Logger) {
	ticker := time.NewTicker(cfg.Worker.NotificationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			l.Info("notification worker stopped")
			return
		case <-ticker.C:
			if err := app.NotificationUseCase.ProcessNotifications(ctx); err != nil {
				l.Error("worker check failed", slog.Any("error", err))
			}
		}
	}
}
