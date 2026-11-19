package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// DeliveryRetryProcessor processes a notification delivery retry request.
//
// ProcessDeliveryRetryEventUseCase implements this interface.
type DeliveryRetryProcessor interface {
	Execute(
		ctx context.Context,
		event notification.DeliveryRetryEvent,
	) error
}

var _ DeliveryRetryProcessor = (*ProcessDeliveryRetryEventUseCase)(nil)

// DeliveryRetryEventHandler handles DELIVERY_RETRY_REQUESTED events from
// the notification outbox.
//
// It only decodes and validates the event, then delegates. It does not
// send email or push itself: ProcessDeliveryRetryEventUseCase re-publishes
// the delivery and the existing delivery worker performs the send.
type DeliveryRetryEventHandler struct {
	retryProcessor DeliveryRetryProcessor
}

var _ OutboxEventHandler = (*DeliveryRetryEventHandler)(nil)

// NewDeliveryRetryEventHandler creates the handler.
func NewDeliveryRetryEventHandler(
	retryProcessor DeliveryRetryProcessor,
) *DeliveryRetryEventHandler {
	return &DeliveryRetryEventHandler{
		retryProcessor: retryProcessor,
	}
}

// Handle decodes the retry event and delegates it to the retry processor.
//
// Errors are returned to ProcessOutboxEventUseCase, which reschedules the
// event or moves it to FAILED after MaxOutboxAttempts.
func (h *DeliveryRetryEventHandler) Handle(
	ctx context.Context,
	event *notification.NotificationOutboxEvent,
) error {

	if h.retryProcessor == nil {
		return fmt.Errorf("delivery retry processor is not configured")
	}

	if event == nil {
		return fmt.Errorf("outbox event cannot be nil")
	}

	// Guards against registering this handler under the wrong type.
	if event.EventType != notification.OutboxEventDeliveryRetryRequested {
		return fmt.Errorf(
			"delivery retry handler received unsupported event type %s (event %s)",
			event.EventType,
			event.ID,
		)
	}

	if len(event.Payload) == 0 {
		return fmt.Errorf("delivery retry event %s has an empty payload", event.ID)
	}

	// Decode the JSON payload written by PrepareDeadLetterRetry.
	var retryEvent notification.DeliveryRetryEvent

	if err := json.Unmarshal(event.Payload, &retryEvent); err != nil {
		return fmt.Errorf("decode delivery retry event %s: %w", event.ID, err)
	}

	// Validate required fields before doing any work.
	if retryEvent.DeliveryID == "" {
		return fmt.Errorf("delivery retry event %s has no delivery ID", event.ID)
	}

	if retryEvent.NotificationID == "" {
		return fmt.Errorf("delivery retry event %s has no notification ID", event.ID)
	}

	// Manual retries start at cycle 1; cycle 0 is the original delivery.
	if retryEvent.RetryCycle <= 0 {
		return fmt.Errorf(
			"delivery retry event %s has invalid retry cycle %d",
			event.ID,
			retryEvent.RetryCycle,
		)
	}

	if err := h.retryProcessor.Execute(ctx, retryEvent); err != nil {
		return fmt.Errorf(
			"process delivery retry for delivery %s: %w",
			retryEvent.DeliveryID,
			err,
		)
	}

	return nil
}
