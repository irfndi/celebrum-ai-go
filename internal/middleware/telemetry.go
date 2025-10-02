package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/irfandi/celebrum-ai-go/internal/telemetry"
)

// Package middleware provides HTTP middleware components for authentication,
// authorization, telemetry, and other cross-cutting concerns.

// TelemetryMiddleware creates a Gin middleware for OpenTelemetry tracing
func TelemetryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip health check endpoints if they're causing issues
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/ready" || c.Request.URL.Path == "/live" {
			c.Next()
			return
		}

		tracer := telemetry.GetHTTPTracer()
		ctx := c.Request.Context()

		// Extract trace context from headers
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(c.Request.Header))

		// Create span attributes
		attrs := []attribute.KeyValue{
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.scheme", c.Request.URL.Scheme),
			attribute.String("http.host", c.Request.Host),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("http.client_ip", c.ClientIP()),
		}

		// Add route path if available
		if routePath := c.FullPath(); routePath != "" {
			attrs = append(attrs, attribute.String("http.route", routePath))
		}

		// Start span
		ctx, span := tracer.Start(
			ctx,
			fmt.Sprintf("HTTP %s %s", c.Request.Method, c.Request.URL.Path),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attrs...),
		)
		defer span.End()

		// Create new request with span context
		c.Request = c.Request.WithContext(ctx)

		// Record request start time
		start := time.Now()

		// Process request
		c.Next()

		// Record response attributes
		statusCode := c.Writer.Status()
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int64("http.response.time_ms", time.Since(start).Milliseconds()),
			attribute.Int64("http.response.size_bytes", int64(c.Writer.Size())),
		)

		// Set span status based on HTTP status code
		if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
			span.RecordError(fmt.Errorf("HTTP %d", statusCode))
		} else {
			span.SetStatus(codes.Ok, fmt.Sprintf("HTTP %d", statusCode))
		}

		// Add response headers as attributes
		contentType := c.Writer.Header().Get("Content-Type")
		if contentType != "" {
			span.SetAttributes(attribute.String("http.response.header.content_type", contentType))
		}
	}
}

// RecordError records an error on the current span
func RecordError(c *gin.Context, err error, description string) {
	span := trace.SpanFromContext(c.Request.Context())
	if span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, description)
	}
}

// AddSpanAttribute adds an attribute to the current span
func AddSpanAttribute(c *gin.Context, key string, value interface{}) {
	span := trace.SpanFromContext(c.Request.Context())
	if span.IsRecording() {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String(key, v))
		case int:
			span.SetAttributes(attribute.Int(key, v))
		case int64:
			span.SetAttributes(attribute.Int64(key, v))
		case float64:
			span.SetAttributes(attribute.Float64(key, v))
		case bool:
			span.SetAttributes(attribute.Bool(key, v))
		default:
			span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", value)))
		}
	}
}

// StartSpan starts a new span for the given context
func StartSpan(c *gin.Context, name string) (context.Context, trace.Span) {
	tracer := telemetry.GetHTTPTracer()
	ctx, span := tracer.Start(c.Request.Context(), name, trace.WithSpanKind(trace.SpanKindServer))
	c.Request = c.Request.WithContext(ctx)
	return ctx, span
}

// HealthCheckTelemetryMiddleware adds telemetry specifically for health check endpoints
func HealthCheckTelemetryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer := telemetry.GetHTTPTracer()
		ctx := c.Request.Context()

		// Start span for health check
		ctx, span := tracer.Start(
			ctx,
			fmt.Sprintf("Health %s", c.Request.URL.Path),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// Set initial attributes
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.host", c.Request.Host),
			attribute.String("span.type", "health_check"),
		)

		// Create new request with span context
		c.Request = c.Request.WithContext(ctx)

		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Record response attributes
		statusCode := c.Writer.Status()
		responseTime := time.Since(start)

		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int64("http.response.time_ms", responseTime.Milliseconds()),
			attribute.String("health.status", getHealthStatusFromCode(statusCode)),
		)

		// Set span status based on HTTP status code
		if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("Health check failed: HTTP %d", statusCode))
			span.RecordError(fmt.Errorf("health check endpoint returned %d", statusCode))
		} else {
			span.SetStatus(codes.Ok, fmt.Sprintf("Health check passed: HTTP %d", statusCode))
		}
	}
}

// getHealthStatusFromCode returns a human-readable status based on HTTP code
func getHealthStatusFromCode(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "healthy"
	case code >= 400 && code < 500:
		return "client_error"
	case code >= 500:
		return "server_error"
	default:
		return "unknown"
	}
}
