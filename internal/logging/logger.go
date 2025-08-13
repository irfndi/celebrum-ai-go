package logging

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// StandardLogger provides a configured logrus logger instance
type StandardLogger struct {
	*logrus.Logger
}

// NewStandardLogger creates a new standardized logger with consistent configuration
func NewStandardLogger(logLevel string, environment string) *StandardLogger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set formatter based on environment
	if environment == "production" {
		// JSON formatter for production
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		// Text formatter for development
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}

	// Set output to stdout
	logger.SetOutput(os.Stdout)

	return &StandardLogger{Logger: logger}
}

// WithService creates a logger with service context
func (l *StandardLogger) WithService(serviceName string) *logrus.Entry {
	return l.WithField("service", serviceName)
}

// WithComponent creates a logger with component context
func (l *StandardLogger) WithComponent(componentName string) *logrus.Entry {
	return l.WithField("component", componentName)
}

// WithOperation creates a logger with operation context
func (l *StandardLogger) WithOperation(operationName string) *logrus.Entry {
	return l.WithField("operation", operationName)
}

// WithRequestID creates a logger with request ID context
func (l *StandardLogger) WithRequestID(requestID string) *logrus.Entry {
	return l.WithField("request_id", requestID)
}

// WithUserID creates a logger with user ID context
func (l *StandardLogger) WithUserID(userID string) *logrus.Entry {
	return l.WithField("user_id", userID)
}

// WithExchange creates a logger with exchange context
func (l *StandardLogger) WithExchange(exchange string) *logrus.Entry {
	return l.WithField("exchange", exchange)
}

// WithSymbol creates a logger with symbol context
func (l *StandardLogger) WithSymbol(symbol string) *logrus.Entry {
	return l.WithField("symbol", symbol)
}

// WithError creates a logger with error context
func (l *StandardLogger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// WithMetrics creates a logger with metrics context
func (l *StandardLogger) WithMetrics(metrics map[string]interface{}) *logrus.Entry {
	return l.WithFields(logrus.Fields(metrics))
}

// LogStartup logs application startup information
func (l *StandardLogger) LogStartup(serviceName string, version string, port int) {
	l.WithFields(logrus.Fields{
		"service": serviceName,
		"version": version,
		"port":    port,
		"event":   "startup",
	}).Info("Service starting")
}

// LogShutdown logs application shutdown information
func (l *StandardLogger) LogShutdown(serviceName string, reason string) {
	l.WithFields(logrus.Fields{
		"service": serviceName,
		"reason":  reason,
		"event":   "shutdown",
	}).Info("Service shutting down")
}

// LogPerformanceMetrics logs performance metrics in a standardized format
func (l *StandardLogger) LogPerformanceMetrics(serviceName string, metrics map[string]interface{}) {
	l.WithFields(logrus.Fields{
		"service": serviceName,
		"event":   "performance_metrics",
		"metrics": metrics,
	}).Debug("Performance metrics collected")
}

// LogResourceStats logs resource statistics in a standardized format
func (l *StandardLogger) LogResourceStats(serviceName string, stats map[string]interface{}) {
	l.WithFields(logrus.Fields{
		"service": serviceName,
		"event":   "resource_stats",
		"stats":   stats,
	}).Info("Resource statistics")
}

// LogCacheOperation logs cache operations in a standardized format
func (l *StandardLogger) LogCacheOperation(operation string, key string, hit bool, duration int64) {
	l.WithFields(logrus.Fields{
		"event":     "cache_operation",
		"operation": operation,
		"key":       key,
		"hit":       hit,
		"duration_ms": duration,
	}).Debug("Cache operation")
}

// LogDatabaseOperation logs database operations in a standardized format
func (l *StandardLogger) LogDatabaseOperation(operation string, table string, duration int64, rowsAffected int64) {
	l.WithFields(logrus.Fields{
		"event":         "database_operation",
		"operation":     operation,
		"table":         table,
		"duration_ms":   duration,
		"rows_affected": rowsAffected,
	}).Debug("Database operation")
}

// LogAPIRequest logs API requests in a standardized format
func (l *StandardLogger) LogAPIRequest(method string, path string, statusCode int, duration int64, userID string) {
	l.WithFields(logrus.Fields{
		"event":       "api_request",
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration_ms": duration,
		"user_id":     userID,
	}).Info("API request")
}

// LogBusinessEvent logs business events in a standardized format
func (l *StandardLogger) LogBusinessEvent(eventType string, details map[string]interface{}) {
	fields := logrus.Fields{
		"event": "business_event",
		"type":  eventType,
	}
	for k, v := range details {
		fields[k] = v
	}
	l.WithFields(fields).Info("Business event")
}