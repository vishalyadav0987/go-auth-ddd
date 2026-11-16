package notification

import "context"

type DeliveryAttemptRepository interface {
	CreateAttempt(
		ctx context.Context,
		attempt *NotificationDeliveryAttempt,
	) error

	UpdateAttempt(
		ctx context.Context,
		deliveryID string,
		attemptNumber int,
		status string,
		providerID string,
		errorMessage string,
		processingToken string,
	) error

	// GetDeliveryAttemptsByDeliveryID retrieves the complete attempt
	// history for a delivery, ordered from the oldest attempt to the newest.
	GetDeliveryAttemptsByDeliveryID(
		ctx context.Context,
		deliveryID string,
	) ([]*NotificationDeliveryAttempt, error)
}
