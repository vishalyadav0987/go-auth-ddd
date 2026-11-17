package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/pkg/notiflog"
)

// NotificationCreatedEventHandler handles NOTIFICATION_CREATED outbox
// events by publishing every PENDING delivery of the notification to the
// delivery queue.
//
// Re-running it is safe: task IDs are deterministic per delivery and retry
// cycle, so already-queued deliveries are not duplicated, and deliveries a
// worker has already claimed are no longer PENDING.
type NotificationCreatedEventHandler struct {
	notificationRepository notification.NotificationRepository
	publisher              DeliveryTaskPublisher
}

var _ OutboxEventHandler = (*NotificationCreatedEventHandler)(nil)

func NewNotificationCreatedEventHandler(
	notificationRepository notification.NotificationRepository,
	publisher DeliveryTaskPublisher,
) *NotificationCreatedEventHandler {
	return &NotificationCreatedEventHandler{
		notificationRepository: notificationRepository,
		publisher:              publisher,
	}
}

func (h *NotificationCreatedEventHandler) Handle(
	ctx context.Context,
	event *notification.NotificationOutboxEvent,
) error {

	if h.notificationRepository == nil || h.publisher == nil {
		return errors.New("notification created handler is not configured")
	}

	if event == nil || event.EventType != notification.OutboxEventNotificationCreated {
		return fmt.Errorf("notification created handler received an unsupported event")
	}

	var payload notification.NotificationCreatedEvent
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode notification created event %s: %w", event.ID, err)
	}

	if payload.NotificationID == "" {
		return fmt.Errorf("notification created event %s has no notification ID", event.ID)
	}

	// The priority decides the queue.
	n, err := h.notificationRepository.GetNotificationByID(ctx, payload.NotificationID)
	if err != nil {
		return fmt.Errorf("load notification %s: %w", payload.NotificationID, err)
	}

	deliveries, err := h.notificationRepository.GetPendingDeliveriesByNotificationID(
		ctx, payload.NotificationID,
	)
	if err != nil {
		return fmt.Errorf("load deliveries of %s: %w", payload.NotificationID, err)
	}

	ctx = notiflog.With(ctx, notiflog.KeyNotificationID, payload.NotificationID)
	lg := notiflog.From(ctx)

	lg.Info("notification_created.publishing",
		"pending_deliveries", len(deliveries),
		"priority", n.Priority,
	)

	// Publish every delivery even if some fail; the outbox retries the
	// whole event and already-queued tasks are deduplicated.
	var publishErrs []error

	for _, delivery := range deliveries {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := h.publisher.PublishDelivery(ctx, delivery, n.Priority); err != nil {
			publishErrs = append(publishErrs, fmt.Errorf("publish delivery %s: %w", delivery.ID, err))
		}
	}

	if len(publishErrs) > 0 {
		lg.Error("notification_created.publish_incomplete",
			"failed", len(publishErrs),
			"published", len(deliveries)-len(publishErrs),
			notiflog.Err(errors.Join(publishErrs...)),
		)
		return errors.Join(publishErrs...)
	}

	lg.Info("notification_created.published", "published", len(deliveries))

	return nil
}
