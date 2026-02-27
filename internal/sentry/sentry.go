package sentry

import (
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

	// Configure Sentry client options
	options := sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		ServerName:       serviceName,
		Release:          serviceVersion,
		AttachStacktrace: true,
		TracesSampleRate: 0.0, // Disable Sentry tracing, use OpenTelemetry instead
	}

	// Initialize Sentry
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
