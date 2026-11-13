package notification

import "context"

type NotificationRepository interface {
	CreateNotification(
		ctx context.Context,
		notification *Notification,
	) error

	GetNotificationByID(
		ctx context.Context,
		id string,
	) (*Notification, error)

	CreateDeliveries(
		ctx context.Context,
		deliveries []*NotificationDelivery,
	) error

	GetDeliveryByID(
		ctx context.Context,
		id string,
	) (*NotificationDelivery, error)

	UpdateDeliveryStatus(
		ctx context.Context,
		deliveryID string,
		status DeliveryStatus,
	) error

	UpdateDeliveryProviderID(
		ctx context.Context,
		deliveryID string,
		providerID string,
	) error

	IncrementDeliveryAttempt(
		ctx context.Context,
		deliveryID string,
	) error

	GetPendingDeliveries(
		ctx context.Context,
		limit int,
	) ([]*NotificationDelivery, error)

	UpdateDeliveryRetry(
		ctx context.Context,
		deliveryID string,
		status DeliveryStatus,
		nextRetryAt int64,
		reason string,
	) error
}
