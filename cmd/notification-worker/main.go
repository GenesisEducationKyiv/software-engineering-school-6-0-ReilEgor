package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
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

	if cfg.Worker.HealthPort != "" {
		g.Go(func() error {
			addr := fmt.Sprintf(":%s", cfg.Worker.HealthPort)
			myLogger.Info("health server starting", slog.String("addr", addr))
			return startHealthServer(ctx, addr, myLogger)
		})
	}

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

func startHealthServer(ctx context.Context, addr string, l *slog.Logger) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			l.Debug("health response write failed", slog.Any("error", err))
		}
	})

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		l.Info("health server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			l.Error("health server shutdown error", slog.Any("error", err))
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("health server error: %w", err)
	}
	return nil
}
