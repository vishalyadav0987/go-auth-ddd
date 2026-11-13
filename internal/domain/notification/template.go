package notification

import "time"

type NotificationTemplate struct {
	ID        string
	Code      string
	Type      NotificationType
	Channel   NotificationChannel
	Subject   string
	Title     string
	Body      string
	HTMLBody  string
	Version   int
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
