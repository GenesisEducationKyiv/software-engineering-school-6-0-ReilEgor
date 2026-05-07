package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
)

type Cache struct {
	client *redis.Client
	logger *slog.Logger
}

const (
	componentCache = "Cache"

	errMsgCacheGet = "cache get error"
	errMsgCacheSet = "cache set error"
)

func NewCache(client *redis.Client) *Cache {
	return &Cache{
		client: client,
		logger: slog.With(slog.String("component", componentCache)),
	}
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	const op = "Cache.Get"
	log := c.logger.With(slog.String("op", op))
	value, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", service.ErrCacheMiss
		}
		log.ErrorContext(ctx, errMsgCacheGet,
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
		return "", fmt.Errorf("redis get: %w", err)
	}
	return value, nil
}

func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	const op = "Cache.Set"
	log := c.logger.With(slog.String("op", op))
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		log.ErrorContext(ctx, errMsgCacheSet,
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
