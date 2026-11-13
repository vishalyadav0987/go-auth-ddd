package notification

import "context"

type NotificationOutboxRepository interface {
	CreateEvent(
		ctx context.Context,
		event *NotificationOutboxEvent,
	) error

	GetPendingEvents(
		ctx context.Context,
		limit int,
	) ([]*NotificationOutboxEvent, error)

	MarkProcessing(
		ctx context.Context,
		eventID string,
	) error

	MarkProcessed(
		ctx context.Context,
		eventID string,
	) error

	MarkFailed(
		ctx context.Context,
		eventID string,
		reason string,
	) error
}
