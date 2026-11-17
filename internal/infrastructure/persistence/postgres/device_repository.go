package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// DeviceRepository is the PostgreSQL implementation of user devices
// (FCM tokens).
type DeviceRepository struct {
	db *sql.DB
}

var _ domain.DeviceRepository = (*DeviceRepository)(nil)

func NewDeviceRepository(db *sql.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// RegisterDevice upserts by FCM token. A token already registered (to this
// or another user) is re-activated and assigned to device.UserID, so a
// shared phone stops receiving the previous account's pushes.
func (r *DeviceRepository) RegisterDevice(
	ctx context.Context,
	device *domain.UserDevice,
) error {

	err := r.db.QueryRowContext(
		ctx,
		`
		INSERT INTO user_devices (
			id, user_id, fcm_token, platform,
			is_active, last_used_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, TRUE, NOW(), NOW(), NOW())
		ON CONFLICT (fcm_token) DO UPDATE
		SET user_id = EXCLUDED.user_id,
		    platform = EXCLUDED.platform,
		    is_active = TRUE,
		    last_used_at = NOW(),
		    updated_at = NOW()
		RETURNING id, is_active, last_used_at, created_at, updated_at
		`,
		uuid.NewString(),
		device.UserID,
		device.FCMToken,
		string(device.Platform),
	).Scan(
		&device.ID,
		&device.IsActive,
		&device.LastUsedAt,
		&device.CreatedAt,
		&device.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("register device: %w", err)
	}

	return nil
}

func (r *DeviceRepository) GetActiveDevicesByUserID(
	ctx context.Context,
	userID string,
) ([]*domain.UserDevice, error) {

	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT id, user_id, fcm_token, platform, is_active,
		       last_used_at, created_at, updated_at
		FROM user_devices
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY updated_at DESC
		`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get active devices: %w", err)
	}
	defer rows.Close()

	var devices []*domain.UserDevice

	for rows.Next() {
		var (
			d        domain.UserDevice
			platform string
		)

		if err := rows.Scan(
			&d.ID,
			&d.UserID,
			&d.FCMToken,
			&platform,
			&d.IsActive,
			&d.LastUsedAt,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}

		d.Platform = domain.DevicePlatform(platform)
		devices = append(devices, &d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate devices: %w", err)
	}

	return devices, nil
}

func (r *DeviceRepository) GetActiveTokensByUserIDs(
	ctx context.Context,
	userIDs []string,
) (map[string][]string, error) {

	tokens := make(map[string][]string)

	if len(userIDs) == 0 {
		return tokens, nil
	}

	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT user_id, fcm_token
		FROM user_devices
		WHERE user_id = ANY($1) AND is_active = TRUE
		ORDER BY user_id, updated_at DESC
		`,
		userIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("get active device tokens: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID, token string
		if err := rows.Scan(&userID, &token); err != nil {
			return nil, fmt.Errorf("scan device token: %w", err)
		}

		tokens[userID] = append(tokens[userID], token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device tokens: %w", err)
	}

	return tokens, nil
}

func (r *DeviceRepository) DeactivateUserDevice(
	ctx context.Context,
	userID string,
	deviceID string,
) error {

	result, err := r.db.ExecContext(
		ctx,
		`
		UPDATE user_devices
		SET is_active = FALSE,
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		`,
		deviceID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("deactivate device: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deactivated device: %w", err)
	}

	if affected == 0 {
		return domain.ErrDeviceNotFound
	}

	return nil
}

func (r *DeviceRepository) DeactivateDeviceByToken(
	ctx context.Context,
	token string,
) error {

	if _, err := r.db.ExecContext(
		ctx,
		`
		UPDATE user_devices
		SET is_active = FALSE,
		    updated_at = NOW()
		WHERE fcm_token = $1 AND is_active = TRUE
		`,
		token,
	); err != nil {
		return fmt.Errorf("deactivate device by token: %w", err)
	}

	return nil
}
