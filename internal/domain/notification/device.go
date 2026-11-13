package notification

import "time"

type DevicePlatform string

const (
	PlatformAndroid DevicePlatform = "ANDROID"
	PlatformIOS     DevicePlatform = "IOS"
)

type UserDevice struct {
	ID         string
	UserID     string
	FCMToken   string
	Platform   DevicePlatform
	IsActive   bool
	LastUsedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (p DevicePlatform) IsValid() bool {
	switch p {
	case PlatformAndroid, PlatformIOS:
		return true
	default:
		return false
	}
}
