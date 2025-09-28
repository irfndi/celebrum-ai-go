package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/irfandi/celebrum-ai-go/internal/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.26.0"
	otelTrace "go.opentelemetry.io/otel/trace"
)

const (
	// Service information
	ServiceName    = "github.com/irfandi/celebrum-ai-go"
	ServiceVersion = "1.0.0"

	// Tracer names
	HTTPTracerName     = "github.com/irfandi/celebrum-ai-go/http"
	DatabaseTracerName = "github.com/irfandi/celebrum-ai-go/database"
	BusinessTracerName = "github.com/irfandi/celebrum-ai-go/business"
	CacheTracerName    = "github.com/irfandi/celebrum-ai-go/cache"
	ExternalTracerName = "github.com/irfandi/celebrum-ai-go/external"
)

// TelemetryConfig holds configuration for telemetry
type TelemetryConfig struct {
	Enabled        bool
	OTLPEndpoint   string
	ServiceName    string
	ServiceVersion string
	Environment    string
	SampleRate     float64
	BatchTimeout   time.Duration
	MaxExportBatch int
	MaxQueueSize   int
	LogLevel       string
}

// DefaultConfig returns default telemetry configuration
func DefaultConfig() *TelemetryConfig {
	return &TelemetryConfig{
		Enabled:        true,
		OTLPEndpoint:   "http://localhost:4318",
		ServiceName:    ServiceName,
		ServiceVersion: ServiceVersion,
		Environment:    "development",
		SampleRate:     1.0,
		BatchTimeout:   5 * time.Second,
		MaxExportBatch: 512,
		MaxQueueSize:   2048,
		LogLevel:       "info",
	}
}

// Provider holds the telemetry provider
type Provider struct {
	Shutdown func(context.Context) error
	logger   *slog.Logger
}

// GetTracer returns a tracer for the given name
func GetTracer(name string) otelTrace.Tracer {
	return otel.Tracer(name)
}

// GetHTTPTracer returns the HTTP tracer
func GetHTTPTracer() otelTrace.Tracer {
	return GetTracer(HTTPTracerName)
}

// GetDatabaseTracer returns the database tracer
func GetDatabaseTracer() otelTrace.Tracer {
	return GetTracer(DatabaseTracerName)
}

// GetBusinessTracer returns the business logic tracer
func GetBusinessTracer() otelTrace.Tracer {
	return GetTracer(BusinessTracerName)
}

// GetCacheTracer returns the cache tracer
func GetCacheTracer() otelTrace.Tracer {
	return GetTracer(CacheTracerName)
}

// GetExternalTracer returns the external service tracer
func GetExternalTracer() otelTrace.Tracer {
	return GetTracer(ExternalTracerName)
}

// Helper functions for common span operations

// StartSpan starts a new span with the given tracer and name
func StartSpan(ctx context.Context, tracer otelTrace.Tracer, name string, opts ...otelTrace.SpanStartOption) (context.Context, otelTrace.Span) {
	slog.Debug("Creating span", "name", name, "tracer", fmt.Sprintf("%T", tracer))
	return tracer.Start(ctx, name, opts...)
}

// SetSpanAttributes sets attributes on a span
func SetSpanAttributes(span otelTrace.Span, attrs ...attribute.KeyValue) {
	span.SetAttributes(attrs...)
}

