package notification

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/pkg/notiflog"
)

// ProcessDeliveryRetryDueUseCase processes an automatically scheduled
// retry event from the notification outbox.
type ProcessDeliveryRetryDueUseCase struct {
	notificationRepository domain.NotificationRepository
	publisher              DeliveryTaskPublisher
}

// NewProcessDeliveryRetryDueUseCase creates the use case.
func NewProcessDeliveryRetryDueUseCase(
	notificationRepository domain.NotificationRepository,
	publisher DeliveryTaskPublisher,
) *ProcessDeliveryRetryDueUseCase {
	return &ProcessDeliveryRetryDueUseCase{
		notificationRepository: notificationRepository,
		publisher:              publisher,
	}
}

// Execute validates an automatic retry event and publishes its delivery.
//
// It does not send the email or push notification itself.
// The existing delivery worker handles the actual provider call.
func (uc *ProcessDeliveryRetryDueUseCase) Execute(
	ctx context.Context,
	event domain.DeliveryRetryDueEvent,
) error {

	// Step 1: Validate the event.
	if event.DeliveryID == "" {
		return errors.New("delivery retry due event has an empty delivery ID")
	}

	if event.NotificationID == "" {
		return errors.New("delivery retry due event has an empty notification ID")
	}

	if event.RetryCycle < 0 {
		return errors.New("delivery retry due event has an invalid retry cycle")
	}

	if event.AttemptCount <= 0 {
		return errors.New("delivery retry due event has an invalid attempt count")
	}

	if event.ScheduledAt.IsZero() {
		return errors.New("delivery retry due event has no scheduled time")
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
		return fmt.Errorf("get automatic retry delivery: %w", err)
	}

	if delivery == nil {
		return domain.ErrDeliveryNotFound
	}

	// Step 3: Prepare structured logging.
	lg := notiflog.From(ctx).With(
		notiflog.KeyDeliveryID, delivery.ID,
		notiflog.KeyNotificationID, delivery.NotificationID,
		notiflog.KeyRetryCycle, event.RetryCycle,
		"event_attempt_count", event.AttemptCount,
	)

	// Step 4: Check whether this event is still valid.
	//
	// The scheduler changes RETRYING to PENDING before creating
	// this event. Therefore, only PENDING deliveries are publishable.
	if delivery.Status != domain.DeliveryPending {
		lg.Info("automatic_retry.event_obsolete",
			"reason", "delivery is no longer PENDING",
			notiflog.KeyStatus, delivery.Status,
		)
		return nil
	}

	// Do not publish a delivery from an older retry generation.
	if delivery.RetryCycle != event.RetryCycle {
		lg.Info("automatic_retry.event_obsolete",
			"reason", "delivery moved to a newer retry cycle",
			"current_retry_cycle", delivery.RetryCycle,
		)
		return nil
	}

	// Do not publish if the delivery has advanced to another attempt.
	if delivery.AttemptCount != event.AttemptCount {
		lg.Info("automatic_retry.event_obsolete",
			"reason", "delivery attempt count has changed",
			"current_attempt_count", delivery.AttemptCount,
		)
		return nil
	}

	// Step 5: Verify the notification ID.
	if delivery.NotificationID != event.NotificationID {
		return errors.New(
			"retry event notification ID does not match delivery",
		)
	}

	// Step 6: Load the original notification.
	originalNotification, err := uc.notificationRepository.GetNotificationByID(
		ctx,
		delivery.NotificationID,
	)
	if err != nil {
		return fmt.Errorf("get notification for automatic retry: %w", err)
	}

	if originalNotification == nil {
		return errors.New(
			"notification associated with automatic retry was not found",
		)
	}

	// Step 7: Publish the delivery using its original priority.
	if err := uc.publisher.PublishDelivery(
		ctx,
		delivery,
		originalNotification.Priority,
	); err != nil {
		lg.Error("automatic_retry.publish_failed", notiflog.Err(err))

		return fmt.Errorf(
			"publish automatic retry delivery task: %w",
			err,
		)
	}

	lg.Info("automatic_retry.published")

	return nil
}

// Compile-time interface verification.
var _ DeliveryRetryDueProcessor = (*ProcessDeliveryRetryDueUseCase)(nil)
