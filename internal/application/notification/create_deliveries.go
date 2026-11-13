package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

type CreateDeliveriesUseCase struct {
	repository domain.NotificationRepository
}

func NewCreateDeliveriesUseCase(
	repository domain.NotificationRepository,
) *CreateDeliveriesUseCase {
	return &CreateDeliveriesUseCase{
		repository: repository,
	}
}

func (uc *CreateDeliveriesUseCase) Execute(
	ctx context.Context,
	notificationID string,
	userIDs []string,
	channels []domain.NotificationChannel,
) error {
	if notificationID == "" {
		return fmt.Errorf("notification ID is required")
	}

	if len(userIDs) == 0 {
		return domain.ErrNoRecipients
	}

	if len(channels) == 0 {
		return domain.ErrNoChannelsSpecified
	}

	now := time.Now().UTC()

	deliveries := make(
		[]*domain.NotificationDelivery,
		0,
		len(userIDs)*len(channels),
	)

	for _, userID := range userIDs {
		for _, channel := range channels {
			if userID == "" || !channel.IsValid() {
				return domain.ErrInvalidNotification
			}

			deliveries = append(deliveries, &domain.NotificationDelivery{
				ID:             uuid.NewString(),
				NotificationID: notificationID,
				UserID:         userID,
				Channel:        channel,
				Status:         domain.DeliveryPending,
				AttemptCount:   0,
				CreatedAt:      now,
				UpdatedAt:      now,
			})
		}
	}

	if err := uc.repository.CreateDeliveries(ctx, deliveries); err != nil {
		return fmt.Errorf("create notification deliveries: %w", err)
	}

	return nil
}
