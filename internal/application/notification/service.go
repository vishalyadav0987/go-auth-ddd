package notification

import (
	"context"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

type NotificationService interface {
	SendNotification(
		ctx context.Context,
		notification *domain.Notification,
	) error

	RegisterDevice(
		ctx context.Context,
		device *domain.UserDevice,
	) error

	UpdatePreferences(
		ctx context.Context,
		preferences []*domain.NotificationPreference,
	) error

	RetryDelivery(
		ctx context.Context,
		deliveryID string,
	) error
}
