package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/irfandi/celebrum-ai-go/internal/telemetry"
)

func TestTelemetryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Initialize telemetry for testing
	config := telemetry.DefaultConfig()
	config.Enabled = false // Disable for testing to avoid network calls
	err := telemetry.InitTelemetry(*config)
	require.NoError(t, err)

	t.Run("regular request tracing", func(t *testing.T) {
		router := gin.New()
		router.Use(TelemetryMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "test-agent")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "test")
	})

	t.Run("health check endpoint skip", func(t *testing.T) {
		router := gin.New()
		router.Use(TelemetryMiddleware())
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		})

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "healthy")
	})

	t.Run("ready endpoint skip", func(t *testing.T) {
		router := gin.New()
		router.Use(TelemetryMiddleware())
		router.GET("/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		})

		req := httptest.NewRequest("GET", "/ready", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ready")
	})

	t.Run("live endpoint skip", func(t *testing.T) {
		router := gin.New()
		router.Use(TelemetryMiddleware())
		router.GET("/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "live"})
		})

		req := httptest.NewRequest("GET", "/live", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "live")
	})

	t.Run("error response tracing", func(t *testing.T) {
		router := gin.New()
		router.Use(TelemetryMiddleware())
		router.GET("/error", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		})

		req := httptest.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "internal error")
	})
}

func TestHealthCheckTelemetryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Initialize telemetry for testing
	config := telemetry.DefaultConfig()
	config.Enabled = false // Disable for testing to avoid network calls
	err := telemetry.InitTelemetry(*config)
	require.NoError(t, err)

	t.Run("health check middleware", func(t *testing.T) {
		router := gin.New()
		router.Use(HealthCheckTelemetryMiddleware())
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		})

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "healthy")
	})

	t.Run("health check with error response", func(t *testing.T) {
		router := gin.New()
		router.Use(HealthCheckTelemetryMiddleware())
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
		})

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "unhealthy")
	})
}

func TestRecordError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Initialize telemetry for testing
	config := telemetry.DefaultConfig()
	config.Enabled = false // Disable for testing to avoid network calls
	err := telemetry.InitTelemetry(*config)
	require.NoError(t, err)

	t.Run("record error with valid context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		// Create a proper request with context
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		// Create a real span for testing
		tracer := telemetry.GetHTTPTracer()
		ctx, span := tracer.Start(c.Request.Context(), "test_span")
		c.Request = c.Request.WithContext(ctx)

		testErr := assert.AnError
		description := "test error description"
		
		// This should not panic
		RecordError(c, testErr, description)
		span.End()
	})

	t.Run("record error with nil context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		// Create a proper request with context
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		testErr := assert.AnError
		description := "test error description"
		
		// This should not panic even with nil context
		RecordError(c, testErr, description)
	})

	t.Run("record error with non-recording span", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		// Create a proper request with context
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		// Create a non-recording span context
		nonRecordingCtx := context.WithValue(c.Request.Context(), "non-recording", true)
		c.Request = c.Request.WithContext(nonRecordingCtx)
		
		testErr := assert.AnError
		description := "test error description"
		
		// This should not panic even with non-recording span
		RecordError(c, testErr, description)
	})

	t.Run("record error with non-recording span", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		// Create a proper request with context
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		// Create a non-recording span context
		nonRecordingCtx := context.WithValue(c.Request.Context(), "non-recording", true)
		c.Request = c.Request.WithContext(nonRecordingCtx)
		
		testErr := assert.AnError
		description := "test error description"
		
		// This should not panic even with non-recording span
		RecordError(c, testErr, description)
	})
}

func TestAddSpanAttribute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Initialize telemetry for testing
	config := telemetry.DefaultConfig()
	config.Enabled = false // Disable for testing to avoid network calls
	err := telemetry.InitTelemetry(*config)
	require.NoError(t, err)

	t.Run("add string attribute", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		// Create a proper request with context
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		// Create a real span for testing
		tracer := telemetry.GetHTTPTracer()
		ctx, span := tracer.Start(c.Request.Context(), "test_span")
		c.Request = c.Request.WithContext(ctx)

		// This should not panic
		AddSpanAttribute(c, "test_key", "test_value")
		span.End()
	})

	t.Run("add int attribute", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		tracer := telemetry.GetHTTPTracer()
		ctx, span := tracer.Start(c.Request.Context(), "test_span")
		c.Request = c.Request.WithContext(ctx)

		AddSpanAttribute(c, "test_key", 42)
		span.End()
	})

	t.Run("add int64 attribute", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		tracer := telemetry.GetHTTPTracer()
		ctx, span := tracer.Start(c.Request.Context(), "test_span")
		c.Request = c.Request.WithContext(ctx)

		AddSpanAttribute(c, "test_key", int64(42))
		span.End()
	})

	t.Run("add float64 attribute", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		tracer := telemetry.GetHTTPTracer()
		ctx, span := tracer.Start(c.Request.Context(), "test_span")
		c.Request = c.Request.WithContext(ctx)

		AddSpanAttribute(c, "test_key", 3.14)
		span.End()
	})

	t.Run("add bool attribute", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		tracer := telemetry.GetHTTPTracer()
		ctx, span := tracer.Start(c.Request.Context(), "test_span")
		c.Request = c.Request.WithContext(ctx)

		AddSpanAttribute(c, "test_key", true)
		span.End()
	})

	t.Run("add unknown type attribute", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		tracer := telemetry.GetHTTPTracer()
		ctx, span := tracer.Start(c.Request.Context(), "test_span")
		c.Request = c.Request.WithContext(ctx)

		AddSpanAttribute(c, "test_key", []string{"item1", "item2"})
		span.End()
	})

	t.Run("add attribute with nil context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		// Create a proper request with context
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		// This should not panic even with nil context
		AddSpanAttribute(c, "test_key", "test_value")
	})

	t.Run("add attribute with non-recording span", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		// Create a proper request with context
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		// Create a non-recording span context
		nonRecordingCtx := context.WithValue(c.Request.Context(), "non-recording", true)
		c.Request = c.Request.WithContext(nonRecordingCtx)
		
		// This should not panic even with non-recording span
		AddSpanAttribute(c, "test_key", "test_value")
	})
}

func TestStartSpan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Initialize telemetry for testing
	config := telemetry.DefaultConfig()
	config.Enabled = false // Disable for testing to avoid network calls
	err := telemetry.InitTelemetry(*config)
	require.NoError(t, err)

	t.Run("start new span", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		// Create a proper request with context
		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		
		// This should not panic and should return valid context and span
		ctx, span := StartSpan(c, "test_span")
		
		assert.NotNil(t, ctx)
		assert.NotNil(t, span)
		assert.Equal(t, ctx, c.Request.Context())
		span.End()
	})
}

func TestGetHealthStatusFromCode(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected string
	}{
		{"healthy - 200", 200, "healthy"},
		{"healthy - 299", 299, "healthy"},
		{"client error - 400", 400, "client_error"},
		{"client error - 499", 499, "client_error"},
		{"server error - 500", 500, "server_error"},
		{"server error - 599", 599, "server_error"},
		{"server error - 600", 600, "server_error"},
		{"unknown - 100", 100, "unknown"},
		{"unknown - 300", 300, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getHealthStatusFromCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}