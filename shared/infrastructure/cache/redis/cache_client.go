package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
)

func NewRedisClient(cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		if closeErr := rdb.Close(); closeErr != nil {
			return nil, fmt.Errorf(
				"failed to connect to redis (%w); additionally, failed to close client: %v",
				err,
				closeErr,
			)
		}
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return rdb, nil
}
