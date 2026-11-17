package notificationQueue

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/hibiken/asynq"

	appnotification "github.com/vishalyadav0987/authentication/internal/application/notification"
	"github.com/vishalyadav0987/authentication/internal/domain/notification"
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

	// Decode and validate the payload's required fields.
	payload, err := UnmarshalDeliveryPayload(task.Payload())
	if err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	deliveryTask := appnotification.DeliveryTask{
		DeliveryID:     payload.DeliveryID,
		NotificationID: payload.NotificationID,
		RetryCycle:     payload.RetryCycle,
	}

	if err := deliveryTask.Validate(); err != nil {
		return fmt.Errorf("%w: invalid delivery task: %v", asynq.SkipRetry, err)
	}

	if err := h.processor.Execute(ctx, deliveryTask); err != nil {

		// A task from an older retry cycle is obsolete, not a failure.
		// Acknowledge it so Asynq does not retry it.
		if errors.Is(err, notification.ErrStaleDeliveryTask) {
			log.Printf(
				"Ignoring stale delivery task: delivery_id=%s retry_cycle=%d",
				deliveryTask.DeliveryID,
				deliveryTask.RetryCycle,
			)

			return nil
		}

		return fmt.Errorf(
			"process delivery task %s (cycle %d): %w",
			deliveryTask.DeliveryID,
			deliveryTask.RetryCycle,
			err,
		)
	}

	return nil
}
