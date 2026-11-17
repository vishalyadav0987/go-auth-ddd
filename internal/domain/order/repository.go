package order

import "context"

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
