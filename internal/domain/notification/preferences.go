package notification

import "time"

type NotificationPreference struct {
	ID     string
	UserID string
	Type   NotificationType

	EmailEnabled bool
	PushEnabled  bool

	CreatedAt time.Time
	UpdatedAt time.Time
}
