package notification

import "time"

type Notification struct {
	ID          string
	Type        NotificationType
	Title       string
	Body        string
	HTMLBody    string
	Target      NotificationTarget
	Channels    []NotificationChannel
	Priority    NotificationPriority
	Metadata    map[string]interface{}
	ScheduledAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
