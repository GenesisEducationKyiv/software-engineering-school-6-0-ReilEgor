package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(expectedKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientKey := c.GetHeader("X-API-Key")

		if clientKey == "" || clientKey != expectedKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: invalid or missing API key",
			})
			return
		}

		c.Next()
	}
}
