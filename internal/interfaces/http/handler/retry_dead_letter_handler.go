package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	notificationApp "github.com/vishalyadav0987/authentication/internal/application/notification"
	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/internal/interfaces/http/middleware"
)

// RetryDeadLetterResponse represents the result of a manual retry request.
type RetryDeadLetterResponse struct {
	Message        string                `json:"message"`
	DeliveryID     string                `json:"delivery_id"`
	NotificationID string                `json:"notification_id"`
	Status         domain.DeliveryStatus `json:"status"`
	PreviousCycle  int                   `json:"previous_retry_cycle"`
	CurrentCycle   int                   `json:"current_retry_cycle"`
	AttemptCount   int                   `json:"lifetime_attempt_count"`
	RequestedAt    time.Time             `json:"requested_at"`
}

// toRetryDeadLetterResponse converts the application output
// into the HTTP response format.
func toRetryDeadLetterResponse(
	output *notificationApp.RetryDeadLetterDeliveryOutput,
) *RetryDeadLetterResponse {

	if output == nil {
		return nil
	}

	return &RetryDeadLetterResponse{
		Message:        "Delivery retry successfully scheduled",
		DeliveryID:     output.DeliveryID,
		NotificationID: output.NotificationID,
		Status:         output.Status,
		PreviousCycle:  output.PreviousCycle,
		CurrentCycle:   output.CurrentCycle,
		AttemptCount:   output.AttemptCount,
		RequestedAt:    time.Now().UTC(),
	}
}

// RetryDeadLetterHandler handles administrator-initiated DLQ retries.
type RetryDeadLetterHandler struct {
	retryUseCase *notificationApp.RetryDeadLetterDeliveryUseCase
}

// NewRetryDeadLetterHandler creates the retry handler.
func NewRetryDeadLetterHandler(
	retryUseCase *notificationApp.RetryDeadLetterDeliveryUseCase,
) *RetryDeadLetterHandler {
	return &RetryDeadLetterHandler{
		retryUseCase: retryUseCase,
	}
}

// RetryDeadLetterDelivery starts a new retry cycle for a DLQ delivery.
func (h *RetryDeadLetterHandler) RetryDeadLetterDelivery(c *gin.Context) {

	// Step 1: Read the delivery ID.
	deliveryID := c.Param("deliveryID")

	if deliveryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "delivery ID is required",
		})
		return
	}

	// Step 2: Retrieve the authenticated administrator ID.
	//
	// Set by middleware.RequireAdmin. Never accept this value from the
	// request body.
	adminIDValue, exists := c.Get(middleware.ContextAdminID)

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "administrator authentication required",
		})
		return
	}

	adminID, ok := adminIDValue.(string)
	if !ok || adminID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid administrator identity",
		})
		return
	}

	// Step 3: Execute the retry use case.
	output, err := h.retryUseCase.Execute(
		c.Request.Context(),
		notificationApp.RetryDeadLetterDeliveryInput{
			DeliveryID: deliveryID,
			AdminID:    adminID,
		},
	)

	if err != nil {

		// Step 4: Handle a delivery that is not in the DLQ.
		if errors.Is(err, domain.ErrDeliveryNotDeadLettered) ||
			errors.Is(err, domain.ErrDeliveryNotFound) {

			c.JSON(http.StatusNotFound, gin.H{
				"error": "dead-letter delivery not found",
			})
			return
		}

		// Step 5: Do not expose internal database errors.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to schedule delivery retry",
		})
		return
	}

	// Step 6: Convert the result to the HTTP response DTO.
	response := toRetryDeadLetterResponse(output)

	// Step 7: Return 202 because processing happens asynchronously.
	c.JSON(http.StatusAccepted, response)
}
