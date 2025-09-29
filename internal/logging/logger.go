package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger interface defines the common logging methods
// This interface is implemented by both the legacy logrus-based logger and the new OTLP logger
type Logger interface {
	WithService(serviceName string) *slog.Logger
	WithComponent(componentName string) *slog.Logger
	WithOperation(operationName string) *slog.Logger
	WithRequestID(requestID string) *slog.Logger
	WithUserID(userID string) *slog.Logger
	WithExchange(exchange string) *slog.Logger
	WithSymbol(symbol string) *slog.Logger
	WithError(err error) *slog.Logger
	WithMetrics(metrics map[string]interface{}) *slog.Logger
	LogStartup(serviceName string, version string, port int)
	LogShutdown(serviceName string, reason string)
	LogPerformanceMetrics(serviceName string, metrics map[string]interface{})
	LogResourceStats(serviceName string, stats map[string]interface{})
	LogCacheOperation(operation string, key string, hit bool, duration int64)
	LogDatabaseOperation(operation string, table string, duration int64, rowsAffected int64)
	LogAPIRequest(method string, path string, statusCode int, duration int64, userID string)
	LogBusinessEvent(eventType string, details map[string]interface{})
	Logger() *slog.Logger
}

// StandardLogger provides a standardized logging interface
type StandardLogger struct {
	logger Logger
}

// NewStandardLogger creates a new standardized logger based on configuration
func NewStandardLogger(logLevel string, environment string) *StandardLogger {
	// For now, return a basic logger - we'll integrate with OTLP in the main initialization
	// This maintains backward compatibility until the telemetry system is initialized
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: getSlogLevel(logLevel),
	}))

	return &StandardLogger{
		logger: &fallbackLogger{logger: logger},
	}
}

// NewStandardOTLPLogger creates a new standardized logger with OTLP support
func NewStandardOTLPLogger(config OTLPConfig) *StandardLogger {
	otlpLogger, err := NewOTLPLogger(config)
	if err != nil {
		// Fallback to basic logger if OTLP setup fails
		basic := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: getSlogLevel(config.LogLevel),
		}))
		return &StandardLogger{logger: &fallbackLogger{logger: basic}}
	}
	return &StandardLogger{logger: &otlpWrapper{logger: otlpLogger}}
}

// SetLogger sets the underlying logger implementation
func (l *StandardLogger) SetLogger(logger Logger) {
	l.logger = logger
}

// WithService creates a logger with service context
func (l *StandardLogger) WithService(serviceName string) *slog.Logger {
	return l.logger.WithService(serviceName)
}

// WithComponent creates a logger with component context
func (l *StandardLogger) WithComponent(componentName string) *slog.Logger {
	return l.logger.WithComponent(componentName)
}

// WithOperation creates a logger with operation context
func (l *StandardLogger) WithOperation(operationName string) *slog.Logger {
	return l.logger.WithOperation(operationName)
}

// WithRequestID creates a logger with request ID context
func (l *StandardLogger) WithRequestID(requestID string) *slog.Logger {
	return l.logger.WithRequestID(requestID)
}

// WithUserID creates a logger with user ID context
func (l *StandardLogger) WithUserID(userID string) *slog.Logger {
	return l.logger.WithUserID(userID)
}

// WithExchange creates a logger with exchange context
func (l *StandardLogger) WithExchange(exchange string) *slog.Logger {
	return l.logger.WithExchange(exchange)
}

// WithSymbol creates a logger with symbol context
func (l *StandardLogger) WithSymbol(symbol string) *slog.Logger {
	return l.logger.WithSymbol(symbol)
}

// WithError creates a logger with error context
func (l *StandardLogger) WithError(err error) *slog.Logger {
	return l.logger.WithError(err)
}

// WithMetrics creates a logger with metrics context
func (l *StandardLogger) WithMetrics(metrics map[string]interface{}) *slog.Logger {
	return l.logger.WithMetrics(metrics)
}

// LogStartup logs application startup information
func (l *StandardLogger) LogStartup(serviceName string, version string, port int) {
	l.logger.LogStartup(serviceName, version, port)
}

// LogShutdown logs application shutdown information
func (l *StandardLogger) LogShutdown(serviceName string, reason string) {
	l.logger.LogShutdown(serviceName, reason)
}

