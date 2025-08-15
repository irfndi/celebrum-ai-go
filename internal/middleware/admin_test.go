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
		// Set environment variable
		_ = os.Setenv("ADMIN_API_KEY", "test-admin-key")
		defer func() { _ = os.Unsetenv("ADMIN_API_KEY") }()

		am := NewAdminMiddleware()
		assert.NotNil(t, am)
		assert.Equal(t, "test-admin-key", am.apiKey)
	})

	t.Run("without environment variable", func(t *testing.T) {
		// Ensure environment variable is not set
		_ = os.Unsetenv("ADMIN_API_KEY")

		am := NewAdminMiddleware()
		assert.NotNil(t, am)
		assert.Equal(t, "admin-dev-key-change-in-production", am.apiKey)
	})
}

func TestAdminMiddleware_RequireAdminAuth(t *testing.T) {
	// Set up test environment
	_ = os.Setenv("ADMIN_API_KEY", "test-admin-key")
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
		req.Header.Set("Authorization", "Bearer test-admin-key")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "admin access granted")
	})

	t.Run("valid API key in X-API-Key header", func(t *testing.T) {
		router := createTestRouter()
		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("X-API-Key", "test-admin-key")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "admin access granted")
	})

	t.Run("valid API key in query parameter", func(t *testing.T) {
		router := createTestRouter()
		req := httptest.NewRequest("GET", "/admin/test?api_key=test-admin-key", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "admin access granted")
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

	t.Run("invalid API key in query parameter", func(t *testing.T) {
		router := createTestRouter()
		req := httptest.NewRequest("GET", "/admin/test?api_key=invalid-key", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Unauthorized")
	})
}

func TestAdminMiddleware_ValidateAdminKey(t *testing.T) {
	_ = os.Setenv("ADMIN_API_KEY", "test-admin-key")
	defer func() { _ = os.Unsetenv("ADMIN_API_KEY") }()

	am := NewAdminMiddleware()

	t.Run("valid key", func(t *testing.T) {
		assert.True(t, am.ValidateAdminKey("test-admin-key"))
	})

	t.Run("invalid key", func(t *testing.T) {
		assert.False(t, am.ValidateAdminKey("invalid-key"))
	})

	t.Run("empty key", func(t *testing.T) {
		assert.False(t, am.ValidateAdminKey(""))
	})
}
