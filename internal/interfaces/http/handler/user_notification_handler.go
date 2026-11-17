package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	notificationApp "github.com/vishalyadav0987/authentication/internal/application/notification"
	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/internal/interfaces/http/middleware"
	"github.com/vishalyadav0987/authentication/pkg/response"
)

// UserNotificationHandler serves the authenticated user's notification
// APIs: inbox, read state, preferences and push devices.
//
// Every route requires middleware.RequireAccessToken; the user ID is taken
// from the token, never from the request.
type UserNotificationHandler struct {
	inbox       *notificationApp.UserInboxUseCase
	preferences *notificationApp.UserPreferencesUseCase
	devices     *notificationApp.UserDeviceUseCase
}

func NewUserNotificationHandler(
	inbox *notificationApp.UserInboxUseCase,
	preferences *notificationApp.UserPreferencesUseCase,
	devices *notificationApp.UserDeviceUseCase,
) *UserNotificationHandler {
	return &UserNotificationHandler{
		inbox:       inbox,
		preferences: preferences,
		devices:     devices,
	}
}

// ---------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------

type InboxItemResponse struct {
	ID        string                      `json:"id"`
	Type      domain.NotificationType     `json:"type"`
	Title     string                      `json:"title"`
	Body      string                      `json:"body"`
	Priority  domain.NotificationPriority `json:"priority"`
	Metadata  map[string]interface{}      `json:"metadata,omitempty"`
	IsRead    bool                        `json:"is_read"`
	ReadAt    *time.Time                  `json:"read_at"`
	CreatedAt time.Time                   `json:"created_at"`
}

type InboxListResponse struct {
	Items       []InboxItemResponse `json:"items"`
	UnreadCount int                 `json:"unread_count"`
	NextCursor  string              `json:"next_cursor,omitempty"`
}

type PreferenceResponse struct {
	Type         domain.NotificationType `json:"type"`
	EmailEnabled bool                    `json:"email_enabled"`
	PushEnabled  bool                    `json:"push_enabled"`
	Mandatory    bool                    `json:"mandatory"`
}

