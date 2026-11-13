package notification

import "context"

type DeviceRepository interface {
	RegisterDevice(
		ctx context.Context,
		device *UserDevice,
	) error

	GetActiveDevicesByUserID(
		ctx context.Context,
		userID string,
	) ([]*UserDevice, error)

	GetAllActiveDevices(
		ctx context.Context,
	) ([]*UserDevice, error)

	UpdateDeviceToken(
		ctx context.Context,
		deviceID string,
		token string,
	) error

	DeactivateDevice(
		ctx context.Context,
		deviceID string,
	) error

	DeactivateDeviceByToken(
		ctx context.Context,
		token string,
	) error
}
