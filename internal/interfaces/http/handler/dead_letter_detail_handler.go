package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	domainApp "github.com/vishalyadav0987/authentication/internal/application/notification"
	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// DeadLetterDetailResponse represents one DLQ delivery
// along with its complete attempt history.
type DeadLetterDetailResponse struct {
	Delivery DeadLetterDeliveryDetailResponse `json:"delivery"`
	Attempts []DeliveryAttemptResponse        `json:"attempts"`
}

// DeadLetterDeliveryDetailResponse contains delivery information.
type DeadLetterDeliveryDetailResponse struct {
	DeliveryID     string                     `json:"delivery_id"`
	NotificationID string                     `json:"notification_id"`
	UserID         string                     `json:"user_id"`
	Channel        domain.NotificationChannel `json:"channel"`
	Recipient      string                     `json:"recipient"`
	Status         domain.DeliveryStatus      `json:"status"`
	ProviderID     string                     `json:"provider_id,omitempty"`
	FailureReason  string                     `json:"failure_reason"`
	AttemptCount   int                        `json:"attempt_count"`
	DeadLetteredAt *time.Time                 `json:"dead_lettered_at"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

// DeliveryAttemptResponse represents one individual delivery attempt.
type DeliveryAttemptResponse struct {
	AttemptID     string                `json:"attempt_id"`
	AttemptNumber int                   `json:"attempt_number"`
	Status        domain.DeliveryStatus `json:"status"`
	ProviderID    string                `json:"provider_id,omitempty"`
	ErrorMessage  string                `json:"error_message,omitempty"`
}

// maskDeadLetterRecipient hides sensitive recipient information.
func maskDeadLetterRecipient(
	channel domain.NotificationChannel,
	recipient string,
) string {

	if recipient == "" {
		return ""
	}

	switch channel {

	case domain.ChannelEmail:
		// Example: vishal@gmail.com -> v******@gmail.com
		at := -1

		for i, char := range recipient {
			if char == '@' {
				at = i
				break
			}
		}

		if at <= 0 || at == len(recipient)-1 {
			return "********"
		}

		return recipient[:1] + "******" + recipient[at:]

	case domain.ChannelPush:
		// FCM tokens should never be returned in full.
		if len(recipient) <= 8 {
			return "********"
		}

		return recipient[:4] + "********" + recipient[len(recipient)-4:]

	default:
		return "********"
	}
}

// toDeadLetterDetailResponse converts the application output
// into an API-safe response.
func toDeadLetterDetailResponse(
	output *domainApp.GetDeadLetterDetailOutput,
) *DeadLetterDetailResponse {

	if output == nil || output.Delivery == nil {
		return nil
	}

	delivery := output.Delivery

	response := &DeadLetterDetailResponse{
		Delivery: DeadLetterDeliveryDetailResponse{
			DeliveryID:     delivery.ID,
			NotificationID: delivery.NotificationID,
			UserID:         delivery.UserID,
			Channel:        delivery.Channel,
			Recipient:      maskDeadLetterRecipient(delivery.Channel, delivery.Recipient),
			Status:         delivery.Status,
			ProviderID:     delivery.ProviderID,
			FailureReason:  delivery.FailureReason,
			AttemptCount:   delivery.AttemptCount,
			DeadLetteredAt: delivery.DeadLetteredAt,
			CreatedAt:      delivery.CreatedAt,
			UpdatedAt:      delivery.UpdatedAt,
		},
		Attempts: make([]DeliveryAttemptResponse, 0, len(output.Attempts)),
	}

	for _, attempt := range output.Attempts {
		if attempt == nil {
			continue
		}

		response.Attempts = append(response.Attempts, DeliveryAttemptResponse{
			AttemptID:     attempt.ID,
			AttemptNumber: attempt.AttemptNumber,
			ProviderID:    attempt.ProviderID,
			ErrorMessage:  attempt.ErrorMessage,
			Status:        domain.DeliveryStatus(attempt.Status),
		})
	}

	return response
}

// DeadLetterDetailHandler handles requests for individual DLQ deliveries.
type DeadLetterDetailHandler struct {
	getDeadLetterDetail *domainApp.GetDeadLetterDetailUseCase
}

// NewDeadLetterDetailHandler creates the handler.
func NewDeadLetterDetailHandler(
	getDeadLetterDetail *domainApp.GetDeadLetterDetailUseCase,
) *DeadLetterDetailHandler {
	return &DeadLetterDetailHandler{
		getDeadLetterDetail: getDeadLetterDetail,
	}
}

// GetDeadLetterDetail retrieves one dead-lettered delivery
// and its complete attempt history.
func (h *DeadLetterDetailHandler) GetDeadLetterDetail(c *gin.Context) {

	// Step 1: Read the delivery ID from the URL.
	deliveryID := c.Param("deliveryID")

	if deliveryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "delivery ID is required",
		})
		return
	}

	// Step 2: Execute the application use case.
	output, err := h.getDeadLetterDetail.Execute(
		c.Request.Context(),
		deliveryID,
	)

	if err != nil {

		// Step 3: Return 404 for missing deliveries or deliveries
		// that have not been moved to the Dead Letter Queue.
		if errors.Is(err, domain.ErrDeliveryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "dead-letter delivery not found",
			})
			return
		}

		// Step 4: Do not expose internal database or repository errors.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve dead-letter delivery",
		})
		return
	}

	// Step 5: Convert the application output into an API response.
	response := toDeadLetterDetailResponse(output)

	c.JSON(http.StatusOK, response)
}