type DeviceResponse struct {
	ID         string                `json:"id"`
	Platform   domain.DevicePlatform `json:"platform"`
	IsActive   bool                  `json:"is_active"`
	LastUsedAt *time.Time            `json:"last_used_at"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}

func toInboxItemResponse(item *domain.InboxItem) InboxItemResponse {
	return InboxItemResponse{
		ID:        item.NotificationID,
		Type:      item.Type,
		Title:     item.Title,
		Body:      item.Body,
		Priority:  item.Priority,
		Metadata:  item.Metadata,
		IsRead:    item.IsRead(),
		ReadAt:    item.ReadAt,
		CreatedAt: item.CreatedAt,
	}
}

func toPreferenceResponses(views []notificationApp.PreferenceView) []PreferenceResponse {
	out := make([]PreferenceResponse, 0, len(views))
	for _, v := range views {
		out = append(out, PreferenceResponse{
			Type:         v.Type,
			EmailEnabled: v.EmailEnabled,
			PushEnabled:  v.PushEnabled,
			Mandatory:    v.Mandatory,
		})
	}
	return out
}

// ---------------------------------------------------------------------
// Inbox
// ---------------------------------------------------------------------

// ListNotifications handles GET /notifications?limit=&cursor=
func (h *UserNotificationHandler) ListNotifications(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	limit := 0
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			response.Error(c, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}

	page, err := h.inbox.List(c.Request.Context(), userID, limit, c.Query("cursor"))
	if err != nil {
		writeNotificationError(c, err)
		return
	}

	items := make([]InboxItemResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, toInboxItemResponse(item))
	}

	response.Success(c, http.StatusOK, InboxListResponse{
		Items:       items,
		UnreadCount: page.UnreadCount,
		NextCursor:  page.NextCursor,
	})
}

// GetNotification handles GET /notifications/:notificationID
func (h *UserNotificationHandler) GetNotification(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	item, err := h.inbox.Get(c.Request.Context(), userID, c.Param("notificationID"))
	if err != nil {
		writeNotificationError(c, err)
		return
	}

	response.Success(c, http.StatusOK, toInboxItemResponse(item))
}

// MarkAsRead handles PATCH /notifications/:notificationID/read
func (h *UserNotificationHandler) MarkAsRead(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	notificationID := c.Param("notificationID")

	readAt, err := h.inbox.MarkRead(c.Request.Context(), userID, notificationID)
	if err != nil {
		writeNotificationError(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"id":      notificationID,
		"is_read": true,
		"read_at": readAt,
	})
}

// MarkAllAsRead handles PATCH /notifications/read-all
func (h *UserNotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	updated, err := h.inbox.MarkAllRead(c.Request.Context(), userID)
	if err != nil {
		writeNotificationError(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"updated_count": updated})
}

// ---------------------------------------------------------------------
// Preferences
// ---------------------------------------------------------------------

// GetPreferences handles GET /notifications/preferences
func (h *UserNotificationHandler) GetPreferences(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	views, err := h.preferences.Get(c.Request.Context(), userID)
	if err != nil {
		writeNotificationError(c, err)
		return
	}

	response.Success(c, http.StatusOK, toPreferenceResponses(views))
}

type updatePreferencesRequest struct {
	Preferences []struct {
		Type         domain.NotificationType `json:"type" binding:"required"`
		EmailEnabled *bool                   `json:"email_enabled" binding:"required"`
		PushEnabled  *bool                   `json:"push_enabled" binding:"required"`
	} `json:"preferences" binding:"required,min=1,dive"`
}

// UpdatePreferences handles PUT /notifications/preferences
//
//	{"preferences": [{"type": "MARKETING", "email_enabled": false, "push_enabled": true}]}
func (h *UserNotificationHandler) UpdatePreferences(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	var req updatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest,
			"body must be {\"preferences\": [{\"type\", \"email_enabled\", \"push_enabled\"}]}")
		return
	}

	updates := make([]notificationApp.PreferenceUpdate, 0, len(req.Preferences))
	for _, p := range req.Preferences {
		updates = append(updates, notificationApp.PreferenceUpdate{
			Type:         p.Type,
			EmailEnabled: *p.EmailEnabled,
			PushEnabled:  *p.PushEnabled,
		})
	}

	views, err := h.preferences.Update(c.Request.Context(), userID, updates)
	if err != nil {
		writeNotificationError(c, err)
		return
	}

	response.Success(c, http.StatusOK, toPreferenceResponses(views))
}

// ---------------------------------------------------------------------
// Devices
// ---------------------------------------------------------------------

type registerDeviceRequest struct {
	FCMToken string                `json:"fcm_token" binding:"required"`
	Platform domain.DevicePlatform `json:"platform" binding:"required"`
}

// RegisterDevice handles POST /notifications/devices
//
//	{"fcm_token": "...", "platform": "ANDROID" | "IOS"}
func (h *UserNotificationHandler) RegisterDevice(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	var req registerDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "fcm_token and platform are required")
		return
	}

	device, err := h.devices.Register(c.Request.Context(), userID, req.FCMToken, req.Platform)
	if err != nil {
		writeNotificationError(c, err)
		return
	}

	// The token itself is not echoed back.
	response.Success(c, http.StatusOK, DeviceResponse{
		ID:         device.ID,
		Platform:   device.Platform,
		IsActive:   device.IsActive,
		LastUsedAt: device.LastUsedAt,
		CreatedAt:  device.CreatedAt,
		UpdatedAt:  device.UpdatedAt,
	})
}

// RemoveDevice handles DELETE /notifications/devices/:deviceID
func (h *UserNotificationHandler) RemoveDevice(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	if err := h.devices.Remove(c.Request.Context(), userID, c.Param("deviceID")); err != nil {
		writeNotificationError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ---------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------

// authenticatedUserID reads the user ID set by RequireAccessToken.
func authenticatedUserID(c *gin.Context) (string, bool) {
	userID := c.GetString(middleware.ContextUserID)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "authentication required")
		return "", false
	}

	return userID, true
}

// writeNotificationError maps domain errors to HTTP responses without
// leaking internal details.
func writeNotificationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInboxItemNotFound):
		response.Error(c, http.StatusNotFound, "notification not found")

	case errors.Is(err, domain.ErrDeviceNotFound):
		response.Error(c, http.StatusNotFound, "device not found")

	case errors.Is(err, notificationApp.ErrInvalidCursor):
		response.Error(c, http.StatusBadRequest, "invalid cursor")

	case errors.Is(err, domain.ErrInvalidType):
		response.Error(c, http.StatusBadRequest, "invalid notification type")

	case errors.Is(err, domain.ErrPreferenceNotEditable):
		response.Error(c, http.StatusBadRequest, "OTP and SECURITY_ALERT notifications cannot be disabled")

	case errors.Is(err, notificationApp.ErrDuplicatePreference):
		response.Error(c, http.StatusBadRequest, "each notification type may appear only once")

	case errors.Is(err, domain.ErrInvalidFCMToken):
		response.Error(c, http.StatusBadRequest, "invalid fcm_token")

	case errors.Is(err, domain.ErrInvalidPlatform):
		response.Error(c, http.StatusBadRequest, "platform must be ANDROID or IOS")

	default:
		log.Printf("notification API error: %v", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}
