package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware provides admin authentication middleware
type AdminMiddleware struct {
	apiKey string
}

// NewAdminMiddleware creates a new admin authentication middleware
func NewAdminMiddleware() *AdminMiddleware {
	// Get admin API key from environment variable
	apiKey := os.Getenv("ADMIN_API_KEY")
	if apiKey == "" {
		// Use a default key for development (should be changed in production)
		apiKey = "admin-dev-key-change-in-production"
	}

	return &AdminMiddleware{
		apiKey: apiKey,
	}
}

// RequireAdminAuth middleware validates admin API keys
func (am *AdminMiddleware) RequireAdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key in Authorization header (Bearer token)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) == 2 && tokenParts[0] == "Bearer" {
				if tokenParts[1] == am.apiKey {
					c.Next()
					return
				}
			}
		}

		// Check for API key in X-API-Key header
		apiKeyHeader := c.GetHeader("X-API-Key")
		if apiKeyHeader == am.apiKey {
			c.Next()
			return
		}

		// Check for API key in query parameter (less secure, for development only)
		apiKeyQuery := c.Query("api_key")
		if apiKeyQuery == am.apiKey {
			c.Next()
			return
		}

		// No valid API key found
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "Valid admin API key required for this endpoint",
		})
		c.Abort()
	}
}

// ValidateAdminKey validates an admin API key
func (am *AdminMiddleware) ValidateAdminKey(key string) bool {
	return key == am.apiKey
}
