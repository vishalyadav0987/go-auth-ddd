package payment

import "errors"

var (
	ErrPaymentNotFound       = errors.New("payment not found")
	ErrPaymentAlreadyExists  = errors.New("payment already exists")
	ErrInvalidPayment        = errors.New("invalid payment")
	ErrInvalidPaymentStatus  = errors.New("invalid payment status")
	ErrPaymentAmountMismatch = errors.New("payment amount mismatch")

	ErrPaymentOrderMismatch = errors.New("payment order mismatch")

	ErrPaymentAlreadySettled = errors.New("payment already settled")

	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")

	ErrIdempotencyConflict = errors.New("idempotency key already used with different payment details")

	ErrCashfreeRequestFailed = errors.New("cashfree request failed")

	ErrInvalidWebhook = errors.New("invalid cashfree webhook")

	ErrDuplicateWebhook = errors.New("duplicate webhook event")
)
