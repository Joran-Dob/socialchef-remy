package sentry

import (
	"net/http"

	"github.com/getsentry/sentry-go"
)

// HTTPMiddleware returns a middleware that captures errors and panics in HTTP handlers.
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a hub for this request
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}

		// Wrap response writer to capture status codes
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Add hub to context
		ctx := sentry.SetHubOnContext(r.Context(), hub)

		// Recover from panics
		defer func() {
			if err := recover(); err != nil {
				hub.Recover(err)
				wrapped.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(wrapped, r.WithContext(ctx))
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
