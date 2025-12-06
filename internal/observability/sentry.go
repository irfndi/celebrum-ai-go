package observability

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/irfandi/celebrum-ai-go/internal/config"
)

// InitSentry configures the Sentry SDK using application config.
func InitSentry(cfg config.SentryConfig, fallbackRelease string, fallbackEnv string) error {
	if !cfg.Enabled || cfg.DSN == "" {
		return nil
	}

	release := cfg.Release
	if release == "" {
		release = fallbackRelease
	}

	environment := cfg.Environment
	if environment == "" {
		environment = fallbackEnv
	}

	return sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      environment,
		Release:          release,
		EnableTracing:    cfg.TracesSampleRate > 0,
		TracesSampleRate: cfg.TracesSampleRate,
		AttachStacktrace: true,
	})
}

// Flush drains buffered Sentry events within the provided context deadline.
func Flush(ctx context.Context) {
	timeout := 2 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout < 0 {
			timeout = 0
		}
	}
	sentry.Flush(timeout)
}

// CaptureException sends an exception to Sentry, using the hub in context when available.
func CaptureException(ctx context.Context, err error) {
	if err == nil {
		return
	}
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureException(err)
		return
	}
	sentry.CaptureException(err)
}
