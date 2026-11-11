package payment

import (
	"context"
	"time"
)

type WebhookSettlement struct {
	PaymentID         string
	CashfreePaymentID string
	Status            PaymentStatus
	AmountPaise       int64
	Currency          string

	// Outbox event information.
	EventID        string
	EventType      string
	EventPayload   []byte
	EventCreatedAt time.Time
}

type PaymentRepository interface {
	// Define methods for payment repository
	CreatePayment(ctx context.Context, payment *Payment) error
	GetPaymentByID(ctx context.Context, paymentID string) (*Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
	GetByCashFreeOrderID(ctx context.Context, cashFreeOrderID string) (*Payment, error)
	GetByIdempotencyKey(ctx context.Context, userID string, idempotencyKey string) (*Payment, error)
	UpdatePaymentStatus(ctx context.Context, paymentID string, status PaymentStatus) error
	UpdateFromWebhook(
		ctx context.Context,
		paymentID string,
		cashfreePaymentID string,
		status PaymentStatus,
	) error

	// ApplyWebhook atomically updates payment and order
	// and inserts the corresponding outbox event.
	ApplyWebhook(
		ctx context.Context,
		settlement *WebhookSettlement,
	) error
}
