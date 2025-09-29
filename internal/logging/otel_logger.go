package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// OTLPLogger provides OpenTelemetry logging capabilities
type OTLPLogger struct {
	logger   *slog.Logger
	provider *log.LoggerProvider
	shutdown func(context.Context) error
}

// OTLPConfig holds configuration for OpenTelemetry logging
type OTLPConfig struct {
	Enabled        bool
	Endpoint       string
	ServiceName    string
	ServiceVersion string
	Environment    string
	LogLevel       string
}

// NewOTLPLogger creates a new OpenTelemetry logger
func NewOTLPLogger(config OTLPConfig) (*OTLPLogger, error) {
	if !config.Enabled {
		// Return a basic slog logger that writes to stdout when OTLP is disabled
		return &OTLPLogger{
			logger:   slog.New(slog.NewJSONHandler(os.Stdout, nil)),
			shutdown: func(ctx context.Context) error { return nil },
		}, nil
	}

	ctx := context.Background()

	// Parse endpoint
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}

	// Create OTLP log exporter
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithURLPath("/v1/logs"),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create logger provider
	provider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
		log.WithResource(res),
	)

	// Get the OpenTelemetry logger
	otelLogger := provider.Logger(config.ServiceName)

	// Create a slog handler that uses OpenTelemetry
	handler := NewOTLPHandler(otelLogger)
	logger := slog.New(handler)

	return &OTLPLogger{
		logger:   logger,
		provider: provider,
		shutdown: provider.Shutdown,
	}, nil
}

// Shutdown gracefully shuts down the logger
func (l *OTLPLogger) Shutdown(ctx context.Context) error {
	if l.shutdown != nil {
		return l.shutdown(ctx)
	}
	return nil
}

// Logger returns the underlying slog.Logger
func (l *OTLPLogger) Logger() *slog.Logger {
	return l.logger
}

// OTLPLogger returns the underlying OpenTelemetry logger for testing
func (l *OTLPLogger) OTLPLogger() otellog.Logger {
	if l.provider != nil {
		return l.provider.Logger("test")
	}
	return nil
}

// OTLPHandler implements slog.Handler for OTLP logging
type OTLPHandler struct {
	logger otellog.Logger
}

// NewOTLPHandler creates a new OTLPHandler
func NewOTLPHandler(logger otellog.Logger) *OTLPHandler {
	return &OTLPHandler{logger: logger}
}

// Enabled implements slog.Handler.Enabled
func (h *OTLPHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

// Handle implements slog.Handler.Handle
func (h *OTLPHandler) Handle(ctx context.Context, record slog.Record) error {
	// Convert slog attributes to OpenTelemetry attributes
	attrs := make([]otellog.KeyValue, 0)
	record.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, otellog.String(a.Key, a.Value.String()))
		return true
	})

	// Create and emit log record
	logRecord := otellog.Record{}
	logRecord.SetTimestamp(record.Time)
	logRecord.SetObservedTimestamp(time.Now())
	logRecord.SetSeverity(convertSlogLevelToSeverity(record.Level))
	logRecord.SetBody(otellog.StringValue(record.Message))
	logRecord.AddAttributes(attrs...)

	h.logger.Emit(ctx, logRecord)

	return nil
}

