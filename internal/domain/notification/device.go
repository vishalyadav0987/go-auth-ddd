package notification

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrDeviceNotFound  = errors.New("device not found")
	ErrInvalidFCMToken = errors.New("invalid FCM token")
	ErrInvalidPlatform = errors.New("invalid device platform")
)

// maxFCMTokenLength bounds stored tokens; real FCM tokens are ~160-200 chars.
const maxFCMTokenLength = 4096

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

// Validate checks a device before registration.
func (d *UserDevice) Validate() error {
	if d.UserID == "" {
		return errors.New("user ID is required")
	}

	if d.FCMToken == "" ||
		len(d.FCMToken) > maxFCMTokenLength ||
		strings.ContainsAny(d.FCMToken, " \t\r\n") {
		return ErrInvalidFCMToken
	}

	if !d.Platform.IsValid() {
		return ErrInvalidPlatform
	}

	return nil
}
