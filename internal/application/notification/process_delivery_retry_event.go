// This use case will:
// - Load the delivery.
// - Verify that it is still PENDING.
// - Verify that the event's retry cycle matches the delivery's current retry cycle.
// - Load the notification to determine its priority.
// - Publish the delivery using the existing queue publisher.

package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// DeliveryTaskPublisher publishes a delivery task to the appropriate
// asynchronous queue.
//
// This interface keeps the application layer independent of Asynq.
type DeliveryTaskPublisher interface {
	PublishDelivery(
		ctx context.Context,
		delivery *notification.NotificationDelivery,
		priority notification.NotificationPriority,
	) error
}

// ProcessDeliveryRetryEventUseCase processes an administrator-requested
// delivery retry event from the transactional outbox.
type ProcessDeliveryRetryEventUseCase struct {
	notificationRepository notification.NotificationRepository
	publisher              DeliveryTaskPublisher
}

// NewProcessDeliveryRetryEventUseCase creates the use case.
func NewProcessDeliveryRetryEventUseCase(
	notificationRepository notification.NotificationRepository,
	publisher DeliveryTaskPublisher,
) *ProcessDeliveryRetryEventUseCase {
	return &ProcessDeliveryRetryEventUseCase{
		notificationRepository: notificationRepository,
		publisher:              publisher,
	}
}

// Execute validates the retry event and publishes its delivery task.
//
// It does not send the email or push notification itself.
// The existing delivery worker handles the actual provider call.
func (uc *ProcessDeliveryRetryEventUseCase) Execute(
	ctx context.Context,
	event notification.DeliveryRetryEvent,
) error {

	// Step 1: Validate the event.
	if event.DeliveryID == "" {
		return errors.New("delivery retry event has an empty delivery ID")
	}

	if event.NotificationID == "" {
		return errors.New("delivery retry event has an empty notification ID")
	}

	if uc.notificationRepository == nil {
		return errors.New("notification repository is not configured")
	}

	if uc.publisher == nil {
		return errors.New("delivery task publisher is not configured")
	}

	// Step 2: Load the delivery from PostgreSQL.
	delivery, err := uc.notificationRepository.GetDeliveryByID(
		ctx,
		event.DeliveryID,
	)
	if err != nil {
		return fmt.Errorf("get retry delivery: %w", err)
	}

	if delivery == nil {
		return notification.ErrDeliveryNotFound
	}

	// Step 3: Check whether this event is still valid.
	//
	// An administrator may retry a delivery more than once over time.
	// An old outbox event must not publish a delivery belonging to a
	// newer retry cycle.
	//
	// The status check also prevents publishing a delivery that has
	// already been cancelled, accepted, or otherwise moved forward.
	if delivery.Status != notification.DeliveryPending {
		return nil
	}

	if delivery.RetryCycle != event.RetryCycle {
		return nil
	}

	// Step 4: Verify that the event references the correct notification.
	if delivery.NotificationID != event.NotificationID {
		return errors.New("retry event notification ID does not match delivery")
	}

	// Step 5: Load the original notification.
	originalNotification, err := uc.notificationRepository.GetNotificationByID(
		ctx,
		delivery.NotificationID,
	)
	if err != nil {
		return fmt.Errorf("get notification for retry: %w", err)
	}

	if originalNotification == nil {
		return errors.New("notification associated with retry delivery was not found")
	}

	// Step 6: Publish the delivery using its original notification priority.
	//
	// The publisher maps HIGH, NORMAL and LOW to their configured
	// Asynq queues.
	if err := uc.publisher.PublishDelivery(
		ctx,
		delivery,
		originalNotification.Priority,
	); err != nil {
		return fmt.Errorf("publish retry delivery task: %w", err)
	}

	return nil
}
