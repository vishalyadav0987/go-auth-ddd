package notificationQueue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	appnotification "github.com/vishalyadav0987/authentication/internal/application/notification"
	"github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/pkg/notiflog"
)

// DeliveryHandler is the Asynq entry point for email and push delivery
// tasks. It decodes the transport payload, maps it to an application
// DeliveryTask and runs the delivery use case.
type DeliveryHandler struct {
	processor DeliveryProcessor
}

// NewDeliveryHandler creates the Asynq delivery handler.
func NewDeliveryHandler(processor DeliveryProcessor) *DeliveryHandler {
	return &DeliveryHandler{processor: processor}
}

// Handle processes one Asynq notification delivery task.
//
// Return values decide what Asynq does next:
//   - nil: task acknowledged (success, or a stale task that must not run).
//   - wraps asynq.SkipRetry: task is malformed and can never succeed, so
//     it is archived immediately instead of being retried.
//   - any other error: database, provider, context or other transient
//     failures are returned unchanged so Asynq's retry policy applies.
func (h *DeliveryHandler) Handle(
	ctx context.Context,
	task *asynq.Task,
) error {

	if task == nil {
		return fmt.Errorf("delivery task cannot be nil")
	}

	if h.processor == nil {
		return fmt.Errorf("delivery processor is not configured")
	}

	start := time.Now()

	// Asynq-level retry info. These retries happen only when Execute
	// returns an error (DB/infra failures); provider failures are retried
	// by the delivery's own RETRYING schedule instead.
	taskID, _ := asynq.GetTaskID(ctx)
	asynqRetry, _ := asynq.GetRetryCount(ctx)
	asynqMaxRetry, _ := asynq.GetMaxRetry(ctx)
	queueName, _ := asynq.GetQueueName(ctx)

	lg := notiflog.L().With(
		notiflog.KeyTaskID, taskID,
		"task_type", task.Type(),
		notiflog.KeyQueue, queueName,
		"asynq_retry", asynqRetry,
		"asynq_max_retry", asynqMaxRetry,
	)

	// Decode and validate the payload's required fields.
	payload, err := UnmarshalDeliveryPayload(task.Payload())
	if err != nil {
		lg.Error("task.payload_invalid", "action", "archived (SkipRetry)", notiflog.Err(err))
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	deliveryTask := appnotification.DeliveryTask{
		DeliveryID:     payload.DeliveryID,
		NotificationID: payload.NotificationID,
		RetryCycle:     payload.RetryCycle,
	}

	lg = lg.With(
		notiflog.KeyDeliveryID, deliveryTask.DeliveryID,
		notiflog.KeyNotificationID, deliveryTask.NotificationID,
		notiflog.KeyRetryCycle, deliveryTask.RetryCycle,
	)

	if err := deliveryTask.Validate(); err != nil {
		lg.Error("task.payload_invalid", "action", "archived (SkipRetry)", notiflog.Err(err))
		return fmt.Errorf("%w: invalid delivery task: %v", asynq.SkipRetry, err)
	}

	lg.Info("task.received")

	// Pass the task context fields down to the use case's logs.
	ctx = notiflog.With(ctx,
		notiflog.KeyTaskID, taskID,
		"asynq_retry", asynqRetry,
	)

	if err := h.processor.Execute(ctx, deliveryTask); err != nil {

		// A task from an older retry cycle is obsolete, not a failure.
		// Acknowledge it so Asynq does not retry it.
		if errors.Is(err, notification.ErrStaleDeliveryTask) {
			lg.Info("task.acknowledged_stale", notiflog.Since(start))
			return nil
		}

		action := "asynq will retry"
		if asynqRetry >= asynqMaxRetry {
			action = "asynq retries exhausted; task will be archived"
		}

		lg.Error("task.failed", "action", action, notiflog.Err(err), notiflog.Since(start))

		return fmt.Errorf(
			"process delivery task %s (cycle %d): %w",
			deliveryTask.DeliveryID,
			deliveryTask.RetryCycle,
			err,
		)
	}

	lg.Info("task.completed", notiflog.Since(start))

	return nil
}
