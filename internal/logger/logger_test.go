package logger

import (
	"context"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestNew(t *testing.T) {
	t.Run("production", func(t *testing.T) {
		l := New("production")
		if l == nil {
			t.Fatal("expected logger to be non-nil")
		}
	})

	t.Run("development", func(t *testing.T) {
		l := New("development")
		if l == nil {
			t.Fatal("expected logger to be non-nil")
		}
	})
}

type mockSpan struct {
	trace.Span
	sc trace.SpanContext
}

func (s mockSpan) SpanContext() trace.SpanContext {
	return s.sc
}

func TestWithTraceContext(t *testing.T) {
	t.Run("valid span", func(t *testing.T) {
		traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
		spanID, _ := trace.SpanIDFromHex("0102030405060708")
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: traceID,
			SpanID:  spanID,
		})
		ctx := trace.ContextWithSpan(context.Background(), mockSpan{sc: sc})

		attr := WithTraceContext(ctx)
		if attr.Key != "trace" {
			t.Errorf("expected key 'trace', got %s", attr.Key)
		}

		group := attr.Value.Group()
		if len(group) != 2 {
			t.Errorf("expected 2 attributes in group, got %d", len(group))
		}

		foundTraceID := false
		foundSpanID := false
		for _, a := range group {
			if a.Key == "trace_id" && a.Value.String() == "0102030405060708090a0b0c0d0e0f10" {
				foundTraceID = true
			}
			if a.Key == "span_id" && a.Value.String() == "0102030405060708" {
				foundSpanID = true
			}
		}

		if !foundTraceID {
			t.Error("trace_id not found or incorrect")
		}
		if !foundSpanID {
			t.Error("span_id not found or incorrect")
		}
	})

	t.Run("invalid span", func(t *testing.T) {
		ctx := context.Background()
		attr := WithTraceContext(ctx)
		if !attr.Equal(slog.Attr{}) {
			t.Errorf("expected empty attribute for invalid span, got %+v", attr)
		}
	})
}
