package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	paymentapp "github.com/vishalyadav0987/authentication/internal/application/payment"
	domainpayment "github.com/vishalyadav0987/authentication/internal/domain/payment"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/payment/cashfree"
)

type PaymentWebhookHandler struct {
	cashfreeClient       *cashfree.Client
	handleWebhookUseCase *paymentapp.HandleWebhookUseCase
}

func NewPaymentWebhookHandler(
	cashfreeClient *cashfree.Client,
	handleWebhookUseCase *paymentapp.HandleWebhookUseCase,
) *PaymentWebhookHandler {
	return &PaymentWebhookHandler{
		cashfreeClient:       cashfreeClient,
		handleWebhookUseCase: handleWebhookUseCase,
	}
}

func (h *PaymentWebhookHandler) Handle(c *gin.Context) {
	signature := c.GetHeader("x-webhook-signature")
	timestamp := c.GetHeader("x-webhook-timestamp")

	// Read the original request body.
	rawBody, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to read webhook body",
		})
		return
	}

	// Verify Cashfree webhook signature before processing.
	err = h.cashfreeClient.VerifyWebhookSignature(
		signature,
		timestamp,
		rawBody,
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid webhook signature",
		})
		return
	}

	// Decode the verified webhook payload.
	var event domainpayment.WebhookEvent

	if err := json.Unmarshal(rawBody, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid webhook payload",
		})
		return
	}

	// Process the payment event.
	if err := h.handleWebhookUseCase.Execute(
		c.Request.Context(),
		event,
	); err != nil {
		log.Printf("Cashfree webhook processing failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to process payment webhook",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "webhook processed successfully",
	})
}
