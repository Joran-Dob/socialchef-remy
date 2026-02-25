package telemetry

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTelemetry initializes OpenTelemetry with OTLP exporter
// Returns shutdown function and error
func InitTelemetry(ctx context.Context, serviceName, serviceVersion, env, otlpEndpoint string, headers map[string]string) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
			semconv.DeploymentEnvironmentKey.String(env),
		),
	)
	if err != nil {
		return nil, err
	}

	endpoint := otlpEndpoint
	insecure := false
	basePath := ""

	if endpoint != "" {
		if strings.HasPrefix(endpoint, "https://") {
			endpoint = strings.TrimPrefix(endpoint, "https://")
		} else if strings.HasPrefix(endpoint, "http://") {
			endpoint = strings.TrimPrefix(endpoint, "http://")
		insecure = true
		}

		if idx := strings.Index(endpoint, "/"); idx > 0 {
			basePath = endpoint[idx:]
			endpoint = endpoint[:idx]
		}
	}

	traceUrlPath := "/v1/traces"
	logUrlPath := "/" // Better Stack accepts logs at root path

	if basePath == "/otlp" {
		traceUrlPath = "/otlp/v1/traces"
		logUrlPath = "/otlp/v1/logs"
	} else if basePath != "" {
		basePath = strings.TrimSuffix(basePath, "/v1/traces")
		basePath = strings.TrimSuffix(basePath, "/v1/logs")
		basePath = strings.TrimSuffix(basePath, "/")
		traceUrlPath = basePath + "/v1/traces"
		logUrlPath = basePath + "/v1/logs"
	}

	traceOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithURLPath(traceUrlPath),
	}
	logOpts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithURLPath(logUrlPath),
	}  // Gzip compression may be needed - testing without first
	if len(headers) > 0 {
		traceOpts = append(traceOpts, otlptracehttp.WithHeaders(headers))
		logOpts = append(logOpts, otlploghttp.WithHeaders(headers))
	}

	if insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
		logOpts = append(logOpts, otlploghttp.WithInsecure())
	}

	traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, err
	}

	logExporter, err := otlploghttp.New(ctx, logOpts...)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)

	// Debug logging
	slog.Info("Telemetry initialized",
		"endpoint", endpoint,
		"trace_path", traceUrlPath,
		"log_path", logUrlPath,
		"insecure", insecure,
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		err1 := tp.Shutdown(ctx)
		err2 := lp.Shutdown(ctx)
		if err1 != nil {
			return err1
		}
		return err2
	}, nil
}

// Tracer returns a tracer with the given name
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// Middleware returns a chi middleware for HTTP tracing
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "http.request")
	}
}
