package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// DefaultTransport is the base transport used by the instrumented client.
var DefaultTransport = http.DefaultTransport

type contextKey string

const providerKey contextKey = "httpclient.provider"

// WithProvider adds a provider name to the context for tracing.
func WithProvider(ctx context.Context, provider string) context.Context {
	return context.WithValue(ctx, providerKey, provider)
}

// providerTransport is a RoundTripper that adds provider attributes to the current span.
type providerTransport struct {
	base http.RoundTripper
}

func (t *providerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	span := trace.SpanFromContext(req.Context())
	if provider, ok := req.Context().Value(providerKey).(string); ok {
		span.SetAttributes(attribute.String("provider", provider))
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if resp.StatusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP status %d", resp.StatusCode))
	}
	return resp, nil
}

func newOtelTransport(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(&providerTransport{base: base},
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			provider, _ := r.Context().Value(providerKey).(string)
			if provider != "" {
				return fmt.Sprintf("%s: %s %s", provider, r.Method, r.URL.Path)
			}
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}),
	)
}

// InstrumentedClient is an http.Client with OpenTelemetry instrumentation.
var InstrumentedClient = &http.Client{
	Transport: newOtelTransport(DefaultTransport),
	Timeout:   180 * time.Second, // Max timeout used in scrapers
}

// NewInstrumentedClient returns a new http.Client with OpenTelemetry instrumentation and custom timeout.
func NewInstrumentedClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: newOtelTransport(DefaultTransport),
		Timeout:   timeout,
	}
}

// WrapClient wraps an existing http.Client's transport with OpenTelemetry instrumentation.
func WrapClient(client *http.Client) *http.Client {
	if client.Transport == nil {
		client.Transport = DefaultTransport
	}
	client.Transport = newOtelTransport(client.Transport)
	return client
}
