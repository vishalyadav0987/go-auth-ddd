package notification

import "time"

// DeliveryRetryEvent represents a request to process a delivery again.
//
// It contains identifiers only. Sensitive notification content,
// email addresses, and FCM tokens are not placed in the outbox payload.
type DeliveryRetryEvent struct {
	DeliveryID     string    `json:"delivery_id"`
	NotificationID string    `json:"notification_id"`
	RetryCycle     int       `json:"retry_cycle"`
	RequestedBy    string    `json:"requested_by"`
	RequestedAt    time.Time `json:"requested_at"`
}

// We'll design and implement an outbox event for manual DLQ retries, so resetting the delivery and scheduling its next processing cycle are reliable even if the application crashes.

// If the application crashes after this transaction commits but before the queue task is published, the delivery could remain pending without being scheduled.

// -------------------------------------------------------

// DeliveryRetryDueEvent represents an automatically scheduled retry.
//
// It contains identifiers and retry metadata only.
// Sensitive notification content, email addresses, and FCM tokens
// must never be included in this event.
type DeliveryRetryDueEvent struct {
	DeliveryID     string    `json:"delivery_id"`
	NotificationID string    `json:"notification_id"`
	RetryCycle     int       `json:"retry_cycle"`
	AttemptCount   int       `json:"attempt_count"`
	ScheduledAt    time.Time `json:"scheduled_at"`
}
