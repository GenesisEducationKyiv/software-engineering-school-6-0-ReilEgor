package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v11"

	sharedConfig "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
)

type Config struct {
	Email        sharedConfig.EmailConfig
	RabbitMQ     sharedConfig.RabbitMQConfig
	SenderConfig sharedConfig.SenderConfig
}

func main() {
	myLogger := setupLogger()
	cfg, err := loadConfig(myLogger)
	if err != nil {
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	_, cleanup, err := InitializeApp(ctx, cfg)
	if err != nil {
		myLogger.Error("application initialization failed", slog.Any("error", err))
		os.Exit(1)
	}
	for true {
	}
	defer cleanup()
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
