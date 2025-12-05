package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
)

const (
	// Service information
	ServiceName    = "github.com/irfandi/celebrum-ai-go"
	ServiceVersion = "1.0.0"
)

// TelemetryConfig holds configuration for telemetry
type TelemetryConfig struct {
	Enabled     bool
	DSN         string
	Environment string
	Release     string
	SampleRate  float64
}

// DefaultConfig returns default telemetry configuration
func DefaultConfig() *TelemetryConfig {
	return &TelemetryConfig{
		Enabled:     true,
		DSN:         "", // Should be provided via env
		Environment: "development",
		Release:     ServiceVersion,
		SampleRate:  0.2,
	}
}

// InitTelemetry initializes Sentry
func InitTelemetry(config TelemetryConfig) error {
	if !config.Enabled {
		return nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              config.DSN,
		Environment:      config.Environment,
		Release:          config.Release,
		TracesSampleRate: config.SampleRate,
		AttachStacktrace: true,
	})
	if err != nil {
		return fmt.Errorf("sentry init: %w", err)
	}

	return nil
}

// Flush flushes buffered events
func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

// Provider holds the telemetry provider
type Provider struct {
	Shutdown func(context.Context) error
	logger   *slog.Logger
}

// Shutdown shuts down the global telemetry provider
func Shutdown() error {
	sentry.Flush(2 * time.Second)
	return nil
}

// Logger returns the global slog.Logger instance for application logging
func Logger() *slog.Logger {
	return slog.Default()
}

// GetLogger is an alias for Logger, kept for compatibility
func GetLogger() *slog.Logger {
	return Logger()
}
