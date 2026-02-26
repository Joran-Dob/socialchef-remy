package worker

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("socialchef/worker")
)

type WorkerMetrics struct {
	jobCounter  metric.Int64Counter
	jobDuration metric.Float64Histogram
}

func NewWorkerMetrics() (*WorkerMetrics, error) {
	jobCounter, err := meter.Int64Counter(
		"worker.jobs.total",
		metric.WithDescription("Total number of worker jobs processed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	jobDuration, err := meter.Float64Histogram(
		"worker.job.duration",
		metric.WithDescription("Duration of worker jobs"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 30, 60, 120),
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return &WorkerMetrics{
		jobCounter:  jobCounter,
		jobDuration: jobDuration,
	}, nil
}

func (m *WorkerMetrics) RecordJob(ctx context.Context, jobType, status string, duration float64) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("job.type", jobType),
	}

	m.jobCounter.Add(ctx, 1, metric.WithAttributes(append(attrs, attribute.String("status", status))...))
	m.jobDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
}
