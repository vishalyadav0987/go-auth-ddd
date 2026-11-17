package notification

// NotificationCreatedEvent is the NOTIFICATION_CREATED outbox payload.
//
// It carries the notification ID only. The handler loads the PENDING
// deliveries from the database, so recipients' emails and FCM tokens are
// never copied into the outbox, and broadcast payloads stay small.
type NotificationCreatedEvent struct {
	NotificationID string `json:"notification_id"`
}
