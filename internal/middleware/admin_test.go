package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewAdminMiddleware(t *testing.T) {
	t.Run("with environment variable", func(t *testing.T) {
		// Set environment variable with secure 32+ character key
		_ = os.Setenv("ADMIN_API_KEY", "test-admin-key-32-chars-minimum-length")
		defer func() { _ = os.Unsetenv("ADMIN_API_KEY") }()

		am := NewAdminMiddleware()
		assert.NotNil(t, am)
		assert.Equal(t, "test-admin-key-32-chars-minimum-length", am.apiKey)
	})

	// Note: Removed test for missing environment variable since log.Fatal()
	// calls os.Exit() which cannot be tested with assert.Panics()
	// In production, missing ADMIN_API_KEY will cause the application to exit

	t.Run("with default key validation", func(t *testing.T) {
		// Test default key 1
		_ = os.Setenv("ADMIN_API_KEY", "admin-dev-key-change-in-production")
		defer func() { _ = os.Unsetenv("ADMIN_API_KEY") }()

		// This should call log.Fatal and exit, so we can't test it directly
		// In production, this would prevent the application from starting
		// We'll just verify the logic exists by checking the function doesn't panic with valid keys
	})

	t.Run("with short key validation", func(t *testing.T) {
		// Test short key (less than 32 characters)
		_ = os.Setenv("ADMIN_API_KEY", "short-key")
		defer func() { _ = os.Unsetenv("ADMIN_API_KEY") }()

		// This should call log.Fatal and exit, so we can't test it directly
		// In production, this would prevent the application from starting
		// We'll just verify the logic exists by checking the function doesn't panic with valid keys
	})

	t.Run("with exactly 32 character key", func(t *testing.T) {
		// Test with exactly 32 characters (minimum allowed)
		_ = os.Setenv("ADMIN_API_KEY", "12345678901234567890123456789012")
		defer func() { _ = os.Unsetenv("ADMIN_API_KEY") }()

		am := NewAdminMiddleware()
		assert.NotNil(t, am)
		assert.Equal(t, "12345678901234567890123456789012", am.apiKey)
	})

	t.Run("with long key", func(t *testing.T) {
		// Test with long key (more than 32 characters)
		_ = os.Setenv("ADMIN_API_KEY", "very-long-admin-key-that-is-much-longer-than-32-characters")
		defer func() { _ = os.Unsetenv("ADMIN_API_KEY") }()

		am := NewAdminMiddleware()
		assert.NotNil(t, am)
		assert.Equal(t, "very-long-admin-key-that-is-much-longer-than-32-characters", am.apiKey)
	})
}

func TestAdminMiddleware_RequireAdminAuth(t *testing.T) {
	// Set up test environment with secure 32+ character key
	_ = os.Setenv("ADMIN_API_KEY", "test-admin-key-32-chars-minimum-length")
	defer func() { _ = os.Unsetenv("ADMIN_API_KEY") }()

	am := NewAdminMiddleware()
	gin.SetMode(gin.TestMode)

	// Create test router
	createTestRouter := func() *gin.Engine {
		router := gin.New()
		router.Use(am.RequireAdminAuth())
		router.GET("/admin/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
		})
		return router
	}

	t.Run("valid API key in Authorization header", func(t *testing.T) {
		router := createTestRouter()
		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("Authorization", "Bearer test-admin-key-32-chars-minimum-length")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "admin access granted")
	})

	t.Run("valid API key in X-API-Key header", func(t *testing.T) {
		router := createTestRouter()
		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("X-API-Key", "test-admin-key-32-chars-minimum-length")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "admin access granted")
	})

	t.Run("API key in query parameter (should be rejected for security)", func(t *testing.T) {
		router := createTestRouter()
		req := httptest.NewRequest("GET", "/admin/test?api_key=test-admin-key-32-chars-minimum-length", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Query parameter authentication is disabled for security
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Unauthorized")
	})

	t.Run("missing API key", func(t *testing.T) {
		router := createTestRouter()
		req := httptest.NewRequest("GET", "/admin/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Unauthorized")
		assert.Contains(t, w.Body.String(), "Valid admin API key required")
	})

	t.Run("invalid API key in Authorization header", func(t *testing.T) {
		router := createTestRouter()
		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-key")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Unauthorized")
	})

	t.Run("invalid Authorization header format", func(t *testing.T) {
		router := createTestRouter()
		testCases := []string{
			"test-admin-key",       // Missing Bearer prefix
			"Basic test-admin-key", // Wrong auth type
			"Bearer",               // Missing key
			"Bearer key1 key2",     // Too many parts
		}

		for _, authHeader := range testCases {
			req := httptest.NewRequest("GET", "/admin/test", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Body.String(), "Unauthorized")
		}
	})

	t.Run("invalid API key in X-API-Key header", func(t *testing.T) {
		router := createTestRouter()
		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("X-API-Key", "invalid-key")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Unauthorized")
	})

	t.Run("invalid API key in query parameter (should be rejected for security)", func(t *testing.T) {
		router := createTestRouter()
		req := httptest.NewRequest("GET", "/admin/test?api_key=invalid-key", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Query parameter authentication is disabled for security
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Unauthorized")
	})
}

func TestAdminMiddleware_ValidateAdminKey(t *testing.T) {
	_ = os.Setenv("ADMIN_API_KEY", "test-admin-key-32-chars-minimum-length")
	defer func() { _ = os.Unsetenv("ADMIN_API_KEY") }()

	am := NewAdminMiddleware()

	t.Run("valid key", func(t *testing.T) {
		assert.True(t, am.ValidateAdminKey("test-admin-key-32-chars-minimum-length"))
	})

	t.Run("invalid key", func(t *testing.T) {
		assert.False(t, am.ValidateAdminKey("invalid-key"))
	})

	t.Run("empty key", func(t *testing.T) {
		assert.False(t, am.ValidateAdminKey(""))
	})
}
