package order

import "context"

// PaymentGateway interacts with external financial systems (Razorpay, BSE Star)
type PaymentGateway interface {
	// CreatePaymentIntent calls the third party and returns their specific Order ID
	CreatePaymentIntent(ctx context.Context, amount int64, receiptID string) (gatewayOrderID string, err error)

	// VerifySignature ensures incoming webhooks are actually from the provider
	VerifySignature(payload []byte, signature string) bool
}

// QueueManager handles background jobs via Redis
type QueueManager interface {
	// EnqueueOrderConfirmation pushes the email/SMS task to Asynq
	EnqueueOrderConfirmation(ctx context.Context, orderID string, userID string) error
}