// WithAttrs implements slog.Handler.WithAttrs
func (h *OTLPHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup implements slog.Handler.WithGroup
func (h *OTLPHandler) WithGroup(name string) slog.Handler {
	return h
}

// convertSlogLevelToSeverity converts slog.Level to otellog.Severity
func convertSlogLevelToSeverity(level slog.Level) otellog.Severity {
	switch level {
	case slog.LevelDebug:
		return otellog.SeverityDebug
	case slog.LevelInfo:
		return otellog.SeverityInfo
	case slog.LevelWarn:
		return otellog.SeverityWarn
	case slog.LevelError:
		return otellog.SeverityError
	default:
		return otellog.SeverityInfo
	}
}

// StandardOTLPLogger wrapper for OTLPLogger
type StandardOTLPLogger struct {
	*OTLPLogger
}

// WithService creates a logger with service context
func (l *StandardOTLPLogger) WithService(serviceName string) *slog.Logger {
	return l.logger.With("service", serviceName)
}

// WithComponent creates a logger with component context
func (l *StandardOTLPLogger) WithComponent(componentName string) *slog.Logger {
	return l.logger.With("component", componentName)
}

// WithOperation creates a logger with operation context
func (l *StandardOTLPLogger) WithOperation(operationName string) *slog.Logger {
	return l.logger.With("operation", operationName)
}

// WithRequestID creates a logger with request ID context
func (l *StandardOTLPLogger) WithRequestID(requestID string) *slog.Logger {
	return l.logger.With("request_id", requestID)
}

// WithUserID creates a logger with user ID context
func (l *StandardOTLPLogger) WithUserID(userID string) *slog.Logger {
	return l.logger.With("user_id", userID)
}

// WithExchange creates a logger with exchange context
func (l *StandardOTLPLogger) WithExchange(exchange string) *slog.Logger {
	return l.logger.With("exchange", exchange)
}

// WithSymbol creates a logger with symbol context
func (l *StandardOTLPLogger) WithSymbol(symbol string) *slog.Logger {
	return l.logger.With("symbol", symbol)
}

// WithError creates a logger with error context
func (l *StandardOTLPLogger) WithError(err error) *slog.Logger {
	return l.logger.With("error", err.Error())
}

// WithMetrics creates a logger with metrics context
func (l *StandardOTLPLogger) WithMetrics(metrics map[string]interface{}) *slog.Logger {
	return l.logger.With("metrics", metrics)
}

// LogStartup logs application startup information
func (l *StandardOTLPLogger) LogStartup(serviceName string, version string, port int) {
	l.logger.Info("Service starting",
		"event", "startup",
		"service", serviceName,
		"version", version,
		"port", port,
	)
}

// LogShutdown logs application shutdown information
func (l *StandardOTLPLogger) LogShutdown(serviceName string, reason string) {
	l.logger.Info("Service shutting down",
		"event", "shutdown",
		"service", serviceName,
		"reason", reason,
	)
}

// LogPerformanceMetrics logs performance metrics in a standardized format
func (l *StandardOTLPLogger) LogPerformanceMetrics(serviceName string, metrics map[string]interface{}) {
	l.logger.Debug("Performance metrics collected",
		"event", "performance_metrics",
		"service", serviceName,
		"metrics", metrics,
	)
}

// LogResourceStats logs resource statistics in a standardized format
func (l *StandardOTLPLogger) LogResourceStats(serviceName string, stats map[string]interface{}) {
	l.logger.Info("Resource statistics",
		"event", "resource_stats",
		"service", serviceName,
		"stats", stats,
	)
}

// LogCacheOperation logs cache operations in a standardized format
func (l *StandardOTLPLogger) LogCacheOperation(operation string, key string, hit bool, duration int64) {
	l.logger.Debug("Cache operation",
		"event", "cache_operation",
		"operation", operation,
		"key", key,
		"hit", hit,
		"duration_ms", duration,
	)
}

// LogDatabaseOperation logs database operations in a standardized format
func (l *StandardOTLPLogger) LogDatabaseOperation(operation string, table string, duration int64, rowsAffected int64) {
	l.logger.Debug("Database operation",
		"event", "database_operation",
		"operation", operation,
		"table", table,
		"duration_ms", duration,
		"rows_affected", rowsAffected,
	)
}

// LogAPIRequest logs API requests in a standardized format
func (l *StandardOTLPLogger) LogAPIRequest(method string, path string, statusCode int, duration int64, userID string) {
	l.logger.Info("API request",
		"event", "api_request",
		"method", method,
		"path", path,
		"status_code", statusCode,
		"duration_ms", duration,
		"user_id", userID,
	)
}

// LogBusinessEvent logs business events in a standardized format
func (l *StandardOTLPLogger) LogBusinessEvent(eventType string, details map[string]interface{}) {
	fields := []interface{}{
		"event", "business_event",
		"type", eventType,
	}

	for k, v := range details {
		fields = append(fields, k, v)
	}

	l.logger.Info("Business event", fields...)
}
