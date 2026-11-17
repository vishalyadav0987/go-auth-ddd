package notification

import (
	"context"
	"errors"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// UserDeviceUseCase registers and removes the authenticated user's push
// devices (FCM tokens).
type UserDeviceUseCase struct {
	devices domain.DeviceRepository
}

func NewUserDeviceUseCase(devices domain.DeviceRepository) *UserDeviceUseCase {
	return &UserDeviceUseCase{devices: devices}
}

// Register stores (or refreshes) an FCM token for userID. Calling it again
// with the same token is safe; the app should call it on every launch and
// whenever FCM rotates the token.
func (uc *UserDeviceUseCase) Register(
	ctx context.Context,
	userID string,
	fcmToken string,
	platform domain.DevicePlatform,
) (*domain.UserDevice, error) {

	device := &domain.UserDevice{
		UserID:   userID,
		FCMToken: fcmToken,
		Platform: platform,
	}

	if err := device.Validate(); err != nil {
		return nil, err
	}

	if err := uc.devices.RegisterDevice(ctx, device); err != nil {
		return nil, err
	}

	return device, nil
}

// Remove deactivates one of the user's devices (e.g. on logout).
func (uc *UserDeviceUseCase) Remove(
	ctx context.Context,
	userID string,
	deviceID string,
) error {

	if userID == "" {
		return errors.New("user ID is required")
	}

	if deviceID == "" {
		return domain.ErrDeviceNotFound
	}

	return uc.devices.DeactivateUserDevice(ctx, userID, deviceID)
}
