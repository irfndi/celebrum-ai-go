package metrics

import (
	"bytes"
	"testing"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/logging"
	"github.com/stretchr/testify/assert"
)

func TestNewMetricsCollector(t *testing.T) {
	logger := logging.NewStandardLogger("info", "development")
	collector := NewMetricsCollector(logger, "test-service")

	assert.NotNil(t, collector)
}

func TestMetricsCollector_RecordCounter(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	tags := map[string]string{"key": "value"}
	collector.RecordCounter("test_counter", 1.0, tags)

	// Test with nil tags
	collector.RecordCounter("test_counter_nil", 2.0, nil)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "test_counter")
	assert.Contains(t, logOutput, "counter")
}

func TestMetricsCollector_RecordGauge(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	tags := map[string]string{"key": "value"}
	collector.RecordGauge("test_gauge", 10.5, "units", tags)

	// Test with zero value
	collector.RecordGauge("test_gauge_zero", 0.0, "units", nil)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "test_gauge")
	assert.Contains(t, logOutput, "gauge")
}

func TestMetricsCollector_RecordTiming(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	tags := map[string]string{"operation": "test"}
	duration := 100 * time.Millisecond
	collector.RecordTiming("test_timing", duration, tags)

	// Test with zero duration
	collector.RecordTiming("test_timing_zero", 0, nil)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "test_timing")
	assert.Contains(t, logOutput, "timing")
}

func TestMetricsCollector_RecordHistogram(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	tags := map[string]string{"bucket": "test"}
	collector.RecordHistogram("request_size", 1024, "bytes", tags)

	// Test with zero value
	collector.RecordHistogram("response_time", 0.0, "ms", nil)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "request_size")
	assert.Contains(t, logOutput, "histogram")
}

func TestMetricsCollector_RecordBusinessMetric(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	tags := map[string]string{"exchange": "binance"}
	fields := map[string]interface{}{
		"buy_price":  100.50,
		"sell_price": 101.00,
		"volume":     1000,
	}

	collector.RecordBusinessMetric("arbitrage_opportunity", 0.5, "percent", tags, fields)

	// Test with nil fields
	collector.RecordBusinessMetric("simple_metric", 42.0, "units", nil, nil)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "arbitrage_opportunity")
	assert.Contains(t, logOutput, "gauge")
}

func TestMetricsCollector_RecordAPIRequestMetrics(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	duration := 150 * time.Millisecond
	collector.RecordAPIRequestMetrics("GET", "/api/symbols", 200, duration, "user123")

	// Test without user ID
	collector.RecordAPIRequestMetrics("POST", "/api/orders", 201, duration, "")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "api_requests_total")
	assert.Contains(t, logOutput, "api_request_duration")
}

func TestMetricsCollector_RecordDatabaseMetrics(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	duration := 50 * time.Millisecond
	collector.RecordDatabaseMetrics("SELECT", "symbols", duration, 10, true)

	// Test failed operation
	collector.RecordDatabaseMetrics("INSERT", "orders", duration, -1, false)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "database_operations_total")
	assert.Contains(t, logOutput, "database_operation_duration")
}

func TestMetricsCollector_RecordCacheMetrics(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	duration := 5 * time.Millisecond
	collector.RecordCacheMetrics("GET", "symbols:binance", true, duration)

	// Test cache miss
	collector.RecordCacheMetrics("GET", "symbols:coinbase", false, duration)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "cache_operations_total")
	assert.Contains(t, logOutput, "cache_operation_duration")
}

func TestMetricsCollector_RecordExchangeMetrics(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	duration := 200 * time.Millisecond
	collector.RecordExchangeMetrics("binance", "fetch_symbols", "BTCUSDT", duration, true)

	// Test failed operation without symbol
	collector.RecordExchangeMetrics("coinbase", "fetch_orderbook", "", duration, false)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "exchange_operations_total")
	assert.Contains(t, logOutput, "exchange_operation_duration")
}

func TestMetricsCollector_RecordArbitrageMetrics(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	collector.RecordArbitrageMetrics("BTCUSDT", 0.5, "binance", "coinbase")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "arbitrage_opportunities_total")
	assert.Contains(t, logOutput, "BTCUSDT")
}

// Edge case tests

func TestMetricsCollector_EmptyTags(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Test with nil tags
	collector.RecordCounter("test_counter", 1.0, nil)
	collector.RecordGauge("test_gauge", 50.0, "units", nil)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "test_counter")
	assert.Contains(t, logOutput, "test_gauge")
	// Should still contain service tag
	assert.Contains(t, logOutput, "test-service")
}

func TestMetricsCollector_ZeroValues(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Test with zero duration
	collector.RecordTiming("zero_timing", 0, map[string]string{"test": "zero"})

	// Test with negative values
	collector.RecordGauge("negative_gauge", -25.5, "units", map[string]string{"type": "delta"})

	logOutput := buf.String()
	assert.Contains(t, logOutput, "zero_timing")
	assert.Contains(t, logOutput, "negative_gauge")
}

func TestMetricsCollector_LargeValues(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Test with large values
	collector.RecordCounter("large_counter", 1000000.0, map[string]string{"scale": "large"})
	collector.RecordGauge("large_gauge", 999999.99, "bytes", map[string]string{"size": "huge"})

	logOutput := buf.String()
	assert.Contains(t, logOutput, "large_counter")
	assert.Contains(t, logOutput, "large_gauge")
}

func TestMetricsCollector_SpecialCharacters(t *testing.T) {
	logger := logging.NewStandardLogger("debug", "development")
	collector := NewMetricsCollector(logger, "test-service")

	// Capture log output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Test with special characters in tags
	tags := map[string]string{
		"symbol":   "BTC/USDT",
		"exchange": "binance-us",
		"type":     "spot_trading",
	}
	collector.RecordCounter("special_chars", 1.0, tags)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "special_chars")
	assert.Contains(t, logOutput, "BTC/USDT")
}
