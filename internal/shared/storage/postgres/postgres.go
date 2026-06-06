package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
)

func New(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, func(), error) {
	slog.Info("connecting to database",
		slog.String("dsn", maskDSN(cfg.DSN)),
	)

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, nil, fmt.Errorf("parse config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxOpenConns
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
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
		slog.Int("max_open_conns", int(cfg.MaxOpenConns)))

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
