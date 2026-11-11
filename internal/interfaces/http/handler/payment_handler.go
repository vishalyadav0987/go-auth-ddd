package handler

import (
	"errors"
	"net/http"
	"strings"

	paymentapp "github.com/vishalyadav0987/authentication/internal/application/payment"
	domainpayment "github.com/vishalyadav0987/authentication/internal/domain/payment"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	createPaymentUseCase *paymentapp.CreatePaymentUseCase
	getUserID            func(*gin.Context) (string, error)
}

func NewPaymentHandler(
	createPaymentUseCase *paymentapp.CreatePaymentUseCase,
	getUserID func(*gin.Context) (string, error),
) *PaymentHandler {
	return &PaymentHandler{
		createPaymentUseCase: createPaymentUseCase,
		getUserID:            getUserID,
	}
}

type CreatePaymentRequest struct {
	OrderID       string `json:"order_id" binding:"required"`
	CustomerPhone string `json:"customer_phone" binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required,email"`
	ReturnURL     string `json:"return_url"`
	NotifyURL     string `json:"notify_url"`
}

type CreatePaymentResponse struct {
	PaymentID        string                      `json:"payment_id"`
	OrderID          string                      `json:"order_id"`
	PaymentSessionID string                      `json:"payment_session_id"`
	AmountPaise      int64                       `json:"amount_paise"`
	Status           domainpayment.PaymentStatus `json:"status"`
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	idempotencyKey := strings.TrimSpace(
		c.GetHeader("Idempotency-Key"),
	)

	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Idempotency-Key header is required",
		})
		return
	}

	var req CreatePaymentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	req.OrderID = strings.TrimSpace(req.OrderID)
	req.CustomerPhone = strings.TrimSpace(req.CustomerPhone)
	req.CustomerEmail = strings.TrimSpace(req.CustomerEmail)

	output, err := h.createPaymentUseCase.Execute(
		c.Request.Context(),
		paymentapp.CreatePaymentInput{
			UserID:         userID,
			OrderID:        req.OrderID,
			IdempotencyKey: idempotencyKey,
			CustomerPhone:  req.CustomerPhone,
			CustomerEmail:  req.CustomerEmail,
			ReturnURL:      req.ReturnURL,
			NotifyURL:      req.NotifyURL,
		},
	)

	if err != nil {
		statusCode := http.StatusInternalServerError

		switch {
		case errors.Is(err, domainpayment.ErrPaymentNotFound):
			statusCode = http.StatusNotFound

		case errors.Is(err, domainpayment.ErrIdempotencyKeyRequired),
			errors.Is(err, domainpayment.ErrIdempotencyConflict),
			errors.Is(err, domainpayment.ErrPaymentAlreadyExists),
			errors.Is(err, domainpayment.ErrInvalidPayment):
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, CreatePaymentResponse{
		PaymentID:        output.PaymentID,
		OrderID:          output.OrderID,
		PaymentSessionID: output.PaymentSessionID,
		AmountPaise:      output.AmountPaise,
		Status:           output.Status,
	})
}
