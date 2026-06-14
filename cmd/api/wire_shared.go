package main

import (
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/cache/redis"
	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/cache"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/storage/postgres"
)

var SharedSet = wire.NewSet(
	postgres.New,
	wire.Bind(new(postgres.PgxInterface), new(*pgxpool.Pool)),
)

var CacheSet = wire.NewSet(
	redis.NewRedisClient,
	redis.NewCache,
	wire.Bind(new(sharedcache.Cache), new(*redis.Cache)),
)
