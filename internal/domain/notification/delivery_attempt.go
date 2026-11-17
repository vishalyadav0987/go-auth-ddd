package notification

import "time"

type NotificationDeliveryAttempt struct {
	ID         string
	DeliveryID string

	// AttemptNumber is the lifetime attempt number for this delivery.
	// It never resets when an administrator starts another retry cycle.
	AttemptNumber int
	// RetryCycle identifies which retry cycle this attempt belongs to.
	// Cycle 0 is the original delivery.
	RetryCycle   int
	Status       string
	ProviderID   string
	ErrorMessage string
	AttemptedAt  time.Time
}
