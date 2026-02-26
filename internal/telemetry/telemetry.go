package telemetry

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
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
		}
		if idx := strings.Index(endpoint, "/"); idx > 0 {
			basePath = endpoint[idx:]
			endpoint = endpoint[:idx]
		}
	}

	traceUrlPath := "/v1/traces"
	logUrlPath := "/v1/logs" // Better Stack uses standard OTLP path
	metricUrlPath := "/v1/metrics"

	if basePath == "/otlp" {
		traceUrlPath = "/otlp/v1/traces"
		logUrlPath = "/otlp/v1/logs"
		metricUrlPath = "/otlp/v1/metrics"
	} else if basePath != "" {
		basePath = strings.TrimSuffix(basePath, "/v1/traces")
		basePath = strings.TrimSuffix(basePath, "/v1/logs")
		basePath = strings.TrimSuffix(basePath, "/v1/metrics")
		basePath = strings.TrimSuffix(basePath, "/")
		traceUrlPath = basePath + "/v1/traces"
		logUrlPath = basePath + "/v1/logs"
		metricUrlPath = basePath + "/v1/metrics"
	}

	traceOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithURLPath(traceUrlPath),
	}
	logOpts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithURLPath(logUrlPath),
	}
	metricOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithURLPath(metricUrlPath),
	}
	if len(headers) > 0 {
		traceOpts = append(traceOpts, otlptracehttp.WithHeaders(headers))
		logOpts = append(logOpts, otlploghttp.WithHeaders(headers))
		metricOpts = append(metricOpts, otlpmetrichttp.WithHeaders(headers))
	}

	if insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
		logOpts = append(logOpts, otlploghttp.WithInsecure())
		metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
	}

	traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, err
	}

	logExporter, err := otlploghttp.New(ctx, logOpts...)
	if err != nil {
		return nil, err
	}

	metricExporter, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter, sdklog.WithExportTimeout(5*time.Second))),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)

	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(60*time.Second))),
	)
	otel.SetMeterProvider(mp)

	if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		err1 := tp.Shutdown(ctx)
		err2 := lp.Shutdown(ctx)
		err3 := mp.Shutdown(ctx)
		if err1 != nil {
			return err1
		}
		if err2 != nil {
			return err2
		}
		return err3
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