// RecordError records an error on a span
func RecordError(span otelTrace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetSpanStatus sets the status of a span
func SetSpanStatus(span otelTrace.Span, code codes.Code, description string) {
	span.SetStatus(code, description)
}

// Attribute helper functions
func StringAttribute(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

func StringSliceAttribute(key string, value []string) attribute.KeyValue {
	return attribute.StringSlice(key, value)
}

func Int64Attribute(key string, value int64) attribute.KeyValue {
	return attribute.Int64(key, value)
}

func Float64Attribute(key string, value float64) attribute.KeyValue {
	return attribute.Float64(key, value)
}

func BoolAttribute(key string, value bool) attribute.KeyValue {
	return attribute.Bool(key, value)
}

// Global provider for shutdown
var globalProvider *Provider
var globalLogger *logging.OTLPLogger

// Logger returns the global slog.Logger instance for application logging
// Falls back to slog.Default() if telemetry logging hasn't been initialized
func Logger() *slog.Logger {
	if globalLogger != nil {
		if l := globalLogger.Logger(); l != nil {
			return l
		}
	}
	return slog.Default()
}

// normalizeOTLPEndpoint ensures the endpoint is a valid base URL and returns host:port and URL path
func normalizeOTLPEndpoint(raw string) (endpointHostPort string, urlPath string, insecure bool, resolved string, err error) {
	if raw == "" {
		raw = "http://localhost:4318"
	}

	// Ensure we don't have double protocol prefix
	raw = strings.TrimPrefix(raw, "http://http://")
	raw = strings.TrimPrefix(raw, "https://https://")
	// Also handle URL-encoded protocols
	raw = strings.ReplaceAll(raw, "http://http%3A//", "http://")
	raw = strings.ReplaceAll(raw, "https://https%3A//", "https://")

	// URL decode the raw endpoint to handle any URL-encoded characters
	decoded, err := url.QueryUnescape(raw)
	if err == nil {
		raw = decoded
	}

	u, parseErr := url.Parse(raw)
	if parseErr != nil || u.Scheme == "" || u.Host == "" {
		return "", "", true, "", fmt.Errorf("invalid OTLPEndpoint: %q: %v", raw, parseErr)
	}

	insecure = (u.Scheme == "http")
	// otlptracehttp.New expects WithEndpoint(host[:port]) and path with WithURLPath
	endpointHostPort = u.Host

	path := u.Path
	if path == "" || path == "/" {
		path = "/v1/traces"
	}
	// if user passed full path ending with /v1/traces keep as-is, otherwise ensure it ends with /v1/traces
	if !strings.HasSuffix(path, "/v1/traces") {
		// trim trailing slash then append
		path = strings.TrimRight(path, "/") + "/v1/traces"
	}

	resolved = fmt.Sprintf("%s://%s%s", u.Scheme, endpointHostPort, path)
	return endpointHostPort, path, insecure, resolved, nil
}

// InitTelemetry initializes OpenTelemetry with simplified config
func InitTelemetry(config TelemetryConfig) error {
	ctx := context.Background()
	logger := slog.Default()

	// Create full config from simplified config
	fullConfig := &TelemetryConfig{
		Enabled:        config.Enabled,
		OTLPEndpoint:   config.OTLPEndpoint,
		ServiceName:    config.ServiceName,
		ServiceVersion: config.ServiceVersion,
		Environment:    "production",
		SampleRate:     1.0,
		BatchTimeout:   5 * time.Second,
		MaxExportBatch: 512,
		MaxQueueSize:   2048,
		LogLevel:       config.LogLevel,
	}

	provider, err := InitTelemetryWithProvider(ctx, fullConfig, logger)
	if err != nil {
		return err
	}

	// Initialize OpenTelemetry logging
	logConfig := logging.OTLPConfig{
		Enabled:        config.Enabled,
		Endpoint:       config.OTLPEndpoint,
		ServiceName:    config.ServiceName,
		ServiceVersion: config.ServiceVersion,
		Environment:    config.Environment,
		LogLevel:       config.LogLevel,
	}

	var logErr error
	globalLogger, logErr = logging.NewOTLPLogger(logConfig)
	if config.Enabled {
		if logErr != nil {
			logger.Error("Failed to initialize OTLP logger", "error", logErr)
			// Continue with fallback logger
		} else {
			logger.Info("OTLP logger initialized successfully")
		}
	}

	globalProvider = provider
	return nil
}

// InitTelemetryWithProvider is the original function renamed
func InitTelemetryWithProvider(ctx context.Context, config *TelemetryConfig, logger *slog.Logger) (*Provider, error) {
	logger.Info("Initializing telemetry",
		"enabled", config.Enabled,
		"otlp_endpoint", config.OTLPEndpoint,
		"service_name", config.ServiceName,
		"service_version", config.ServiceVersion)

	if !config.Enabled {
		logger.Info("Telemetry disabled")
		return &Provider{
			Shutdown: func(context.Context) error { return nil },
			logger:   logger,
		}, nil
	}

	// Debug: log the raw endpoint value
	logger.Debug("Raw OTLP endpoint from config", "endpoint", config.OTLPEndpoint)

	endpointHostPort, urlPath, insecureHTTP, resolved, nerr := normalizeOTLPEndpoint(config.OTLPEndpoint)
	if nerr != nil {
		logger.Error("Invalid OTLP endpoint", "error", nerr.Error(), "raw", config.OTLPEndpoint)
		return nil, nerr
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP HTTP exporter with proper endpoint configuration
	fullURL := fmt.Sprintf("%s://%s%s", map[bool]string{true: "http", false: "https"}[insecureHTTP], endpointHostPort, urlPath)
	logger.Debug("Creating OTLP exporter",
		"endpointHostPort", endpointHostPort,
		"urlPath", urlPath,
		"insecureHTTP", insecureHTTP,
		"full_url", fullURL,
		"raw_input", config.OTLPEndpoint)

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(fullURL),
	}

	otlpExporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create console exporter for debugging
	consoleExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create console exporter: %w", err)
	}

	// Create trace provider with both exporters
	tp := trace.NewTracerProvider(
		trace.WithBatcher(otlpExporter,
			trace.WithBatchTimeout(config.BatchTimeout),
			trace.WithMaxExportBatchSize(config.MaxExportBatch),
			trace.WithMaxQueueSize(config.MaxQueueSize),
		),
		trace.WithBatcher(consoleExporter), // Add console exporter for debugging
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(config.SampleRate)),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("OpenTelemetry initialized",
		"endpoint_raw", config.OTLPEndpoint,
		"endpoint_hostport", endpointHostPort,
		"url_path", urlPath,
		"resolved_url", resolved,
		"insecure_http", insecureHTTP,
		"service", config.ServiceName,
		"version", config.ServiceVersion,
		"environment", config.Environment,
		"debug_path_unescaped", strings.ReplaceAll(urlPath, "%2F", "/"),
	)

	return &Provider{
		Shutdown: tp.Shutdown,
		logger:   logger,
	}, nil
}

// GetLogger returns the global OTLP logger
func GetLogger() *logging.OTLPLogger {
	return globalLogger
}

// Shutdown shuts down the global telemetry provider
func Shutdown() error {
	if globalProvider != nil {
		return globalProvider.Shutdown(context.Background())
	}
	if globalLogger != nil {
		return globalLogger.Shutdown(context.Background())
	}
	return nil
}
