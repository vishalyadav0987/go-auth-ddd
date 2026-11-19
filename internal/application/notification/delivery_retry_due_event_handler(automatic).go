package notification

import (
	"context"
	"encoding/json"
	"fmt"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// DeliveryRetryDueProcessor processes an automatically scheduled retry.
//
// ProcessDeliveryRetryDueUseCase will implement this interface.
type DeliveryRetryDueProcessor interface {
	Execute(
		ctx context.Context,
		event domain.DeliveryRetryDueEvent,
	) error
}

var _ DeliveryRetryDueProcessor = (*ProcessDeliveryRetryDueUseCase)(nil)

// DeliveryRetryDueEventHandler handles DELIVERY_RETRY_DUE events.
//
// It decodes and validates the event, then delegates processing
// to the application use case.
type DeliveryRetryDueEventHandler struct {
	retryProcessor DeliveryRetryDueProcessor
}

var _ OutboxEventHandler = (*DeliveryRetryDueEventHandler)(nil)

// NewDeliveryRetryDueEventHandler creates the handler.
func NewDeliveryRetryDueEventHandler(
	retryProcessor DeliveryRetryDueProcessor,
) *DeliveryRetryDueEventHandler {
	return &DeliveryRetryDueEventHandler{
		retryProcessor: retryProcessor,
	}
}

// Handle processes an automatically scheduled retry event.
func (h *DeliveryRetryDueEventHandler) Handle(
	ctx context.Context,
	event *domain.NotificationOutboxEvent,
) error {

	if h.retryProcessor == nil {
		return fmt.Errorf("delivery retry due processor is not configured")
	}

	if event == nil {
		return fmt.Errorf("outbox event cannot be nil")
	}

	// Ensure this handler receives the correct event type.
	if event.EventType != domain.OutboxEventDeliveryRetryDue {
		return fmt.Errorf(
			"delivery retry due handler received unsupported event type %s (event %s)",
			event.EventType,
			event.ID,
		)
	}

	if len(event.Payload) == 0 {
		return fmt.Errorf(
			"delivery retry due event %s has an empty payload",
			event.ID,
		)
	}

	// Decode the automatic retry event.
	var retryEvent domain.DeliveryRetryDueEvent

	if err := json.Unmarshal(event.Payload, &retryEvent); err != nil {
		return fmt.Errorf(
			"decode delivery retry due event %s: %w",
			event.ID,
			err,
		)
	}

	// Validate required identifiers.
	if retryEvent.DeliveryID == "" {
		return fmt.Errorf(
			"delivery retry due event %s has no delivery ID",
			event.ID,
		)
	}

	if retryEvent.NotificationID == "" {
		return fmt.Errorf(
			"delivery retry due event %s has no notification ID",
			event.ID,
		)
	}

	if retryEvent.RetryCycle < 0 {
		return fmt.Errorf(
			"delivery retry due event %s has invalid retry cycle %d",
			event.ID,
			retryEvent.RetryCycle,
		)
	}

	if retryEvent.AttemptCount <= 0 {
		return fmt.Errorf(
			"delivery retry due event %s has invalid attempt count %d",
			event.ID,
			retryEvent.AttemptCount,
		)
	}

	if retryEvent.ScheduledAt.IsZero() {
		return fmt.Errorf(
			"delivery retry due event %s has no scheduled time",
			event.ID,
		)
	}

	// Delegate actual delivery validation and queue publishing.
	if err := h.retryProcessor.Execute(ctx, retryEvent); err != nil {
		return fmt.Errorf(
			"process automatic retry for delivery %s: %w",
			retryEvent.DeliveryID,
			err,
		)
	}

	return nil
}
