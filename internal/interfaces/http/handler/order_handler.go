package handler

import (
	"errors"
	"net/http"
	"strings"

	apporder "github.com/vishalyadav0987/authentication/internal/application/order"
	domainorder "github.com/vishalyadav0987/authentication/internal/domain/order"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	createOrderUseCase *apporder.CreateOrderUseCase
	getUserID          func(*gin.Context) (string, error)
}

func NewOrderHandler(
	createOrderUseCase *apporder.CreateOrderUseCase,
	getUserID func(*gin.Context) (string, error),
) *OrderHandler {
	return &OrderHandler{
		createOrderUseCase: createOrderUseCase,
		getUserID:          getUserID,
	}
}

type CreateOrderRequest struct {
	SchemeCode     string `json:"scheme_code" binding:"required"`
	AmountPaise    int64  `json:"amount_paise" binding:"required,gt=0"`
	InvestmentType string `json:"investment_type" binding:"required"`
}

type OrderResponse struct {
	OrderID        string `json:"order_id"`
	SchemeCode     string `json:"scheme_code"`
	AmountPaise    int64  `json:"amount_paise"`
	InvestmentType string `json:"investment_type"`
	Status         string `json:"status"`
}

func (h *OrderHandler) CreateOrder(ctx *gin.Context) {
	userID, err := h.getUserID(ctx)
	if err != nil || strings.TrimSpace(userID) == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	var req CreateOrderRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	idempotencyKey := strings.TrimSpace(
		ctx.GetHeader("Idempotency-Key"),
	)

	if idempotencyKey == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Idempotency-Key header is required",
		})
		return
	}

	result, err := h.createOrderUseCase.Execute(
		ctx.Request.Context(),
		apporder.CreateOrderRequest{
			UserID:         userID,
			SchemeCode:     req.SchemeCode,
			AmountPaise:    req.AmountPaise,
			InvestmentType: domainorder.InvestmentType(req.InvestmentType),
			IdempotencyKey: idempotencyKey,
		},
	)

	if err != nil {
		status := http.StatusBadRequest

		if errors.Is(err, domainorder.ErrUnauthorized) {
			status = http.StatusUnauthorized
		}

		if errors.Is(err, domainorder.ErrIdempotencyConflict) {
			status = http.StatusConflict
		}

		ctx.JSON(status, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, OrderResponse{
		OrderID:        result.OrderID,
		SchemeCode:     result.SchemeCode,
		AmountPaise:    result.AmountPaise,
		InvestmentType: string(result.InvestmentType),
		Status:         string(result.Status),
	})
}
