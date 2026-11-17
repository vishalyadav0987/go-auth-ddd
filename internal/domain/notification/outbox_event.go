package notification

import "time"

type OutboxEventStatus string

const (
	OutboxPending    OutboxEventStatus = "PENDING"
	OutboxProcessing OutboxEventStatus = "PROCESSING"
	OutboxProcessed  OutboxEventStatus = "PROCESSED"

	// OutboxFailed is terminal: the event exhausted its attempts
	// (or is not processable) and needs operator attention.
	OutboxFailed OutboxEventStatus = "FAILED"
)

// Outbox event types used by the notification system.
const (
	OutboxEventNotificationCreated = "NOTIFICATION_CREATED"

	// OutboxEventDeliveryRetryRequested represents an administrator's
	// request to retry a previously dead-lettered delivery.
	OutboxEventDeliveryRetryRequested = "DELIVERY_RETRY_REQUESTED"
)

// NotificationOutboxEvent is a row in notification_outbox_events.
type NotificationOutboxEvent struct {
	ID          string
	EventType   string
	AggregateID string
	Payload     []byte

	Status      OutboxEventStatus
	Attempts    int
	NextRetryAt *time.Time
	LastError   string

	// ProcessingToken is set when the event is claimed. It must be
	// passed back to MarkProcessed / MarkFailed / MarkDead to prove
	// the caller still owns the lease.
	ProcessingToken string

	CreatedAt   time.Time
	ProcessedAt *time.Time
	UpdatedAt   time.Time
}
