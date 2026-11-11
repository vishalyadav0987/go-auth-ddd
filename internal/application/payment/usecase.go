package payment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domainorder "github.com/vishalyadav0987/authentication/internal/domain/order"
	domainpayment "github.com/vishalyadav0987/authentication/internal/domain/payment"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/payment/cashfree"
)

type PaymentIDGenerator interface {
	NewOrderID() string
}

type CashfreeClient interface {
	CreateOrder(
		ctx context.Context,
		idempotencyKey string,
		req cashfree.CreateOrderRequest,
	) (*cashfree.CreateOrderResponse, error)
}

type CreatePaymentInput struct {
	UserID         string
	OrderID        string
	IdempotencyKey string
	CustomerPhone  string
	CustomerEmail  string
	ReturnURL      string
	NotifyURL      string
}

type CreatePaymentOutput struct {
	PaymentID        string
	OrderID          string
	PaymentSessionID string
	AmountPaise      int64
	Status           domainpayment.PaymentStatus
}

type CreatePaymentUseCase struct {
	orderRepo   domainorder.OrderRepository
	paymentRepo domainpayment.PaymentRepository
	cashfree    CashfreeClient
	idGenerator PaymentIDGenerator
}

func NewCashfreeClient(config cashfree.Config) CashfreeClient {
	return cashfree.NewClient(config)
}

func NewCreatePaymentUseCase(
	orderRepo domainorder.OrderRepository,
	paymentRepo domainpayment.PaymentRepository,
	cashfreeClient CashfreeClient,
	idGenerator PaymentIDGenerator,
) *CreatePaymentUseCase {
	return &CreatePaymentUseCase{
		orderRepo:   orderRepo,
		paymentRepo: paymentRepo,
		cashfree:    cashfreeClient,
		idGenerator: idGenerator,
	}
}

func (uc *CreatePaymentUseCase) Execute(
	ctx context.Context,
	input CreatePaymentInput,
) (*CreatePaymentOutput, error) {
	if strings.TrimSpace(input.UserID) == "" {
		return nil, errors.New("user ID is required")
	}

	if strings.TrimSpace(input.OrderID) == "" {
		return nil, errors.New("order ID is required")
	}

	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return nil, domainpayment.ErrIdempotencyKeyRequired
	}

	if strings.TrimSpace(input.CustomerPhone) == "" {
		return nil, errors.New("customer phone is required")
	}

	if strings.TrimSpace(input.CustomerEmail) == "" {
		return nil, errors.New("customer email is required")
	}

	// Check whether this idempotency key was already used.
	existingPayment, err := uc.paymentRepo.GetByIdempotencyKey(
		ctx,
		input.UserID,
		input.IdempotencyKey,
	)

	if err == nil {
		if existingPayment.OrderID != input.OrderID {
			return nil, domainpayment.ErrIdempotencyConflict
		}

		return toCreatePaymentOutput(existingPayment), nil
	}

	if !errors.Is(err, domainpayment.ErrPaymentNotFound) {
		return nil, err
	}

	// Fetch the order.
	order, err := uc.orderRepo.GetByOrderID(
		ctx,
		input.OrderID,
	)
	if err != nil {
		return nil, err
	}

	if order.UserID != input.UserID {
		return nil, errors.New("order does not belong to user")
	}

	if string(order.Status) != "PENDING" {
		return nil, errors.New("order is not eligible for payment")
	}

	// Prevent creating another payment for the same order.
	existingPayment, err = uc.paymentRepo.GetByOrderID(
		ctx,
		input.OrderID,
	)

	if err == nil {
		return nil, domainpayment.ErrPaymentAlreadyExists
	}

	if !errors.Is(err, domainpayment.ErrPaymentNotFound) {
		return nil, err
	}

	paymentID := uc.idGenerator.NewOrderID()
	now := time.Now().UTC()

	cashfreeOrderID := "CF-" + paymentID

	// Create order with Cashfree.
	cashfreeResponse, err := uc.cashfree.CreateOrder(
		ctx,
		input.IdempotencyKey,
		cashfree.CreateOrderRequest{
			OrderID:       cashfreeOrderID,
			OrderAmount:   fmt.Sprintf("%.2f", float64(order.AmountPaise)/100),
			OrderCurrency: "INR",
			CustomerDetails: cashfree.CustomerDetails{
				CustomerID:    input.UserID,
				CustomerPhone: input.CustomerPhone,
				CustomerEmail: input.CustomerEmail,
			},
			OrderMeta: cashfree.OrderMeta{
				ReturnURL: input.ReturnURL,
				NotifyURL: input.NotifyURL,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			domainpayment.ErrCashfreeRequestFailed,
			err,
		)
	}

	p := &domainpayment.Payment{
		PaymentID:        paymentID,
		OrderID:          order.OrderID,
		UserID:           input.UserID,
		IdempotencyKey:   input.IdempotencyKey,
		CashfreeOrderID:  cashfreeResponse.OrderID,
		PaymentSessionID: cashfreeResponse.PaymentSessionID,
		AmountPaise:      order.AmountPaise,
		Currency:         "INR",
		Status:           domainpayment.PaymentStatusCreated,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	if err := uc.paymentRepo.CreatePayment(ctx, p); err != nil {
		return nil, err
	}

	return toCreatePaymentOutput(p), nil
}

func toCreatePaymentOutput(
	p *domainpayment.Payment,
) *CreatePaymentOutput {
	return &CreatePaymentOutput{
		PaymentID:        p.PaymentID,
		OrderID:          p.OrderID,
		PaymentSessionID: p.PaymentSessionID,
		AmountPaise:      p.AmountPaise,
		Status:           p.Status,
	}
}
