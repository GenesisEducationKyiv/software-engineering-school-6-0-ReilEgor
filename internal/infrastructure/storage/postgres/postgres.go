package postgres

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/ReilEgor/RepoNotifier/internal/config"
)

const (
	MAX_OPEN_CONNS = 25
	MAX_IDLE_CONNS = 25
)

func New(ctx context.Context, dsn config.DSNType) (*pgxpool.Pool, func(), error) {
	slog.With(slog.String("component", "postgres"))
	slog.Info("connecting to database",
		slog.String("dsn", maskDSN(string(dsn))),
	)
	config, err := pgxpool.ParseConfig(string(dsn))
	if err != nil {
		return nil, nil, err
	}

	config.MaxConns = MAX_OPEN_CONNS
	config.MaxConnIdleTime = time.Duration(MAX_IDLE_CONNS) * time.Second
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, nil, err
	}

	start := time.Now()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping failed",
			slog.Any("error", err),
			slog.Duration("duration", time.Since(start)))
		return nil, nil, err
	}

	slog.Info("successful connection to PostgreSQL",
		slog.Duration("latency", time.Since(start)),
		slog.Int("max_open_conns", MAX_OPEN_CONNS))

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
