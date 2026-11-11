package payment

import (
	"errors"
	"strings"
	"time"
)

type PaymentStatus string

const (
	PaymentStatusCreated   PaymentStatus = "CREATED"
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusSuccess   PaymentStatus = "SUCCESS"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusCancelled PaymentStatus = "CANCELLED"
)

type Payment struct {
	ID                int64
	PaymentID         string
	OrderID           string
	UserID            string
	AmountPaise       int64
	CashfreeOrderID   string
	PaymentSessionID  string
	CashfreePaymentID string
	Currency          string
	Status            PaymentStatus
	IdempotencyKey    string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (p *Payment) Validate() error {
	if p == nil {
		return errors.New("payment cannot be nil")
	}
	if strings.TrimSpace(p.PaymentID) == "" {
		return errors.New("payment ID is required")
	}

	if strings.TrimSpace(p.OrderID) == "" {
		return errors.New("order ID is required")
	}

	if strings.TrimSpace(p.UserID) == "" {
		return errors.New("user ID is required")
	}

	if strings.TrimSpace(p.IdempotencyKey) == "" {
		return errors.New("idempotency key is required")
	}

	if strings.TrimSpace(p.CashfreeOrderID) == "" {
		return errors.New("cashfree order ID is required")
	}

	if p.AmountPaise <= 0 {
		return errors.New("payment amount must be greater than zero")
	}

	if p.Currency != "INR" {
		return errors.New("only INR payments are supported")
	}

	switch p.Status {
	case PaymentStatusCreated,
		PaymentStatusPending,
		PaymentStatusSuccess,
		PaymentStatusFailed,
		PaymentStatusCancelled:
	default:
		return errors.New("invalid payment status")
	}

	return nil
}
