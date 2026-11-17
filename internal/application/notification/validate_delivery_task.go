package notification

import (
	"fmt"

	"github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// DeliveryTask identifies one queued attempt to process a delivery.
//
// It is the application-level input for ProcessDeliveryUseCase. The queue
// adapter maps its transport payload onto this type, so the application
// layer does not depend on Asynq or on the queue package.
type DeliveryTask struct {
	DeliveryID     string
	NotificationID string

	// RetryCycle is the delivery's retry cycle when the task was
	// published. A task from an older cycle is stale.
	RetryCycle int
}

// Validate checks the task's own required fields.
func (t DeliveryTask) Validate() error {
	if t.DeliveryID == "" {
		return fmt.Errorf("delivery ID is required")
	}

	if t.NotificationID == "" {
		return fmt.Errorf("notification ID is required")
	}

	if t.RetryCycle < 0 {
		return fmt.Errorf("retry cycle cannot be negative")
	}

	return nil
}

// ValidateDeliveryTask verifies that a task still represents the current
// retry cycle of the stored delivery.
//
// It returns notification.ErrStaleDeliveryTask when the task belongs to an
// older cycle. It does not claim the delivery: ClaimDelivery re-checks the
// retry cycle atomically, closing the race with a concurrent DLQ retry.
func ValidateDeliveryTask(
	delivery *notification.NotificationDelivery,
	task DeliveryTask,
) error {

	if err := task.Validate(); err != nil {
		return err
	}

	if delivery == nil {
		return notification.ErrDeliveryNotFound
	}

	if delivery.ID != task.DeliveryID {
		return fmt.Errorf("delivery ID mismatch")
	}

	if delivery.NotificationID != task.NotificationID {
		return fmt.Errorf("notification ID mismatch")
	}

	if delivery.RetryCycle != task.RetryCycle {
		return fmt.Errorf(
			"%w: task cycle=%d, current cycle=%d",
			notification.ErrStaleDeliveryTask,
			task.RetryCycle,
			delivery.RetryCycle,
		)
	}

	return nil
}
