package worker

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/socialchef/remy/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OTelMiddleware wraps asynq job handlers with OpenTelemetry spans.
func OTelMiddleware(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		tracer := telemetry.Tracer("worker")

		// Extract asynq specific info from context
		taskID, _ := asynq.GetTaskID(ctx)
		queueName, _ := asynq.GetQueueName(ctx)
		retryCount, _ := asynq.GetRetryCount(ctx)

		// Create span
		ctx, span := tracer.Start(ctx, fmt.Sprintf("job:%s", t.Type()), trace.WithSpanKind(trace.SpanKindConsumer))
		defer span.End()

		span.SetAttributes(
			attribute.String("job.id", taskID),
			attribute.String("job.type", t.Type()),
			attribute.String("job.queue", queueName),
			attribute.Int("job.retry_count", retryCount),
		)

		err := h.ProcessTask(ctx, t)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	})
}
