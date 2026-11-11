package order

import (
	"errors"
	"strings"
	"time"
)

// OrderStatus represents the lifecycle status of an order.
type OrderStatus string

// InvestmentType represents the type of investment.
type InvestmentType string

// Allowed order statuses.
const (
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusCompleted  OrderStatus = "COMPLETED"
	OrderStatusFailed     OrderStatus = "FAILED"
	OrderStatusCancelled  OrderStatus = "CANCELLED"
)

// Allowed investment types.
const (
	InvestmentTypeLumpsum InvestmentType = "LUMPSUM"
	InvestmentTypeSIP     InvestmentType = "SIP"
)

// Order represents a mutual fund investment order.
type Order struct {
	ID             int64
	OrderID        string
	UserID         string
	SchemeCode     string
	AmountPaise    int64
	InvestmentType InvestmentType
	IdempotencyKey string
	Status         OrderStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Validate checks whether the order contains valid basic information.
func (o *Order) Validate() error {
	if o == nil {
		return errors.New("order cannot be nil")
	}

	if strings.TrimSpace(o.OrderID) == "" {
		return errors.New("order ID is required")
	}

	if strings.TrimSpace(o.UserID) == "" {
		return errors.New("user ID is required")
	}

	if strings.TrimSpace(o.SchemeCode) == "" {
		return errors.New("scheme code is required")
	}

	if o.AmountPaise <= 0 {
		return errors.New("investment amount must be greater than zero")
	}

	switch o.InvestmentType {
	case InvestmentTypeLumpsum, InvestmentTypeSIP:
		// Valid investment type.
	default:
		return errors.New("invalid investment type")
	}

	switch o.Status {
	case OrderStatusPending,
		OrderStatusProcessing,
		OrderStatusCompleted,
		OrderStatusFailed,
		OrderStatusCancelled:
		// Valid order status.
	default:
		return errors.New("invalid order status")
	}

	return nil
}
