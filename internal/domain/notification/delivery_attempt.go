package notification

import "time"

type NotificationDeliveryAttempt struct {
	ID            string
	DeliveryID    string
	AttemptNumber int
	Status        string
	ProviderID    string
	ErrorMessage  string
	AttemptedAt   time.Time
}
