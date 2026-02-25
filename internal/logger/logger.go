package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
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

	return slog.New(&otelHandler{handler: handler})
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

type otelHandler struct {
	handler slog.Handler
}

func (h *otelHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.handler.Enabled(ctx, l)
}

func (h *otelHandler) Handle(ctx context.Context, r slog.Record) error {
	// Always log to stdout first
	if err := h.handler.Handle(ctx, r); err != nil {
		return err
	}

	// Debug: log that we're trying to send to OTel
	fmt.Fprintf(os.Stderr, "[LOGGER DEBUG] Emitting log to OTel: %s\n", r.Message)

	// Check if logger provider is available
	if global.GetLoggerProvider() == nil {
		fmt.Fprintf(os.Stderr, "[LOGGER DEBUG] Logger provider is NIL!\n")
		return nil
	}
	// Also send to OpenTelemetry Logs
	logger := global.GetLoggerProvider().Logger("github.com/socialchef/remy")

	var otelRecord log.Record
	otelRecord.SetTimestamp(r.Time)
	otelRecord.SetBody(log.StringValue(r.Message))

	// Map slog level to OTel severity
	var severity log.Severity
	switch {
	case r.Level >= slog.LevelError:
		severity = log.SeverityError
	case r.Level >= slog.LevelWarn:
		severity = log.SeverityWarn
	case r.Level >= slog.LevelInfo:
		severity = log.SeverityInfo
	default:
		severity = log.SeverityDebug
	}
	otelRecord.SetSeverity(severity)
	otelRecord.SetSeverityText(r.Level.String())

	// Add attributes from the record
	r.Attrs(func(a slog.Attr) bool {
		otelRecord.AddAttributes(log.KeyValue{
			Key:   a.Key,
			Value: toOTelValue(a.Value),
		})
		return true
	})

	logger.Emit(ctx, otelRecord)
	return nil
}

func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *otelHandler) WithGroup(name string) slog.Handler {
	return &otelHandler{handler: h.handler.WithGroup(name)}
}

func toOTelValue(v slog.Value) log.Value {
	switch v.Kind() {
	case slog.KindString:
		return log.StringValue(v.String())
	case slog.KindInt64:
		return log.Int64Value(v.Int64())
	case slog.KindBool:
		return log.BoolValue(v.Bool())
	case slog.KindFloat64:
		return log.Float64Value(v.Float64())
	default:
		return log.StringValue(v.String())
	}
}
