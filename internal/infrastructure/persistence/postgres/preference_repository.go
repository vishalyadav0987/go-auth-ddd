package postgres

import (
	"context"
	"database/sql"
	"fmt"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// PreferenceRepository is the PostgreSQL implementation of notification
// preferences.
type PreferenceRepository struct {
	db *sql.DB
}

var _ domain.PreferenceRepository = (*PreferenceRepository)(nil)

func NewPreferenceRepository(db *sql.DB) *PreferenceRepository {
	return &PreferenceRepository{db: db}
}

func (r *PreferenceRepository) GetPreferencesByUserID(
	ctx context.Context,
	userID string,
) ([]*domain.NotificationPreference, error) {

	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT user_id, type, email_enabled, push_enabled, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1
		`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get notification preferences: %w", err)
	}
	defer rows.Close()

	var preferences []*domain.NotificationPreference

	for rows.Next() {
		var (
			p        domain.NotificationPreference
			prefType string
		)

		if err := rows.Scan(
			&p.UserID,
			&prefType,
			&p.EmailEnabled,
			&p.PushEnabled,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification preference: %w", err)
		}

		p.Type = domain.NotificationType(prefType)
		preferences = append(preferences, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification preferences: %w", err)
	}

	return preferences, nil
}

// SavePreferences upserts every preference atomically: either all of the
// user's changes are stored or none are.
func (r *PreferenceRepository) SavePreferences(
	ctx context.Context,
	preferences []*domain.NotificationPreference,
) error {

	if len(preferences) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin preference transaction: %w", err)
	}
	defer tx.Rollback()

	for _, p := range preferences {
		if err := tx.QueryRowContext(
			ctx,
			`
			INSERT INTO notification_preferences (
				user_id, type, email_enabled, push_enabled,
				created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			ON CONFLICT (user_id, type) DO UPDATE
			SET email_enabled = EXCLUDED.email_enabled,
			    push_enabled = EXCLUDED.push_enabled,
			    updated_at = NOW()
			RETURNING created_at, updated_at
			`,
			p.UserID,
			string(p.Type),
			p.EmailEnabled,
			p.PushEnabled,
		).Scan(&p.CreatedAt, &p.UpdatedAt); err != nil {
			return fmt.Errorf("save preference %s: %w", p.Type, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit preferences: %w", err)
	}

	return nil
}

// GetEligibleUsers applies preferences (missing row = enabled; mandatory
// types bypass them) and channel reachability.
func (r *PreferenceRepository) GetEligibleUsers(
	ctx context.Context,
	notificationType domain.NotificationType,
	channel domain.NotificationChannel,
	userIDs []string,
) ([]string, error) {

	// Column and reachability clause come from a fixed whitelist, never
	// from input, so building the query is injection-safe.
	var enabledColumn, reachable string

	switch channel {
	case domain.ChannelEmail:
		enabledColumn = "p.email_enabled"
		reachable = "u.email IS NOT NULL AND u.email <> ''"

	case domain.ChannelPush:
		enabledColumn = "p.push_enabled"
		reachable = `EXISTS (
			SELECT 1 FROM user_devices d
			WHERE d.user_id = u.id AND d.is_active = TRUE
		)`

	default:
		return nil, domain.ErrInvalidChannel
	}

	if !notificationType.IsValid() {
		return nil, domain.ErrInvalidType
	}

	allUsers := userIDs == nil
	if userIDs == nil {
		userIDs = []string{}
	}

	query := `
		SELECT u.id
		FROM users u
		LEFT JOIN notification_preferences p
		       ON p.user_id = u.id AND p.type = $1
		WHERE ($2 OR u.id = ANY($3))
		  AND ($4 OR COALESCE(` + enabledColumn + `, TRUE))
		  AND ` + reachable + `
		ORDER BY u.id
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		string(notificationType),
		allUsers,
		userIDs,
		notificationType.IsMandatory(),
	)
	if err != nil {
		return nil, fmt.Errorf("get eligible users: %w", err)
	}
	defer rows.Close()

	var eligible []string

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan eligible user: %w", err)
		}

		eligible = append(eligible, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eligible users: %w", err)
	}

	return eligible, nil
}
