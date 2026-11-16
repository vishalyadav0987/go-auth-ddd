package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	notificationApp "github.com/vishalyadav0987/authentication/internal/application/notification"
	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// DeadLetterDeliveryResponse represents one DLQ record
// exposed through the admin API.
type DeadLetterDeliveryResponse struct {
	DeliveryID       string                     `json:"delivery_id"`
	NotificationID   string                     `json:"notification_id"`
	NotificationType domain.NotificationType    `json:"notification_type"`
	Title            string                     `json:"title"`
	Channel          domain.NotificationChannel `json:"channel"`
	Recipient        string                     `json:"recipient"`
	Status           domain.DeliveryStatus      `json:"status"`
	FailureReason    string                     `json:"failure_reason"`
	AttemptCount     int                        `json:"attempt_count"`
	DeadLetteredAt   time.Time                  `json:"dead_lettered_at"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

// DeadLetterListResponse represents the paginated DLQ response.
type DeadLetterListResponse struct {
	Data    []*DeadLetterDeliveryResponse `json:"data"`
	Limit   int                           `json:"limit"`
	Offset  int                           `json:"offset"`
	HasMore bool                          `json:"has_more"`
}

// maskRecipient masks an FCM token or email address
// before exposing it through the admin API.
func maskRecipient(channel domain.NotificationChannel, recipient string) string {

	switch channel {

	case domain.ChannelPush:
		if len(recipient) <= 8 {
			return "********"
		}

		return recipient[:4] + "********" + recipient[len(recipient)-4:]

	case domain.ChannelEmail:
		at := -1

		for i, char := range recipient {
			if char == '@' {
				at = i
				break
			}
		}

		if at <= 0 {
			return "********"
		}

		return recipient[:1] + "********" + recipient[at:]

	default:
		return "********"
	}
}

// DeadLetterHandler handles admin DLQ requests.
type DeadLetterHandler struct {
	getDeadLetters *notificationApp.GetDeadLetterDeliveriesUseCase
}

// NewDeadLetterHandler creates the DLQ handler.
func NewDeadLetterHandler(
	getDeadLetters *notificationApp.GetDeadLetterDeliveriesUseCase,
) *DeadLetterHandler {

	return &DeadLetterHandler{
		getDeadLetters: getDeadLetters,
	}
}

// GetDeadLetters returns a paginated list of dead-lettered deliveries.
func (h *DeadLetterHandler) GetDeadLetters(c *gin.Context) {

	// STEP 1: Parse limit.
	limit := 20

	if value := c.Query("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid limit parameter",
			})
			return
		}

		limit = parsed
	}

	// STEP 2: Parse offset.
	offset := 0

	if value := c.Query("offset"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid offset parameter",
			})
			return
		}

		offset = parsed
	}

	// STEP 3: Execute application use case.
	result, err := h.getDeadLetters.Execute(
		c.Request.Context(),
		notificationApp.GetDeadLetterDeliveriesInput{
			Limit:  limit,
			Offset: offset,
		},
	)

	if err != nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})

		return
	}

	// STEP 4: Convert domain records into API DTOs.
	response := DeadLetterListResponse{
		Data:    make([]*DeadLetterDeliveryResponse, 0),
		Limit:   result.Limit,
		Offset:  result.Offset,
		HasMore: result.HasMore,
	}

	for _, delivery := range result.Deliveries {

		response.Data = append(
			response.Data,
			&DeadLetterDeliveryResponse{
				DeliveryID:       delivery.DeliveryID,
				NotificationID:   delivery.NotificationID,
				NotificationType: delivery.NotificationType,
				Title:            delivery.Title,
				Channel:          delivery.Channel,
				Recipient: maskRecipient(
					delivery.Channel,
					delivery.Recipient,
				),
				Status:         delivery.Status,
				FailureReason:  delivery.FailureReason,
				AttemptCount:   delivery.AttemptCount,
				DeadLetteredAt: delivery.DeadLetteredAt,
				CreatedAt:      delivery.CreatedAt,
				UpdatedAt:      delivery.UpdatedAt,
			},
		)
	}

	// STEP 5: Return successful response.
	c.JSON(http.StatusOK, response)
}
