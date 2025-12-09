package middleware

import (
	"fmt"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// TelemetryMiddleware creates a Gin middleware for Sentry tracing.
// It initializes the Sentry hub for each request.
//
// Returns:
//
//	gin.HandlerFunc: Gin handler.
func TelemetryMiddleware() gin.HandlerFunc {
	return sentrygin.New(sentrygin.Options{
		Repanic: true,
	})
}

// HealthCheckTelemetryMiddleware creates a Gin middleware for health check endpoints.
// It tags the transaction as a health check.
//
// Returns:
//
//	gin.HandlerFunc: Gin handler.
func HealthCheckTelemetryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For now, we just pass through as we want to monitor health checks too
		// But we can tag them if needed
		if hub := sentrygin.GetHubFromContext(c); hub != nil {
			hub.Scope().SetTag("transaction_type", "health_check")
		}
		c.Next()
	}
}

// RecordError records an error on the current span/hub.
//
// Parameters:
//
//	c: Gin context.
//	err: Error to record.
//	description: Description of the error.
func RecordError(c *gin.Context, err error, description string) {
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.CaptureException(err)
		if span := sentry.TransactionFromContext(c.Request.Context()); span != nil {
			span.Status = sentry.SpanStatusInternalError
		}
	}
}

// StartSpan starts a new span or adds a breadcrumb.
// This is a wrapper to maintain compatibility with existing code but using Sentry.
//
// Parameters:
//
//	c: Gin context.
//	name: Span name.
func StartSpan(c *gin.Context, name string) {
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		// Sentry Gin middleware starts the transaction automatically.
		// If we want custom spans, we can use sentry.StartSpan but it requires managing the span lifecycle.
		// For now, we can just add breadcrumbs or do nothing as the transaction is already running.
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "span",
			Message:  name,
			Level:    sentry.LevelInfo,
		}, nil)
	}
}

// AddSpanAttribute adds an attribute to the current span.
//
// Parameters:
//
//	c: Gin context.
//	key: Attribute key.
//	value: Attribute value.
func AddSpanAttribute(c *gin.Context, key string, value interface{}) {
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.Scope().SetTag(key, fmt.Sprint(value))
	}
}
