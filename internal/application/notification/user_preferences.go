package notification

import (
	"context"
	"errors"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/pkg/notiflog"
)

// PreferenceView is the effective preference for one notification type.
type PreferenceView struct {
	Type         domain.NotificationType
	EmailEnabled bool
	PushEnabled  bool

	// Mandatory types are always delivered and cannot be switched off.
	Mandatory bool
}

// PreferenceUpdate is one requested change.
type PreferenceUpdate struct {
	Type         domain.NotificationType
	EmailEnabled bool
	PushEnabled  bool
}

// UserPreferencesUseCase reads and updates the authenticated user's
// notification preferences.
type UserPreferencesUseCase struct {
	preferences domain.PreferenceRepository
}

func NewUserPreferencesUseCase(
	preferences domain.PreferenceRepository,
) *UserPreferencesUseCase {
	return &UserPreferencesUseCase{preferences: preferences}
}

// Get returns the effective preference for every notification type,
// filling in defaults for types the user has never changed.
func (uc *UserPreferencesUseCase) Get(
	ctx context.Context,
	userID string,
) ([]PreferenceView, error) {

	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	stored, err := uc.preferences.GetPreferencesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	byType := make(map[domain.NotificationType]*domain.NotificationPreference, len(stored))
	for _, p := range stored {
		byType[p.Type] = p
	}

	types := domain.AllNotificationTypes()
	views := make([]PreferenceView, 0, len(types))

	for _, t := range types {
		p, ok := byType[t]
		if !ok || t.IsMandatory() {
			p = domain.DefaultPreference(userID, t)
		}

		views = append(views, PreferenceView{
			Type:         t,
			EmailEnabled: p.EmailEnabled,
			PushEnabled:  p.PushEnabled,
			Mandatory:    t.IsMandatory(),
		})
	}

	return views, nil
}

// Update stores the requested changes atomically and returns the full,
// updated preference list. Types not included are left unchanged.
//
// It rejects unknown types, duplicate types and attempts to switch off a
// mandatory type.
func (uc *UserPreferencesUseCase) Update(
	ctx context.Context,
	userID string,
	updates []PreferenceUpdate,
) ([]PreferenceView, error) {

	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	if len(updates) == 0 {
		return nil, errors.New("at least one preference is required")
	}

	seen := make(map[domain.NotificationType]bool, len(updates))
	toSave := make([]*domain.NotificationPreference, 0, len(updates))

	for _, u := range updates {
		if seen[u.Type] {
			return nil, ErrDuplicatePreference
		}
		seen[u.Type] = true

		p := &domain.NotificationPreference{
			UserID:       userID,
			Type:         u.Type,
			EmailEnabled: u.EmailEnabled,
			PushEnabled:  u.PushEnabled,
		}

		if err := p.Validate(); err != nil {
			return nil, err
		}

		toSave = append(toSave, p)
	}

	lg := notiflog.From(ctx).With(notiflog.KeyUserID, userID)

	if err := uc.preferences.SavePreferences(ctx, toSave); err != nil {
		lg.Error("preferences.update_failed", notiflog.Err(err))
		return nil, err
	}

	for _, p := range toSave {
		lg.Info("preferences.updated",
			"notification_type", p.Type,
			"email_enabled", p.EmailEnabled,
			"push_enabled", p.PushEnabled,
		)
	}

	return uc.Get(ctx, userID)
}

// ErrDuplicatePreference is returned when one request repeats a type.
var ErrDuplicatePreference = errors.New("duplicate notification type in request")
