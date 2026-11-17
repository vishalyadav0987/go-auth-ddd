package order

import (
	"context"
	"errors"
	"time"
)

// ErrOutboxLeaseLost is returned when an order outbox event can no longer
// be updated by the caller because its lease expired and another worker
// reclaimed it (or it was already finalised).
var ErrOutboxLeaseLost = errors.New("order outbox event lease lost")

// OutboxEvent is a row in the order outbox (outbox_events).
//
// Order and payment events are written here; the order outbox worker
// publishes them to the order event queue.
type OutboxEvent struct {
	ID              string
	AggregateID     string
	EventType       string
	Payload         []byte
	ProcessingToken string
	CreatedAt       time.Time
}

// OrderOutboxRepository reads and updates order outbox events.
//
// Events are inserted atomically with their aggregate by
// OrderRepository.CreateWithOutbox and the payment repository.
type OrderOutboxRepository interface {
	// ClaimPending atomically claims up to limit PENDING events (or
	// PROCESSING events with an expired lease) and assigns each a fresh
	// ProcessingToken.
	ClaimPending(
		ctx context.Context,
		limit int,
	) ([]OutboxEvent, error)

	MarkProcessed(
		ctx context.Context,
		eventID string,
		processingToken string,
	) error

	MarkFailed(
		ctx context.Context,
		eventID string,
		processingToken string,
	) error
}
