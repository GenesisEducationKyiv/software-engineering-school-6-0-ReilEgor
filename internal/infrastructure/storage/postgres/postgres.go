package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/config"
)

const (
	MaxOpenConnections = 25
	MaxIdleConnections = 25
)

func New(ctx context.Context, dsn config.DSNType) (*pgxpool.Pool, func(), error) {
	slog.With(slog.String("component", "postgres"))
	slog.Info("connecting to database",
		slog.String("dsn", maskDSN(string(dsn))),
	)
	myConfig, err := pgxpool.ParseConfig(string(dsn))
	if err != nil {
		return nil, nil, fmt.Errorf("parse config: %w", err)
	}

	myConfig.MaxConns = MaxOpenConnections
	myConfig.MaxConnIdleTime = time.Duration(MaxIdleConnections) * time.Second
	myConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, myConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("create pool: %w", err)
	}
	start := time.Now()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping failed",
			slog.Any("error", err),
			slog.Duration("duration", time.Since(start)))
		return nil, nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("successful connection to PostgreSQL",
		slog.Duration("latency", time.Since(start)),
		slog.Int("max_open_conns", MaxOpenConnections))

	cleanup := func() {
		slog.Info("closing database connections")
		pool.Close()
	}

	return pool, cleanup, nil
}

func maskDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return "invalid-dsn"
	}

	return u.Redacted()
}
