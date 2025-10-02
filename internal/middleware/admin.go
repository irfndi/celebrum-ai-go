package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// Package middleware provides HTTP middleware components for authentication,
// authorization, telemetry, and other cross-cutting concerns.

// AdminMiddleware provides admin authentication middleware
type AdminMiddleware struct {
	apiKey string
}

// NewAdminMiddleware creates a new admin authentication middleware
func NewAdminMiddleware() *AdminMiddleware {
	// Get admin API key from environment variable
	apiKey := os.Getenv("ADMIN_API_KEY")

	// Validate API key configuration
	if apiKey == "" {
		log.Fatal("ADMIN_API_KEY environment variable must be set")
	}

	// Prevent use of default/example keys in any environment
	if apiKey == "admin-dev-key-change-in-production" || apiKey == "admin-secret-key-change-me" {
		log.Fatal("ADMIN_API_KEY cannot use default/example values. Please set a secure API key.")
	}

	// Ensure minimum security requirements
	if len(apiKey) < 32 {
		log.Fatal("ADMIN_API_KEY must be at least 32 characters long for security")
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

		// Query parameter authentication removed for security reasons
		// API keys should only be passed via headers

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
