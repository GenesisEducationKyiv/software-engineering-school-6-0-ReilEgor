package middleware

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/zsais/go-gin-prometheus"
)

func SetupMiddleware(router *gin.Engine, logger *slog.Logger, redisClient *redis.Client) {
	router.Use(customCORS())
	p := ginprometheus.NewPrometheus("gin")
	p.Use(router)

	router.Use(gin.Recovery())
	router.Use(slogMiddleware(logger))
	router.Use(Timeout(5 * time.Second))

	rateLimiter, err := RateLimit(redisClient)
	if err != nil {
		logger.Error("failed to create rate limiter", "error", err)
		os.Exit(1)
	}
	router.Use(rateLimiter)
}

func customCORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Accept", "Authorization", "Content-Type", "X-API-Key"},
		ExposeHeaders:   []string{"Content-Length"},
		MaxAge:          12 * time.Hour,
	})
}

func slogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		logger.Info("request handled",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.ClientIP()),
		)
	}
}
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
