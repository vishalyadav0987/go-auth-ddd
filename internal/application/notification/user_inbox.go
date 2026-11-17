package notification

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

const (
	DefaultInboxPageSize = 20
	MaxInboxPageSize     = 100
)

// ErrInvalidCursor is returned for a malformed pagination cursor.
var ErrInvalidCursor = errors.New("invalid pagination cursor")

// InboxPage is one page of a user's inbox.
type InboxPage struct {
	Items       []*domain.InboxItem
	UnreadCount int

	// NextCursor is empty when there are no more items.
	NextCursor string
}

// UserInboxUseCase lists and reads the authenticated user's notifications.
// userID always comes from the authenticated identity.
type UserInboxUseCase struct {
	inbox domain.InboxRepository
}

func NewUserInboxUseCase(inbox domain.InboxRepository) *UserInboxUseCase {
	return &UserInboxUseCase{inbox: inbox}
}

// List returns one page, newest first. limit <= 0 uses the default page
// size; larger values are capped at MaxInboxPageSize.
func (uc *UserInboxUseCase) List(
	ctx context.Context,
	userID string,
	limit int,
	cursor string,
) (*InboxPage, error) {

	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	if limit <= 0 {
		limit = DefaultInboxPageSize
	}

	if limit > MaxInboxPageSize {
		limit = MaxInboxPageSize
	}

	after, err := decodeInboxCursor(cursor)
	if err != nil {
		return nil, err
	}

	// Fetch one extra row to know whether another page exists.
	items, err := uc.inbox.ListForUser(ctx, userID, limit+1, after)
	if err != nil {
		return nil, err
	}

	page := &InboxPage{Items: items}

	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		page.NextCursor = encodeInboxCursor(last.CreatedAt, last.NotificationID)
	}

	unread, err := uc.inbox.CountUnread(ctx, userID)
	if err != nil {
		return nil, err
	}

	page.UnreadCount = unread

	return page, nil
}

// Get returns one notification from the user's inbox.
func (uc *UserInboxUseCase) Get(
	ctx context.Context,
	userID string,
	notificationID string,
) (*domain.InboxItem, error) {

	if userID == "" || notificationID == "" {
		return nil, domain.ErrInboxItemNotFound
	}

	return uc.inbox.GetForUser(ctx, userID, notificationID)
}

// MarkRead marks one notification as read (idempotent).
func (uc *UserInboxUseCase) MarkRead(
	ctx context.Context,
	userID string,
	notificationID string,
) (time.Time, error) {

	if userID == "" || notificationID == "" {
		return time.Time{}, domain.ErrInboxItemNotFound
	}

	return uc.inbox.MarkRead(ctx, userID, notificationID)
}

// MarkAllRead marks every unread notification as read.
func (uc *UserInboxUseCase) MarkAllRead(
	ctx context.Context,
	userID string,
) (int64, error) {

	if userID == "" {
		return 0, errors.New("user ID is required")
	}

	return uc.inbox.MarkAllRead(ctx, userID)
}

// The cursor is opaque to clients: base64("<RFC3339Nano>|<notificationID>").
func encodeInboxCursor(createdAt time.Time, notificationID string) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + notificationID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeInboxCursor(cursor string) (*domain.InboxCursor, error) {
	if cursor == "" {
		return nil, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	createdAtText, notificationID, ok := strings.Cut(string(raw), "|")
	if !ok || notificationID == "" {
		return nil, ErrInvalidCursor
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtText)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCursor, err)
	}

	return &domain.InboxCursor{
		CreatedAt:      createdAt,
		NotificationID: notificationID,
	}, nil
}
