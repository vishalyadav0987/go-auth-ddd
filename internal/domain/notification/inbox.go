package notification

import (
	"context"
	"errors"
	"time"
)

// ErrInboxItemNotFound is returned when the notification does not exist in
// the user's inbox. It is deliberately the same answer for "does not
// exist" and "belongs to another user".
var ErrInboxItemNotFound = errors.New("notification not found in inbox")

// InboxItem is one notification as seen in a user's inbox.
type InboxItem struct {
	NotificationID string
	Type           NotificationType
	Title          string
	Body           string
	Priority       NotificationPriority
	Metadata       map[string]interface{}

	ReadAt    *time.Time
	CreatedAt time.Time
}

// IsRead reports whether the user has read the notification.
func (i *InboxItem) IsRead() bool {
	return i.ReadAt != nil
}

// InboxCursor is a keyset pagination position: items strictly older than
// (CreatedAt, NotificationID) are returned next.
type InboxCursor struct {
	CreatedAt      time.Time
	NotificationID string
}

// InboxRepository reads and updates a user's inbox. Every method is scoped
// to userID, so one user can never read or modify another user's items.
type InboxRepository interface {
	// ListForUser returns up to limit items, newest first, starting after
	// the cursor (nil = first page). Notifications scheduled in the future
	// are not shown.
	ListForUser(
		ctx context.Context,
		userID string,
		limit int,
		after *InboxCursor,
	) ([]*InboxItem, error)

	// GetForUser returns ErrInboxItemNotFound if the item is not in the
	// user's inbox.
	GetForUser(
		ctx context.Context,
		userID string,
		notificationID string,
	) (*InboxItem, error)

	// MarkRead is idempotent: an already-read item keeps its original
	// read time. Returns the read time, or ErrInboxItemNotFound.
	MarkRead(
		ctx context.Context,
		userID string,
		notificationID string,
	) (time.Time, error)

	// MarkAllRead marks every unread item as read and returns how many
	// items changed.
	MarkAllRead(
		ctx context.Context,
		userID string,
	) (int64, error)

	CountUnread(
		ctx context.Context,
		userID string,
	) (int, error)
}
