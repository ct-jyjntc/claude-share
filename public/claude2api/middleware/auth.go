package middleware

import (
	"claude2api/config"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware initializes the Claude client from the request header
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.ConfigInstance.EnableMirrorApi && strings.HasPrefix(c.Request.URL.Path, config.ConfigInstance.MirrorApiPrefix) {
			c.Set("UseMirrorApi", true)
			c.Next()
			return
		}
		Key := c.GetHeader("Authorization")
		if Key != "" {
			Key = strings.TrimPrefix(Key, "Bearer ")
			if Key != config.ConfigInstance.APIKey {
				c.JSON(401, gin.H{
					"error": "Invalid API key",
				})
				c.Abort()
				return
			}
			c.Next()
			return
		}
		c.JSON(401, gin.H{
			"error": "Missing or invalid Authorization header",
		})
		c.Abort()
	}
}
