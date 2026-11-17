package notification

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

type CreateNotificationUseCase struct {
	repository domain.NotificationRepository
}

func NewCreateNotificationUseCase(
	repository domain.NotificationRepository,
) *CreateNotificationUseCase {
	return &CreateNotificationUseCase{
		repository: repository,
	}
}

// Execute validates and persists a notification.
func (uc *CreateNotificationUseCase) Execute(
	ctx context.Context,
	input *domain.Notification,
) error {

	if uc == nil || uc.repository == nil {
		return errors.New("notification repository is not configured")
	}

	if input == nil {
		return errors.New("notification is required")
	}

	if err := input.Validate(); err != nil {
		return fmt.Errorf("validate notification: %w", err)
	}

	if err := uc.repository.CreateNotification(ctx, input); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	return nil
}
