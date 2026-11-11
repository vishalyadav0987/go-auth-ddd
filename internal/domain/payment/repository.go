package payment

import "context"

type PaymentRepository interface {
	// Define methods for payment repository
	CreatePayment(ctx context.Context, payment *Payment) error
	GetPaymentByID(ctx context.Context, paymentID string) (*Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
	GetByCashFreeOrderID(ctx context.Context, cashFreeOrderID string) (*Payment, error)
	GetByIdempotencyKey(ctx context.Context, userID string, idempotencyKey string) (*Payment, error)
	UpdatePaymentStatus(ctx context.Context, paymentID string, status PaymentStatus) error
}
