package redis

import (
	"context"
	"fmt"

	redis "github.com/redis/go-redis/v9"

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
		return nil, err
	}

	return rdb, nil
}
