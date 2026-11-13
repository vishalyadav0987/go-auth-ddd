package notification

import "context"

type PreferenceRepository interface {
	GetPreferencesByUserID(
		ctx context.Context,
		userID string,
	) ([]*NotificationPreference, error)

	GetPreference(
		ctx context.Context,
		userID string,
		notificationType NotificationType,
	) (*NotificationPreference, error)

	SavePreference(
		ctx context.Context,
		preference *NotificationPreference,
	) error

	GetEligibleUsers(
		ctx context.Context,
		notificationType NotificationType,
		channel NotificationChannel,
		userIDs []string,
	) ([]string, error)
}
