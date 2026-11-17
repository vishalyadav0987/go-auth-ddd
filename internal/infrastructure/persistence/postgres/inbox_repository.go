package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// InboxRepository is the PostgreSQL implementation of the user inbox
// (user_notifications joined with notifications).
type InboxRepository struct {
	db *sql.DB
}

var _ domain.InboxRepository = (*InboxRepository)(nil)

func NewInboxRepository(db *sql.DB) *InboxRepository {
	return &InboxRepository{db: db}
}

// inboxSelect is shared by list and get. Scheduled notifications are
// hidden until their scheduled time.
const inboxSelect = `
	SELECT
		n.id,
		n.type,
		COALESCE(n.title, ''),
		COALESCE(n.body, ''),
		n.priority,
		n.metadata,
		un.read_at,
		un.created_at
	FROM user_notifications un
	JOIN notifications n ON n.id = un.notification_id
	WHERE un.user_id = $1
	  AND (n.scheduled_at IS NULL OR n.scheduled_at <= NOW())
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInboxItem(row rowScanner) (*domain.InboxItem, error) {
	var (
		item         domain.InboxItem
		itemType     string
		priority     string
		metadataJSON []byte
		readAt       sql.NullTime
	)

	if err := row.Scan(
		&item.NotificationID,
		&itemType,
		&item.Title,
		&item.Body,
		&priority,
		&metadataJSON,
		&readAt,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}

	item.Type = domain.NotificationType(itemType)
	item.Priority = domain.NotificationPriority(priority)

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &item.Metadata); err != nil {
			return nil, fmt.Errorf("decode notification metadata: %w", err)
		}
	}

	if readAt.Valid {
		t := readAt.Time
		item.ReadAt = &t
	}

	return &item, nil
}

// ListForUser returns one keyset-paginated page, newest first.
func (r *InboxRepository) ListForUser(
	ctx context.Context,
	userID string,
	limit int,
	after *domain.InboxCursor,
) ([]*domain.InboxItem, error) {

	if limit <= 0 {
		return nil, domain.ErrInvalidPagination
	}

	query := inboxSelect
	args := []any{userID, limit}

	if after != nil {
		query += `
	  AND (un.created_at, un.notification_id) < ($3, $4)`
		args = append(args, after.CreatedAt, after.NotificationID)
	}

	query += `
	ORDER BY un.created_at DESC, un.notification_id DESC
	LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list inbox: %w", err)
	}
	defer rows.Close()

	items := make([]*domain.InboxItem, 0, limit)

	for rows.Next() {
		item, err := scanInboxItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan inbox item: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inbox: %w", err)
	}

	return items, nil
}

func (r *InboxRepository) GetForUser(
	ctx context.Context,
	userID string,
	notificationID string,
) (*domain.InboxItem, error) {

	item, err := scanInboxItem(r.db.QueryRowContext(
		ctx,
		inboxSelect+`
	  AND un.notification_id = $2`,
		userID,
		notificationID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInboxItemNotFound
		}

		return nil, fmt.Errorf("get inbox item: %w", err)
	}

	return item, nil
}

// MarkRead keeps the original read time when the item is already read.
func (r *InboxRepository) MarkRead(
	ctx context.Context,
	userID string,
	notificationID string,
) (time.Time, error) {

	var readAt time.Time

	err := r.db.QueryRowContext(
		ctx,
		`
		UPDATE user_notifications un
		SET read_at = COALESCE(un.read_at, NOW())
		FROM notifications n
		WHERE un.user_id = $1
		  AND un.notification_id = $2
		  AND n.id = un.notification_id
		  AND (n.scheduled_at IS NULL OR n.scheduled_at <= NOW())
		RETURNING un.read_at
		`,
		userID,
		notificationID,
	).Scan(&readAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, domain.ErrInboxItemNotFound
		}

		return time.Time{}, fmt.Errorf("mark notification read: %w", err)
	}

	return readAt, nil
}

func (r *InboxRepository) MarkAllRead(
	ctx context.Context,
	userID string,
) (int64, error) {

	result, err := r.db.ExecContext(
		ctx,
		`
		UPDATE user_notifications un
		SET read_at = NOW()
		FROM notifications n
		WHERE un.user_id = $1
		  AND un.read_at IS NULL
		  AND n.id = un.notification_id
		  AND (n.scheduled_at IS NULL OR n.scheduled_at <= NOW())
		`,
		userID,
	)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count notifications marked read: %w", err)
	}

	return affected, nil
}

func (r *InboxRepository) CountUnread(
	ctx context.Context,
	userID string,
) (int, error) {

	var count int

	err := r.db.QueryRowContext(
		ctx,
		`
		SELECT COUNT(*)
		FROM user_notifications un
		JOIN notifications n ON n.id = un.notification_id
		WHERE un.user_id = $1
		  AND un.read_at IS NULL
		  AND (n.scheduled_at IS NULL OR n.scheduled_at <= NOW())
		`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}

	return count, nil
}
