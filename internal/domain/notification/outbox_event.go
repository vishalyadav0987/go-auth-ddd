package notification

import "time"

type OutboxEventStatus string

const (
	OutboxPending    OutboxEventStatus = "PENDING"
	OutboxProcessing OutboxEventStatus = "PROCESSING"
	OutboxProcessed  OutboxEventStatus = "PROCESSED"
	OutboxFailed     OutboxEventStatus = "FAILED"
)

// Outbox event types used by the notification system.
const (
	OutboxEventNotificationCreated = "NOTIFICATION_CREATED"

	// OutboxEventDeliveryRetryRequested represents an administrator's
	// request to retry a previously dead-lettered delivery.
	OutboxEventDeliveryRetryRequested = "DELIVERY_RETRY_REQUESTED"
)

type NotificationOutboxEvent struct {
	ID          string
	EventType   string
	AggregateID string
	Payload     []byte

	Status      OutboxEventStatus
	Attempts    int
	NextRetryAt *time.Time

	CreatedAt   time.Time
	ProcessedAt *time.Time
	UpdatedAt   time.Time
}
