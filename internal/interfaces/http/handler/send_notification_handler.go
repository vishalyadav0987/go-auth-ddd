package handler

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	notificationApp "github.com/vishalyadav0987/authentication/internal/application/notification"
	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/pkg/response"
)

// SendNotificationHandler lets administrators create notifications
// (announcements, alerts, targeted messages).
type SendNotificationHandler struct {
	sendUseCase *notificationApp.SendNotificationUseCase
}

func NewSendNotificationHandler(
	sendUseCase *notificationApp.SendNotificationUseCase,
) *SendNotificationHandler {
	return &SendNotificationHandler{sendUseCase: sendUseCase}
}

type sendNotificationRequest struct {
	Type     domain.NotificationType      `json:"type" binding:"required"`
	Title    string                       `json:"title"`
	Body     string                       `json:"body"`
	HTMLBody string                       `json:"html_body"`
	Priority domain.NotificationPriority  `json:"priority" binding:"required"`
	Channels []domain.NotificationChannel `json:"channels" binding:"required,min=1"`
	Target   struct {
		Type    domain.NotificationTargetType `json:"type" binding:"required"`
		UserIDs []string                      `json:"user_ids"`
	} `json:"target" binding:"required"`
	Metadata    map[string]interface{} `json:"metadata"`
	ScheduledAt *time.Time             `json:"scheduled_at"`
}

// SendNotification handles POST /api/v1/notification/send (admin only).
//
//	{
//	  "type": "ANNOUNCEMENT", "priority": "NORMAL",
//	  "title": "Maintenance", "body": "Tonight 2-3 AM",
//	  "html_body": "<p>Tonight 2-3 AM</p>",
//	  "channels": ["EMAIL", "PUSH"],
//	  "target": {"type": "ALL_USERS"},
//	  "scheduled_at": "2026-11-20T02:00:00Z"
//	}
//
// Returns 202: deliveries are created now and sent asynchronously.
func (h *SendNotificationHandler) SendNotification(c *gin.Context) {
	var req sendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest,
			"type, priority, channels and target.type are required")
		return
	}

	result, err := h.sendUseCase.Execute(c.Request.Context(), &domain.Notification{
		Type:     req.Type,
		Title:    req.Title,
		Body:     req.Body,
		HTMLBody: req.HTMLBody,
		Target: domain.NotificationTarget{
			Type:    req.Target.Type,
			UserIDs: req.Target.UserIDs,
		},
		Channels:    req.Channels,
		Priority:    req.Priority,
		Metadata:    req.Metadata,
		ScheduledAt: req.ScheduledAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, notificationApp.ErrInvalidSendRequest):
			// Validation messages are safe to show to an admin.
			response.Error(c, http.StatusBadRequest, err.Error())

		case errors.Is(err, domain.ErrNoRecipients):
			response.Error(c, http.StatusUnprocessableEntity,
				"no eligible recipients (check target, preferences, emails and devices)")

		case errors.Is(err, domain.ErrInvalidTarget):
			response.Error(c, http.StatusBadRequest, "invalid target")

		default:
			log.Printf("send notification: %v", err)
			response.Error(c, http.StatusInternalServerError, "failed to create notification")
		}
		return
	}

	response.Success(c, http.StatusAccepted, gin.H{
		"notification_id":  result.NotificationID,
		"recipient_count":  result.RecipientCount,
		"email_deliveries": result.EmailDeliveries,
		"push_deliveries":  result.PushDeliveries,
		"scheduled_at":     result.ScheduledAt,
	})
}
