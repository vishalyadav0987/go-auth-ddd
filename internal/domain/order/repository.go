package order

import (
	"context"
	"time"
)

type OutboxEvent struct {
	ID          string
	AggregateID string
	EventType   string
	Payload     []byte
	CreatedAt   time.Time
}

type OrderRepository interface {
	Create(ctx context.Context, order *Order) error

	GetByOrderID(ctx context.Context, orderID string) (*Order, error)

	GetByIdempotencyKey(
		ctx context.Context,
		userID string,
		idempotencyKey string,
	) (*Order, error)

	UpdateStatus(
		ctx context.Context,
		orderID string,
		status OrderStatus,
	) error

	CreateWithOutbox(
		ctx context.Context,
		order *Order,
		event *OutboxEvent,
	) error
}
