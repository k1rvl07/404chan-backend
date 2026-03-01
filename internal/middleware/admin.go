package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AdminAPIKeyMiddleware(adminAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if adminAPIKey == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "cleanup not configured"})
			c.Abort()
			return
		}

		apiKey := c.GetHeader("X-Admin-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey != adminAPIKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			c.Abort()
			return
		}

		c.Next()
	}
}
