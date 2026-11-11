package order

import "errors"

var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrOrderAlreadyExists  = errors.New("order already exists")
	ErrInvalidOrder        = errors.New("invalid order")
	ErrInvalidOrderStatus  = errors.New("invalid order status")
	ErrInvalidAmount       = errors.New("investment amount must be greater than zero")
	ErrIdempotencyConflict = errors.New("idempotency key already used with different request")
	ErrUnauthorized        = errors.New("unauthorized access to order")
)
