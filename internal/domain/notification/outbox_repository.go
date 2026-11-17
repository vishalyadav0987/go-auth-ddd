package notification

import (
	"context"
	"errors"
	"time"
)

// ErrOutboxLeaseLost is returned when an outbox event can no longer be
// updated by the caller because its lease expired and another worker
// reclaimed it (or it was already finalised).
var ErrOutboxLeaseLost = errors.New("notification outbox event lease lost")

// NotificationOutboxRepository persists notification outbox events.
//
// It is backed by its own table and is independent of the order outbox.
type NotificationOutboxRepository interface {
	// Create stores a new PENDING event.
	//
	// Repositories that must write an event atomically with other
	// changes insert it inside their own transaction instead.
	Create(
		ctx context.Context,
		event *NotificationOutboxEvent,
	) error

	// ClaimPending atomically claims up to limit due events (PENDING with
	// next_retry_at <= now, or PROCESSING with an expired lease), moves
	// them to PROCESSING and assigns each a fresh ProcessingToken.
	ClaimPending(
		ctx context.Context,
		limit int,
	) ([]*NotificationOutboxEvent, error)

	// MarkProcessed finalises a successfully dispatched event.
	MarkProcessed(
		ctx context.Context,
		eventID string,
		processingToken string,
	) error

	// MarkFailed records a failed attempt and returns the event to
	// PENDING, to be retried at nextRetryAt.
	MarkFailed(
		ctx context.Context,
		eventID string,
		processingToken string,
		reason string,
		nextRetryAt time.Time,
	) error

	// MarkDead records the final failed attempt and moves the event to
	// the terminal FAILED state.
	MarkDead(
		ctx context.Context,
		eventID string,
		processingToken string,
		reason string,
	) error

	Heartbeat(
		ctx context.Context,
		eventID string,
		processingToken string,
	) (bool, error)
}
