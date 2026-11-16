// Now we'll combine two things:
// 1. Fetch the delivery and verify that it is actually in the Dead Letter Queue.
// 2. Fetch its complete attempt history.

package notification

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// GetDeadLetterDetailOutput contains a failed delivery and its
// complete attempt history.
type GetDeadLetterDetailOutput struct {
	Delivery *domain.NotificationDelivery
	Attempts []*domain.NotificationDeliveryAttempt
}

// GetDeadLetterDetailUseCase retrieves detailed information about
// one delivery that has been moved to the Dead Letter Queue.
type GetDeadLetterDetailUseCase struct {
	notificationRepository domain.NotificationRepository
	attemptRepository      domain.DeliveryAttemptRepository
}

// NewGetDeadLetterDetailUseCase creates the DLQ detail use case.
func NewGetDeadLetterDetailUseCase(
	notificationRepository domain.NotificationRepository,
	attemptRepository domain.DeliveryAttemptRepository,
) *GetDeadLetterDetailUseCase {
	return &GetDeadLetterDetailUseCase{
		notificationRepository: notificationRepository,
		attemptRepository:      attemptRepository,
	}
}

// Execute retrieves a DLQ delivery and all its delivery attempts.
func (uc *GetDeadLetterDetailUseCase) Execute(
	ctx context.Context,
	deliveryID string,
) (*GetDeadLetterDetailOutput, error) {

	if uc.notificationRepository == nil {
		return nil, errors.New("notification repository is not configured")
	}

	if uc.attemptRepository == nil {
		return nil, errors.New("delivery attempt repository is not configured")
	}

	if deliveryID == "" {
		return nil, fmt.Errorf("delivery ID is required")
	}

	// Step 1: Retrieve the delivery from the database.
	delivery, err := uc.notificationRepository.GetDeliveryByID(
		ctx,
		deliveryID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve delivery: %w", err)
	}

	if delivery == nil {
		return nil, domain.ErrDeliveryNotFound
	}

	// Step 2: Only expose deliveries that have actually entered the DLQ.
	// 	Why check DeadLetteredAt?
	// A delivery can have FAILED status without being in the DLQ. For example, a permanent provider rejection may mark a delivery as failed immediately.
	if delivery.DeadLetteredAt == nil {
		return nil, domain.ErrDeliveryNotFound
	}

	// Step 3: Retrieve every attempt made for this delivery.
	attempts, err := uc.attemptRepository.GetDeliveryAttemptsByDeliveryID(
		ctx,
		delivery.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve delivery attempt history: %w", err)
	}

	// Step 4: Return the delivery and its attempt history together.
	return &GetDeadLetterDetailOutput{
		Delivery: delivery,
		Attempts: attempts,
	}, nil
}
