package middleware

import (
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestID(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Header("X-Request-ID", reqID)

		reqLogger := logger.With(slog.String("request_id", reqID))
		ctx := ctxlog.WithLogger(c.Request.Context(), reqLogger)
		ctx = ctxlog.WithRequestID(ctx, reqID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
