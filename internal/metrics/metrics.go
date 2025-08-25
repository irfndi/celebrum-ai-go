package metrics

import (
	"strconv"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/logging"
)

// MetricType represents the type of metric being recorded
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeTiming    MetricType = "timing"
)

// Metric represents a standardized metric structure
type Metric struct {
	Name      string                 `json:"name"`
	Type      MetricType             `json:"type"`
	Value     float64                `json:"value"`
	Unit      string                 `json:"unit"`
	Timestamp time.Time              `json:"timestamp"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// MetricsCollector provides standardized metrics collection
type MetricsCollector struct {
	logger      *logging.StandardLogger
	serviceName string
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger *logging.StandardLogger, serviceName string) *MetricsCollector {
	return &MetricsCollector{
		logger:      logger,
		serviceName: serviceName,
	}
}

// RecordCounter records a counter metric
func (mc *MetricsCollector) RecordCounter(name string, value float64, tags map[string]string) {
	metric := Metric{
		Name:      name,
		Type:      MetricTypeCounter,
		Value:     value,
		Unit:      "count",
		Timestamp: time.Now(),
		Tags:      mc.addServiceTag(tags),
	}
	mc.logMetric(metric)
}

// RecordGauge records a gauge metric
func (mc *MetricsCollector) RecordGauge(name string, value float64, unit string, tags map[string]string) {
	metric := Metric{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
		Tags:      mc.addServiceTag(tags),
	}
	mc.logMetric(metric)
}

// RecordTiming records a timing metric
func (mc *MetricsCollector) RecordTiming(name string, duration time.Duration, tags map[string]string) {
	metric := Metric{
		Name:      name,
		Type:      MetricTypeTiming,
		Value:     float64(duration.Milliseconds()),
		Unit:      "ms",
		Timestamp: time.Now(),
		Tags:      mc.addServiceTag(tags),
	}
	mc.logMetric(metric)
}

// RecordHistogram records a histogram metric
func (mc *MetricsCollector) RecordHistogram(name string, value float64, unit string, tags map[string]string) {
	metric := Metric{
		Name:      name,
		Type:      MetricTypeHistogram,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
		Tags:      mc.addServiceTag(tags),
	}
	mc.logMetric(metric)
}

// RecordBusinessMetric records a business-specific metric with additional fields
func (mc *MetricsCollector) RecordBusinessMetric(name string, value float64, unit string, tags map[string]string, fields map[string]interface{}) {
	metric := Metric{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
		Tags:      mc.addServiceTag(tags),
		Fields:    fields,
	}
	mc.logMetric(metric)
}

// addServiceTag adds the service name to tags
func (mc *MetricsCollector) addServiceTag(tags map[string]string) map[string]string {
	// Create a copy of the input map to avoid modifying the original
	result := make(map[string]string)
	for k, v := range tags {
		result[k] = v
	}
	result["service"] = mc.serviceName
	return result
}

// logMetric logs the metric using the standardized logger
func (mc *MetricsCollector) logMetric(metric Metric) {
	mc.logger.Logger().Debug("Metric recorded",
		"event", "metric",
		"metric", metric,
	)
}

// Performance metrics helpers

// RecordAPIRequestMetrics records standardized API request metrics
func (mc *MetricsCollector) RecordAPIRequestMetrics(method, endpoint string, statusCode int, duration time.Duration, userID string) {
	tags := map[string]string{
		"method":      method,
		"endpoint":    endpoint,
		"status_code": strconv.Itoa(statusCode),
	}
	if userID != "" {
		tags["user_id"] = userID
	}

	mc.RecordCounter("api_requests_total", 1, tags)
	mc.RecordTiming("api_request_duration", duration, tags)
}

// RecordDatabaseMetrics records standardized database operation metrics
func (mc *MetricsCollector) RecordDatabaseMetrics(operation, table string, duration time.Duration, rowsAffected int64, success bool) {
	tags := map[string]string{
		"operation": operation,
		"table":     table,
		"success":   "true",
	}
	if !success {
		tags["success"] = "false"
	}

	mc.RecordCounter("database_operations_total", 1, tags)
	mc.RecordTiming("database_operation_duration", duration, tags)
	if rowsAffected >= 0 {
		mc.RecordGauge("database_rows_affected", float64(rowsAffected), "rows", tags)
	}
}

// RecordCacheMetrics records standardized cache operation metrics
func (mc *MetricsCollector) RecordCacheMetrics(operation, key string, hit bool, duration time.Duration) {
	tags := map[string]string{
		"operation": operation,
		"hit":       "false",
	}
	if hit {
		tags["hit"] = "true"
	}

	mc.RecordCounter("cache_operations_total", 1, tags)
	mc.RecordTiming("cache_operation_duration", duration, tags)
}

// RecordExchangeMetrics records standardized exchange operation metrics
func (mc *MetricsCollector) RecordExchangeMetrics(exchange, operation, symbol string, duration time.Duration, success bool) {
	tags := map[string]string{
		"exchange":  exchange,
		"operation": operation,
		"success":   "true",
	}
	if symbol != "" {
		tags["symbol"] = symbol
	}
	if !success {
		tags["success"] = "false"
	}

	mc.RecordCounter("exchange_operations_total", 1, tags)
	mc.RecordTiming("exchange_operation_duration", duration, tags)
}

// RecordArbitrageMetrics records standardized arbitrage opportunity metrics
func (mc *MetricsCollector) RecordArbitrageMetrics(symbol string, profitPercent float64, buyExchange, sellExchange string) {
	tags := map[string]string{
		"symbol":        symbol,
		"buy_exchange":  buyExchange,
		"sell_exchange": sellExchange,
	}

	mc.RecordCounter("arbitrage_opportunities_total", 1, tags)
	mc.RecordGauge("arbitrage_profit_percent", profitPercent, "percent", tags)
}

// RecordSystemMetrics records standardized system resource metrics
func (mc *MetricsCollector) RecordSystemMetrics(memoryMB, goroutines int, cpuPercent float64) {
	tags := map[string]string{}

	mc.RecordGauge("system_memory_usage", float64(memoryMB), "MB", tags)
	mc.RecordGauge("system_goroutines", float64(goroutines), "count", tags)
	mc.RecordGauge("system_cpu_usage", cpuPercent, "percent", tags)
}

// RecordNotificationMetrics records standardized notification metrics
func (mc *MetricsCollector) RecordNotificationMetrics(notificationType, userID string, success bool) {
	tags := map[string]string{
		"type":    notificationType,
		"user_id": userID,
		"success": "true",
	}
	if !success {
		tags["success"] = "false"
	}

	mc.RecordCounter("notifications_sent_total", 1, tags)
}
