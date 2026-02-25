package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// New creates a new slog.Logger based on the environment.
// For "production", it returns a JSON handler.
// For other environments, it returns a text handler with debug level.
func New(env string) *slog.Logger {
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	return slog.New(handler)
}

// WithTraceContext returns a slog.Attr containing trace_id and span_id if available in the context.
func WithTraceContext(ctx context.Context) slog.Attr {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return slog.Attr{}
	}
	sc := span.SpanContext()
	return slog.Group("trace",
		slog.String("trace_id", sc.TraceID().String()),
		slog.String("span_id", sc.SpanID().String()),
	)
}