// LogPerformanceMetrics logs performance metrics in a standardized format
func (l *StandardLogger) LogPerformanceMetrics(serviceName string, metrics map[string]interface{}) {
	l.logger.LogPerformanceMetrics(serviceName, metrics)
}

// LogResourceStats logs resource statistics in a standardized format
func (l *StandardLogger) LogResourceStats(serviceName string, stats map[string]interface{}) {
	l.logger.LogResourceStats(serviceName, stats)
}

// LogCacheOperation logs cache operations in a standardized format
func (l *StandardLogger) LogCacheOperation(operation string, key string, hit bool, duration int64) {
	l.logger.LogCacheOperation(operation, key, hit, duration)
}

// LogDatabaseOperation logs database operations in a standardized format
func (l *StandardLogger) LogDatabaseOperation(operation string, table string, duration int64, rowsAffected int64) {
	l.logger.LogDatabaseOperation(operation, table, duration, rowsAffected)
}

// LogAPIRequest logs API requests in a standardized format
func (l *StandardLogger) LogAPIRequest(method string, path string, statusCode int, duration int64, userID string) {
	l.logger.LogAPIRequest(method, path, statusCode, duration, userID)
}

// LogBusinessEvent logs business events in a standardized format
func (l *StandardLogger) LogBusinessEvent(eventType string, details map[string]interface{}) {
	l.logger.LogBusinessEvent(eventType, details)
}

// Logger returns the underlying *slog.Logger
func (l *StandardLogger) Logger() *slog.Logger {
	return l.logger.Logger()
}

