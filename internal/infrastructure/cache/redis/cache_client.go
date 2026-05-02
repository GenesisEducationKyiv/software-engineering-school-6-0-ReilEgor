package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/ReilEgor/RepoNotifier/internal/config"
)

func NewRedisClient(
	host config.RedisHostType,
	port config.RedisPortType,
	password config.RedisPasswordType,
	db int,
) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: string(password),
		DB:       db,
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
