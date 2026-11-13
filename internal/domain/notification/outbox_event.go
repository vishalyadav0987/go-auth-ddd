package notification

import "time"

type OutboxEventStatus string

const (
	OutboxPending    OutboxEventStatus = "PENDING"
	OutboxProcessing OutboxEventStatus = "PROCESSING"
	OutboxProcessed  OutboxEventStatus = "PROCESSED"
	OutboxFailed     OutboxEventStatus = "FAILED"
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
