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

	AttemptCount int
	NextRetryAt  *time.Time
	SentAt       *time.Time
	DeliveredAt  *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
