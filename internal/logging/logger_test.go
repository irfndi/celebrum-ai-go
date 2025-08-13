package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestLogger creates a logger with colors disabled for testing
func setupTestLogger(level, env string) (*StandardLogger, *bytes.Buffer) {
	logger := NewStandardLogger(level, env)
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	// Disable colors for testing
	logger.SetFormatter(&logrus.TextFormatter{ForceColors: false})
	return logger, &buf
}

func TestNewStandardLogger_Development(t *testing.T) {
	logger := NewStandardLogger("info", "development")

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
	assert.Equal(t, logrus.InfoLevel, logger.Level)

	// Check that text formatter is used in development
	_, ok := logger.Formatter.(*logrus.TextFormatter)
	assert.True(t, ok, "Expected TextFormatter for development environment")
}

func TestNewStandardLogger_Production(t *testing.T) {
	logger := NewStandardLogger("info", "production")

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
	assert.Equal(t, logrus.InfoLevel, logger.Level)

	// Check that JSON formatter is used in production
	_, ok := logger.Formatter.(*logrus.JSONFormatter)
	assert.True(t, ok, "Expected JSONFormatter for production environment")
}

func TestNewStandardLogger_UnknownEnvironment(t *testing.T) {
	logger := NewStandardLogger("info", "unknown")

	assert.NotNil(t, logger)
	// Should default to text formatter for unknown environments
	_, ok := logger.Formatter.(*logrus.TextFormatter)
	assert.True(t, ok, "Expected TextFormatter for unknown environment")
}

func TestNewStandardLogger_LogLevels(t *testing.T) {
	tests := []struct {
		levelStr string
		expected logrus.Level
	}{
		{"debug", logrus.DebugLevel},
		{"info", logrus.InfoLevel},
		{"warn", logrus.WarnLevel},
		{"error", logrus.ErrorLevel},
		{"invalid", logrus.InfoLevel}, // Should default to info
	}

	for _, tt := range tests {
		t.Run(tt.levelStr, func(t *testing.T) {
			logger := NewStandardLogger(tt.levelStr, "development")
			assert.Equal(t, tt.expected, logger.Level)
		})
	}
}

