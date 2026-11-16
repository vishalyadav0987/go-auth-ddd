package notification

import "time"

// DeadLetterDelivery represents a notification delivery
// that exhausted its retry attempts and entered the DLQ.
//
// This model is intended for administrative inspection
// and manual recovery workflows.
type DeadLetterDelivery struct {

	// Unique delivery identifier.
	DeliveryID string

	// Parent notification identifier.
	NotificationID string

	// Notification category.
	NotificationType NotificationType

	// Notification title.
	Title string

	// Delivery channel: EMAIL or PUSH.
	Channel NotificationChannel

	// Destination of the notification.
	// Email address for EMAIL.
	// FCM registration token for PUSH.
	Recipient string

	// Final delivery status.
	// Expected value: FAILED.
	Status DeliveryStatus

	// Reason the notification failed.
	FailureReason string

	// Number of provider delivery attempts.
	AttemptCount int

	// Timestamp when the delivery entered DLQ.
	DeadLetteredAt time.Time

	// Original delivery creation timestamp.
	CreatedAt time.Time

	// Last delivery update timestamp.
	UpdatedAt time.Time
}
