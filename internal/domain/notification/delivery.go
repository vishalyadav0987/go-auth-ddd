package notification

import "time"

type NotificationDelivery struct {
	ID             string
	NotificationID string
	UserID         string
	Channel        NotificationChannel
	Status         DeliveryStatus

	Recipient     string
	ProviderID    string
	FailureReason string

	AttemptCount   int
	NextRetryAt    *time.Time
	SentAt         *time.Time
	DeliveredAt    *time.Time
	DeadLetteredAt *time.Time

	// RetryCycle identifies the current retry cycle.
	// Cycle 0 is the original delivery.
	// Every administrator-initiated retry increments this value.
	RetryCycle int

	CreatedAt time.Time
	UpdatedAt time.Time
}