func TestStandardLogger_WithService(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	logger.WithService("new-service").Info("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "service=new-service")
	assert.Contains(t, logOutput, "test message")
}

func TestStandardLogger_WithComponent(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	logger.WithComponent("database").Info("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "component=database")
	assert.Contains(t, logOutput, "test message")
}

func TestStandardLogger_WithOperation(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	logger.WithOperation("fetch_symbols").Info("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "operation=fetch_symbols")
	assert.Contains(t, logOutput, "test message")
}

func TestStandardLogger_WithRequestID(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	requestID := "req-123456"
	logger.WithRequestID(requestID).Info("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "request_id=req-123456")
	assert.Contains(t, logOutput, "test message")
}

func TestStandardLogger_WithUserID(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	userID := "user-789"
	logger.WithUserID(userID).Info("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "user_id=user-789")
	assert.Contains(t, logOutput, "test message")
}

func TestStandardLogger_WithExchange(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	logger.WithExchange("binance").Info("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "exchange=binance")
	assert.Contains(t, logOutput, "test message")
}

func TestStandardLogger_WithSymbol(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	logger.WithSymbol("BTC/USD").Info("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "symbol=BTC/USD")
	assert.Contains(t, logOutput, "test message")
}

func TestStandardLogger_WithError(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	testErr := assert.AnError
	logger.WithError(testErr).Error("test error message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "error=")
	assert.Contains(t, logOutput, "test error message")
}

func TestStandardLogger_WithMetrics(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	metrics := map[string]interface{}{
		"duration_ms": 150,
		"status_code": 200,
		"bytes_sent":  1024,
	}

	logger.WithMetrics(metrics).Info("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "duration_ms=150")
	assert.Contains(t, logOutput, "status_code=200")
	assert.Contains(t, logOutput, "bytes_sent=1024")
	assert.Contains(t, logOutput, "test message")
}

func TestStandardLogger_LogStartup(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	logger.LogStartup("test-service", "1.0.0", 8080)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "service=test-service")
	assert.Contains(t, logOutput, "version=1.0.0")
	assert.Contains(t, logOutput, "port=8080")
	assert.Contains(t, logOutput, "event=startup")
	assert.Contains(t, logOutput, "Service starting")
}

func TestStandardLogger_LogShutdown(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	logger.LogShutdown("test-service", "graceful")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "service=test-service")
	assert.Contains(t, logOutput, "reason=graceful")
	assert.Contains(t, logOutput, "event=shutdown")
	assert.Contains(t, logOutput, "Service shutting down")
}

func TestStandardLogger_LogPerformanceMetrics(t *testing.T) {
	logger, buf := setupTestLogger("debug", "development")

	metrics := map[string]interface{}{
		"cpu_usage":    75.5,
		"memory_usage": 1024,
	}

	logger.LogPerformanceMetrics("test-service", metrics)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "service=test-service")
	assert.Contains(t, logOutput, "event=performance_metrics")
	assert.Contains(t, logOutput, "Performance metrics collected")
}

func TestStandardLogger_LogResourceStats(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	stats := map[string]interface{}{
		"goroutines": 100,
		"heap_size":  2048,
	}

	logger.LogResourceStats("test-service", stats)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "service=test-service")
	assert.Contains(t, logOutput, "event=resource_stats")
	assert.Contains(t, logOutput, "Resource statistics")
}

func TestStandardLogger_LogCacheOperation(t *testing.T) {
	logger, buf := setupTestLogger("debug", "development")

	logger.LogCacheOperation("get", "symbols:binance", true, 15)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "event=cache_operation")
	assert.Contains(t, logOutput, "operation=get")
	assert.Contains(t, logOutput, "key=\"symbols:binance\"")
	assert.Contains(t, logOutput, "hit=true")
	assert.Contains(t, logOutput, "duration_ms=15")
	assert.Contains(t, logOutput, "Cache operation")
}

func TestStandardLogger_LogDatabaseOperation(t *testing.T) {
	logger, buf := setupTestLogger("debug", "development")

	logger.LogDatabaseOperation("insert", "users", 250, 1)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "event=database_operation")
	assert.Contains(t, logOutput, "operation=insert")
	assert.Contains(t, logOutput, "table=users")
	assert.Contains(t, logOutput, "duration_ms=250")
	assert.Contains(t, logOutput, "rows_affected=1")
	assert.Contains(t, logOutput, "Database operation")
}

func TestStandardLogger_LogAPIRequest(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	logger.LogAPIRequest("GET", "/api/symbols", 200, 150, "user123")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "event=api_request")
	assert.Contains(t, logOutput, "method=GET")
	assert.Contains(t, logOutput, "path=/api/symbols")
	assert.Contains(t, logOutput, "status_code=200")
	assert.Contains(t, logOutput, "duration_ms=150")
	assert.Contains(t, logOutput, "user_id=user123")
	assert.Contains(t, logOutput, "API request")
}

func TestStandardLogger_LogBusinessEvent(t *testing.T) {
	logger, buf := setupTestLogger("info", "development")

	details := map[string]interface{}{
		"symbol":     "BTC/USD",
		"exchange":   "binance",
		"profit_pct": 2.5,
	}

	logger.LogBusinessEvent("arbitrage_opportunity", details)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "event=business_event")
	assert.Contains(t, logOutput, "type=arbitrage_opportunity")
	assert.Contains(t, logOutput, "symbol=BTC/USD")
	assert.Contains(t, logOutput, "exchange=binance")
	assert.Contains(t, logOutput, "profit_pct=2.5")
	assert.Contains(t, logOutput, "Business event")
}

// Test edge cases
func TestStandardLogger_EmptyValues(t *testing.T) {
	logger := NewStandardLogger("info", "development")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Test with empty strings
	logger.WithComponent("").Info("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "test message")
}

func TestStandardLogger_NilMetrics(t *testing.T) {
	logger := NewStandardLogger("info", "development")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Test with nil metrics
	logger.WithMetrics(nil).Info("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "test message")
}

func TestStandardLogger_JSONFormatter(t *testing.T) {
	logger := NewStandardLogger("info", "production")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	logger.WithService("test-service").Info("test message")

	logOutput := buf.String()
	// Should be valid JSON
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &jsonData)
	require.NoError(t, err, "Output should be valid JSON")

	assert.Equal(t, "test message", jsonData["message"])
	assert.Equal(t, "info", jsonData["level"])
	assert.Equal(t, "test-service", jsonData["service"])
}
