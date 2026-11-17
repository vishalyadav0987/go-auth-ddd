package notification

import (
	"errors"
	"time"
)

// ErrPreferenceNotEditable is returned when a user tries to switch off a
// mandatory notification type.
var ErrPreferenceNotEditable = errors.New("notification type cannot be disabled")

// NotificationPreference holds one user's channel settings for one
// notification type. A user without a stored preference gets
// DefaultPreference (everything enabled).
type NotificationPreference struct {
	UserID string
	Type   NotificationType

	EmailEnabled bool
	PushEnabled  bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

// AllNotificationTypes lists every notification type, in display order.
func AllNotificationTypes() []NotificationType {
	return []NotificationType{
		NotificationOTP,
		NotificationSecurityAlert,
		NotificationOrderCreated,
		NotificationPaymentSuccess,
		NotificationPaymentFailed,
		NotificationInvestmentUpdate,
		NotificationAnnouncement,
		NotificationMarketing,
	}
}

// IsMandatory reports whether users must always receive this type.
// Mandatory types ignore stored preferences.
func (t NotificationType) IsMandatory() bool {
	switch t {
	case NotificationOTP, NotificationSecurityAlert:
		return true
	default:
		return false
	}
}

// DefaultPreference is the effective preference when none is stored.
func DefaultPreference(userID string, t NotificationType) *NotificationPreference {
	return &NotificationPreference{
		UserID:       userID,
		Type:         t,
		EmailEnabled: true,
		PushEnabled:  true,
	}
}

// Allows reports whether the preference permits delivery on channel.
func (p *NotificationPreference) Allows(channel NotificationChannel) bool {
	if p.Type.IsMandatory() {
		return true
	}

	switch channel {
	case ChannelEmail:
		return p.EmailEnabled
	case ChannelPush:
		return p.PushEnabled
	default:
		return false
	}
}

// Validate checks the preference before it is stored.
func (p *NotificationPreference) Validate() error {
	if p.UserID == "" {
		return errors.New("user ID is required")
	}

	if !p.Type.IsValid() {
		return ErrInvalidType
	}

	if p.Type.IsMandatory() && (!p.EmailEnabled || !p.PushEnabled) {
		return ErrPreferenceNotEditable
	}

	return nil
}
