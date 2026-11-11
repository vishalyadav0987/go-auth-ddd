package order

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	domainorder "github.com/vishalyadav0987/authentication/internal/domain/order"
)

type CreateOrderRequest struct {
	UserID         string
	SchemeCode     string
	AmountPaise    int64
	InvestmentType domainorder.InvestmentType
	IdempotencyKey string
}

type OutboxEvent = domainorder.OutboxEvent

type OrderIDGenerator interface {
	NewOrderID() (string, error)
}

type CreateOrderUseCase struct {
	repository  domainorder.OrderRepository
	idGenerator OrderIDGenerator
}

func NewCreateOrderUseCase(
	repository domainorder.OrderRepository,
	idGenerator OrderIDGenerator,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		repository:  repository,
		idGenerator: idGenerator,
	}
}

func (uc *CreateOrderUseCase) Execute(
	ctx context.Context,
	req CreateOrderRequest,
) (*domainorder.Order, error) {

	if strings.TrimSpace(req.UserID) == "" {
		return nil, domainorder.ErrUnauthorized
	}

	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, errors.New("idempotency key is required")
	}

	if req.InvestmentType != domainorder.InvestmentTypeLumpsum {
		return nil, errors.New("only LUMPSUM orders are supported")
	}

	existingOrder, err := uc.repository.GetByIdempotencyKey(
		ctx,
		req.UserID,
		req.IdempotencyKey,
	)
	if err == nil {
		if existingOrder.SchemeCode != req.SchemeCode ||
			existingOrder.AmountPaise != req.AmountPaise ||
			existingOrder.InvestmentType != req.InvestmentType {
			return nil, domainorder.ErrIdempotencyConflict
		}

		return existingOrder, nil
	}

	if !errors.Is(err, domainorder.ErrOrderNotFound) {
		return nil, err
	}

	orderID, err := uc.idGenerator.NewOrderID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	newOrder := &domainorder.Order{
		OrderID:        orderID,
		UserID:         req.UserID,
		SchemeCode:     strings.TrimSpace(req.SchemeCode),
		AmountPaise:    req.AmountPaise,
		InvestmentType: req.InvestmentType,
		IdempotencyKey: req.IdempotencyKey,
		Status:         domainorder.OrderStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := newOrder.Validate(); err != nil {
		return nil, domainorder.ErrInvalidOrder
	}

	payload, err := json.Marshal(newOrder)
	if err != nil {
		return nil, err
	}

	event := &OutboxEvent{
		ID:          orderID + ".created",
		AggregateID: orderID,
		EventType:   "ORDER_CREATED",
		Payload:     payload,
		CreatedAt:   now,
	}

	if err := uc.repository.CreateWithOutbox(ctx, newOrder, event); err != nil {
		return nil, err
	}

	return newOrder, nil
}
