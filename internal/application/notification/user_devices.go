package notification

import (
	"context"
	"errors"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/pkg/notiflog"
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

	lg := notiflog.From(ctx).With(
		notiflog.KeyUserID, userID,
		"platform", platform,
		"fcm_token", notiflog.MaskToken(fcmToken),
	)

	if err := device.Validate(); err != nil {
		lg.Warn("device.register_rejected", notiflog.Err(err))
		return nil, err
	}

	if err := uc.devices.RegisterDevice(ctx, device); err != nil {
		lg.Error("device.register_failed", notiflog.Err(err))
		return nil, err
	}

	lg.Info("device.registered", "device_id", device.ID)

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

	lg := notiflog.From(ctx).With(notiflog.KeyUserID, userID, "device_id", deviceID)

	if err := uc.devices.DeactivateUserDevice(ctx, userID, deviceID); err != nil {
		lg.Warn("device.remove_failed", notiflog.Err(err))
		return err
	}

	lg.Info("device.removed")

	return nil
}
