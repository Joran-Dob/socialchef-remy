package sentry

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
)

// Init initializes Sentry with the provided configuration.
// If DSN is empty, Sentry initialization is skipped and nil is returned.
func Init(dsn, env, serviceName, serviceVersion string) error {
	if dsn == "" {
		return nil
	}

	options := sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		ServerName:       serviceName,
		Release:          serviceVersion,
		AttachStacktrace: true,
		TracesSampleRate: 0.0, // Disable Sentry tracing, use OpenTelemetry instead
	}

	if err := sentry.Init(options); err != nil {
		return fmt.Errorf("failed to initialize Sentry: %w", err)
	}

	return nil
}

// Flush waits for all pending Sentry events to be sent.
// Call this during graceful shutdown.
func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

// Recover captures a panic and forwards it to Sentry.
// Should be used with defer in goroutines.
func Recover() {
	sentry.Recover()
}

// CaptureMessage sends a message to Sentry.
func CaptureMessage(msg string) {
	sentry.CaptureMessage(msg)
}

// CaptureException sends an error to Sentry.
func CaptureException(err error) {
	sentry.CaptureException(err)
}

// CaptureError sends an error to Sentry with additional context.
func CaptureError(err error, tags map[string]string) {
	if err == nil {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		sentry.CaptureException(err)
	})
}

// CaptureWithContext captures an error with context information.
func CaptureWithContext(ctx context.Context, err error) {
	if err == nil {
		return
	}
	
	hub := sentry.GetHubFromContext(ctx)
	if hub != nil {
		hub.CaptureException(err)
	} else {
		sentry.CaptureException(err)
	}
}

// AddBreadcrumb adds a breadcrumb to the current scope.
func AddBreadcrumb(category, message string, data map[string]interface{}) {
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Category: category,
		Message:  message,
		Data:     data,
	})
}