// getSlogLevel converts string level to slog.Level
func getSlogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ParseLogrusLevel converts string level to logrus.Level
func ParseLogrusLevel(level string) logrus.Level {
	switch strings.ToLower(level) {
	case "debug":
		return logrus.DebugLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}

// otlpWrapper wraps OTLPLogger to implement Logger interface
type otlpWrapper struct {
	logger *OTLPLogger
}

func (o *otlpWrapper) WithService(serviceName string) *slog.Logger {
	return o.logger.logger.With("service", serviceName)
}

func (o *otlpWrapper) WithComponent(componentName string) *slog.Logger {
	return o.logger.logger.With("component", componentName)
}

func (o *otlpWrapper) WithOperation(operationName string) *slog.Logger {
	return o.logger.logger.With("operation", operationName)
}

func (o *otlpWrapper) WithRequestID(requestID string) *slog.Logger {
	return o.logger.logger.With("request_id", requestID)
}

func (o *otlpWrapper) WithUserID(userID string) *slog.Logger {
	return o.logger.logger.With("user_id", userID)
}

func (o *otlpWrapper) WithExchange(exchange string) *slog.Logger {
	return o.logger.logger.With("exchange", exchange)
}

func (o *otlpWrapper) WithSymbol(symbol string) *slog.Logger {
	return o.logger.logger.With("symbol", symbol)
}

func (o *otlpWrapper) WithError(err error) *slog.Logger {
	return o.logger.logger.With("error", err.Error())
}

func (o *otlpWrapper) WithMetrics(metrics map[string]interface{}) *slog.Logger {
	return o.logger.logger.With("metrics", metrics)
}

func (o *otlpWrapper) LogStartup(serviceName string, version string, port int) {
	o.logger.logger.Info("Application startup",
		"service", serviceName,
		"version", version,
		"port", port,
		"event", "startup",
	)
}

func (o *otlpWrapper) LogShutdown(serviceName string, reason string) {
	o.logger.logger.Info("Application shutdown",
		"service", serviceName,
		"reason", reason,
		"event", "shutdown",
	)
}

func (o *otlpWrapper) LogPerformanceMetrics(serviceName string, metrics map[string]interface{}) {
	o.logger.logger.Info("Performance metrics",
		"service", serviceName,
		"metrics", metrics,
		"event", "performance",
	)
}

func (o *otlpWrapper) LogResourceStats(serviceName string, stats map[string]interface{}) {
	o.logger.logger.Info("Resource statistics",
		"service", serviceName,
		"stats", stats,
		"event", "resource",
	)
}

func (o *otlpWrapper) LogCacheOperation(operation string, key string, hit bool, duration int64) {
	o.logger.logger.Info("Cache operation",
		"operation", operation,
		"key", key,
		"hit", hit,
		"duration_ms", duration,
		"event", "cache",
	)
}

func (o *otlpWrapper) LogDatabaseOperation(operation string, table string, duration int64, rowsAffected int64) {
	o.logger.logger.Info("Database operation",
		"operation", operation,
		"table", table,
		"duration_ms", duration,
		"rows_affected", rowsAffected,
		"event", "database",
	)
}

func (o *otlpWrapper) LogAPIRequest(method string, path string, statusCode int, duration int64, userID string) {
	o.logger.logger.Info("API request",
		"method", method,
		"path", path,
		"status", statusCode,
		"duration_ms", duration,
		"user_id", userID,
		"event", "api",
	)
}

func (o *otlpWrapper) LogBusinessEvent(eventType string, details map[string]interface{}) {
	o.logger.logger.Info("Business event",
		"event_type", eventType,
		"details", details,
		"event", "business",
	)
}

func (o *otlpWrapper) Logger() *slog.Logger {
	return o.logger.logger
}

// fallbackLogger is a simple implementation that uses slog directly
// This is used as a fallback when OTLP is not configured
type fallbackLogger struct {
	logger *slog.Logger
}

func (f *fallbackLogger) WithService(serviceName string) *slog.Logger {
	return f.logger.With("service", serviceName)
}

func (f *fallbackLogger) WithComponent(componentName string) *slog.Logger {
	return f.logger.With("component", componentName)
}

func (f *fallbackLogger) WithOperation(operationName string) *slog.Logger {
	return f.logger.With("operation", operationName)
}

func (f *fallbackLogger) WithRequestID(requestID string) *slog.Logger {
	return f.logger.With("request_id", requestID)
}

func (f *fallbackLogger) WithUserID(userID string) *slog.Logger {
	return f.logger.With("user_id", userID)
}

func (f *fallbackLogger) WithExchange(exchange string) *slog.Logger {
	return f.logger.With("exchange", exchange)
}

func (f *fallbackLogger) WithSymbol(symbol string) *slog.Logger {
	return f.logger.With("symbol", symbol)
}

func (f *fallbackLogger) WithError(err error) *slog.Logger {
	return f.logger.With("error", err.Error())
}

func (f *fallbackLogger) WithMetrics(metrics map[string]interface{}) *slog.Logger {
	return f.logger.With("metrics", metrics)
}

func (f *fallbackLogger) LogStartup(serviceName string, version string, port int) {
	f.logger.Info("Application startup",
		"service", serviceName,
		"version", version,
		"port", port,
		"event", "startup",
	)
}

func (f *fallbackLogger) LogShutdown(serviceName string, reason string) {
	f.logger.Info("Application shutdown",
		"service", serviceName,
		"reason", reason,
		"event", "shutdown",
	)
}

func (f *fallbackLogger) LogPerformanceMetrics(serviceName string, metrics map[string]interface{}) {
	f.logger.Info("Performance metrics",
		"service", serviceName,
		"metrics", metrics,
		"event", "performance",
	)
}

func (f *fallbackLogger) LogResourceStats(serviceName string, stats map[string]interface{}) {
	f.logger.Info("Resource statistics",
		"service", serviceName,
		"stats", stats,
		"event", "resource",
	)
}

func (f *fallbackLogger) LogCacheOperation(operation string, key string, hit bool, duration int64) {
	f.logger.Info("Cache operation",
		"operation", operation,
		"key", key,
		"hit", hit,
		"duration_ms", duration,
		"event", "cache",
	)
}

func (f *fallbackLogger) LogDatabaseOperation(operation string, table string, duration int64, rowsAffected int64) {
	f.logger.Info("Database operation",
		"operation", operation,
		"table", table,
		"duration_ms", duration,
		"rows_affected", rowsAffected,
		"event", "database",
	)
}

func (f *fallbackLogger) LogAPIRequest(method string, path string, statusCode int, duration int64, userID string) {
	f.logger.Info("API request",
		"method", method,
		"path", path,
		"status", statusCode,
		"duration_ms", duration,
		"user_id", userID,
		"event", "api",
	)
}

func (f *fallbackLogger) LogBusinessEvent(eventType string, details map[string]interface{}) {
	f.logger.Info("Business event",
		"event_type", eventType,
		"details", details,
		"event", "business",
	)
}

func (f *fallbackLogger) Logger() *slog.Logger {
	return f.logger
}
