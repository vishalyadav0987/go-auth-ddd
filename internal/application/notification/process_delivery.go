// This use case will be responsible for processing an individual notification delivery.

// It retrieves the notification and delivery details, selects the correct provider, sends the message, and updates the delivery status.

package notification

import (
	"context"
	"fmt"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

type ProcessDeliveryUseCase struct {
	notificationRepository domain.NotificationRepository
	emailProvider          domain.EmailProvider
	pushProvider           domain.PushProvider
}

func NewProcessDeliveryUseCase(
	notificationRepository domain.NotificationRepository,
	emailProvider domain.EmailProvider,
	pushProvider domain.PushProvider,
) *ProcessDeliveryUseCase {
	return &ProcessDeliveryUseCase{
		notificationRepository: notificationRepository,
		emailProvider:          emailProvider,
		pushProvider:           pushProvider,
	}
}

func (uc *ProcessDeliveryUseCase) Execute(
	ctx context.Context,
	deliveryID string,
) error {
	delivery, err := uc.notificationRepository.GetDeliveryByID(
		ctx,
		deliveryID,
	)
	if err != nil {
		return fmt.Errorf("get notification delivery: %w", err)
	}

	if delivery == nil {
		return domain.ErrDeliveryNotFound
	}

	if delivery.Status != domain.DeliveryPending &&
		delivery.Status != domain.DeliveryRetrying {
		return domain.ErrDeliveryNotPending
	}

	notification, err := uc.notificationRepository.GetNotificationByID(
		ctx,
		delivery.NotificationID,
	)
	if err != nil {
		return fmt.Errorf("get notification: %w", err)
	}

	if notification == nil {
		return domain.ErrNotificationNotFound
	}

	if err := uc.notificationRepository.UpdateDeliveryStatus(
		ctx,
		delivery.ID,
		domain.DeliveryProcessing,
	); err != nil {
		return fmt.Errorf("mark delivery processing: %w", err)
	}

	if err := uc.notificationRepository.IncrementDeliveryAttempt(
		ctx,
		delivery.ID,
	); err != nil {
		return fmt.Errorf("increment delivery attempt: %w", err)
	}

	var providerID string

	switch delivery.Channel {
	case domain.ChannelEmail:
		if uc.emailProvider == nil {
			return fmt.Errorf("email provider is not configured")
		}

		err = uc.emailProvider.Send(
			ctx,
			delivery.Recipient,
			notification.Title,
			notification.HTMLBody,
		)

	case domain.ChannelPush:
		if uc.pushProvider == nil {
			return fmt.Errorf("push provider is not configured")
		}

		providerID, err = uc.pushProvider.Send(
			ctx,
			domain.PushMessage{
				Token: delivery.Recipient,
				Title: notification.Title,
				Body:  notification.Body,
			},
		)

	default:
		return domain.ErrInvalidChannel
	}

	if err != nil {
		_ = uc.notificationRepository.UpdateDeliveryStatus(
			ctx,
			delivery.ID,
			domain.DeliveryFailed,
		)

		return fmt.Errorf("send notification: %w", err)
	}

	if err := uc.notificationRepository.UpdateDeliveryProviderID(
		ctx,
		delivery.ID,
		providerID,
	); err != nil {
		return fmt.Errorf("save provider message ID: %w", err)
	}

	if err := uc.notificationRepository.UpdateDeliveryStatus(
		ctx,
		delivery.ID,
		domain.DeliveryAccepted,
	); err != nil {
		return fmt.Errorf("mark delivery accepted: %w", err)
	}

	return nil
}
