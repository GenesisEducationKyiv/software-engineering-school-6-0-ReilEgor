package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	limiter "github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	redisstore "github.com/ulule/limiter/v3/drivers/store/redis"
)

func RateLimit(client *redis.Client) (gin.HandlerFunc, error) {
	rate, err := limiter.NewRateFromFormatted("10-S")
	if err != nil {
		return nil, err
	}

	store, err := redisstore.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix: "rate_limit",
	})
	if err != nil {
		return nil, err
	}

	instance := limiter.New(store, rate)
	return mgin.NewMiddleware(instance), nil
}
