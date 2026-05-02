package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"

	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	redisstore "github.com/ulule/limiter/v3/drivers/store/redis"
)

func RateLimit(client *redis.Client) (gin.HandlerFunc, error) {
	rate, err := limiter.NewRateFromFormatted("10-S")
	if err != nil {
		return nil, fmt.Errorf("parse rate: %w", err)
	}

	store, err := redisstore.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix: "rate_limit",
	})
	if err != nil {
		return nil, fmt.Errorf("create redis store: %w", err)
	}

	instance := limiter.New(store, rate)
	return mgin.NewMiddleware(instance), nil
}
