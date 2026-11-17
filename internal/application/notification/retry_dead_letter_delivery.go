package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// RetryDeadLetterDeliveryInput contains the administrator's retry request.
type RetryDeadLetterDeliveryInput struct {
	DeliveryID string
	AdminID    string
}

// RetryDeadLetterDeliveryOutput contains information about the
// newly scheduled retry cycle.
type RetryDeadLetterDeliveryOutput struct {
	DeliveryID     string
	NotificationID string
	Status         domain.DeliveryStatus
	PreviousCycle  int
	CurrentCycle   int
	AttemptCount   int
	RequestedBy    string
}

// RetryDeadLetterDeliveryUseCase handles manual retries of DLQ deliveries.
type RetryDeadLetterDeliveryUseCase struct {
	notificationRepository domain.NotificationRepository
}

// NewRetryDeadLetterDeliveryUseCase creates the use case.
func NewRetryDeadLetterDeliveryUseCase(
	notificationRepository domain.NotificationRepository,
) *RetryDeadLetterDeliveryUseCase {
	return &RetryDeadLetterDeliveryUseCase{
		notificationRepository: notificationRepository,
	}
}

// Execute starts a new retry cycle for a dead-lettered delivery.
func (uc *RetryDeadLetterDeliveryUseCase) Execute(
	ctx context.Context,
	input RetryDeadLetterDeliveryInput,
) (*RetryDeadLetterDeliveryOutput, error) {

	// Step 1: Validate repository dependency.
	if uc.notificationRepository == nil {
		return nil, errors.New("notification repository is not configured")
	}

	// Step 2: Validate request.
	if input.DeliveryID == "" {
		return nil, errors.New("delivery ID is required")
	}

	if input.AdminID == "" {
		return nil, errors.New("admin ID is required")
	}

	// Step 3: Retrieve the current delivery to capture its previous cycle.
	previousDelivery, err := uc.notificationRepository.GetDeliveryByID(
		ctx,
		input.DeliveryID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve delivery: %w", err)
	}

	if previousDelivery == nil || previousDelivery.DeadLetteredAt == nil {
		return nil, domain.ErrDeliveryNotDeadLettered
	}

	previousCycle := previousDelivery.RetryCycle

	// Step 4: Prepare the retry.
	//
	// The repository atomically resets the delivery and inserts
	// the DELIVERY_RETRY_REQUESTED outbox event.
	retriedDelivery, err := uc.notificationRepository.PrepareDeadLetterRetry(
		ctx,
		input.DeliveryID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare delivery retry: %w", err)
	}

	// Step 5: Build the response.
	return &RetryDeadLetterDeliveryOutput{
		DeliveryID:     retriedDelivery.ID,
		NotificationID: retriedDelivery.NotificationID,
		Status:         retriedDelivery.Status,
		PreviousCycle:  previousCycle,
		CurrentCycle:   retriedDelivery.RetryCycle,
		AttemptCount:   retriedDelivery.AttemptCount,
		RequestedBy:    input.AdminID,
	}, nil
}

// NewRetryRequestID can be used when a separate request identifier
// is required for retry audit records.
func NewRetryRequestID() string {
	return uuid.NewString()
}
