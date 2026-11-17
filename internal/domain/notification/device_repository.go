package notification

import "context"

type DeviceRepository interface {
	// RegisterDevice upserts by FCM token: an existing token is
	// re-activated and moved to device.UserID. device.ID, CreatedAt and
	// UpdatedAt are filled from the stored row.
	RegisterDevice(
		ctx context.Context,
		device *UserDevice,
	) error

	GetActiveDevicesByUserID(
		ctx context.Context,
		userID string,
	) ([]*UserDevice, error)

	// GetActiveTokensByUserIDs returns each user's active FCM tokens in
	// one query. Users without active devices are absent from the map.
	GetActiveTokensByUserIDs(
		ctx context.Context,
		userIDs []string,
	) (map[string][]string, error)

	// DeactivateUserDevice deactivates one of userID's devices. Returns
	// ErrDeviceNotFound if the device does not belong to the user.
	DeactivateUserDevice(
		ctx context.Context,
		userID string,
		deviceID string,
	) error

	// DeactivateDeviceByToken is used when FCM reports a token as
	// unregistered.
	DeactivateDeviceByToken(
		ctx context.Context,
		token string,
	) error
}
