package notification

import (
	"context"
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

func (uc *CreateNotificationUseCase) Execute(
	ctx context.Context,
	input *domain.Notification,
) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validate notification: %w", err)
	}

	if err := uc.repository.CreateNotification(ctx, input); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	return nil
}
