package notification

import "context"

type PreferenceRepository interface {
	// GetPreferencesByUserID returns only the preferences the user has
	// stored; types without a row use DefaultPreference.
	GetPreferencesByUserID(
		ctx context.Context,
		userID string,
	) ([]*NotificationPreference, error)

	// SavePreferences upserts all preferences in one transaction.
	SavePreferences(
		ctx context.Context,
		preferences []*NotificationPreference,
	) error

	// GetEligibleUsers returns the users who may receive notificationType
	// on channel: preference allows it (missing row = allowed, mandatory
	// types always allowed) and the user is reachable on that channel
	// (has an email / has an active device).
	//
	// userIDs limits the candidates; nil means all users.
	GetEligibleUsers(
		ctx context.Context,
		notificationType NotificationType,
		channel NotificationChannel,
		userIDs []string,
	) ([]string, error)
}
