package main

import (
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/cache/redis"
	sharedConfig "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/cache"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/storage/postgres"
)

func ProvideSubscriptionPoolConfig(cfg sharedConfig.SubscriptionDBConfig) postgres.PoolConfig {
	return postgres.PoolConfig{
		DSN:               cfg.DSN,
		MaxOpenConns:      cfg.MaxOpenConns,
		MaxConnIdleTime:   cfg.MaxConnIdleTime,
		HealthCheckPeriod: cfg.HealthCheckPeriod,
	}
}

var SharedSet = wire.NewSet(
	ProvideSubscriptionPoolConfig,
	postgres.New,
	postgres.NewTransactor,
	wire.Bind(new(postgres.PgxInterface), new(*pgxpool.Pool)),
)

var CacheSet = wire.NewSet(
	redis.NewRedisClient,
	redis.NewCache,
	wire.Bind(new(sharedcache.Cache), new(*redis.Cache)),
)
