package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/metrics"
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		statusCode := strconv.Itoa(c.Writer.Status())

		metrics.HTTPRequestsTotal.WithLabelValues(c.Request.Method, route, statusCode).Inc()
		metrics.HTTPRequestDurationSeconds.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}
