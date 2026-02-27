package worker

import (
	"context"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/hibiken/asynq"
)

// SentryMiddleware wraps asynq job handlers with Sentry error capture.
func SentryMiddleware(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		taskID, _ := asynq.GetTaskID(ctx)
		queueName, _ := asynq.GetQueueName(ctx)
		retryCount, _ := asynq.GetRetryCount(ctx)

		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetTag("task_type", t.Type())
		hub.Scope().SetTag("task_id", taskID)
		hub.Scope().SetTag("queue", queueName)
		hub.Scope().SetTag("retry_count", strconv.Itoa(retryCount))

		ctx = sentry.SetHubOnContext(ctx, hub)

		err := h.ProcessTask(ctx, t)
		if err != nil {
			hub.CaptureException(err)
		}

		return err
	})
}
