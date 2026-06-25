package main

import (
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	sharedConfig "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/cache"
	redis2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/cache/redis"
	postgres2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"
)

func ProvideSubscriptionPoolConfig(cfg sharedConfig.SubscriptionDBConfig) postgres2.PoolConfig {
	return postgres2.PoolConfig{
		DSN:               cfg.DSN,
		MaxOpenConns:      cfg.MaxOpenConns,
		MaxConnIdleTime:   cfg.MaxConnIdleTime,
		HealthCheckPeriod: cfg.HealthCheckPeriod,
	}
}

var SharedSet = wire.NewSet(
	ProvideSubscriptionPoolConfig,
	postgres2.New,
	wire.Bind(new(postgres2.PgxInterface), new(*pgxpool.Pool)),
)

var CacheSet = wire.NewSet(
	redis2.NewRedisClient,
	redis2.NewCache,
	wire.Bind(new(sharedcache.Cache), new(*redis2.Cache)),
)
