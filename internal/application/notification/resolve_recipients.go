package notification

import (
	"context"
	"fmt"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

type ResolveRecipientsUseCase struct {
	preferenceRepository domain.PreferenceRepository
}

func NewResolveRecipientsUseCase(
	preferenceRepository domain.PreferenceRepository,
) *ResolveRecipientsUseCase {
	return &ResolveRecipientsUseCase{
		preferenceRepository: preferenceRepository,
	}
}

func (uc *ResolveRecipientsUseCase) Execute(
	ctx context.Context,
	target domain.NotificationTarget,
	notificationType domain.NotificationType,
	channel domain.NotificationChannel,
) ([]string, error) {
	if !target.IsValid() {
		return nil, domain.ErrInvalidTarget
	}

	var userIDs []string

	switch target.Type {
	case domain.TargetAllUsers:
		userIDs = nil

	case domain.TargetSingleUser, domain.TargetSelectedUsers:
		userIDs = target.UserIDs

	case domain.TargetSegment:
		return nil, fmt.Errorf("segment targeting is not implemented yet")

	default:
		return nil, domain.ErrInvalidTarget
	}

	recipients, err := uc.preferenceRepository.GetEligibleUsers(
		ctx,
		notificationType,
		channel,
		userIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve notification recipients: %w", err)
	}

	if len(recipients) == 0 {
		return nil, domain.ErrNoRecipients
	}

	return recipients, nil
}
