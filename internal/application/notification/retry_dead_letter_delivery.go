// Now we'll create the application use case that allows an administrator to request a retry.

// The responsibility of this use case is simple:
// 1. Validate the delivery ID.
// 2. Call PrepareDeadLetterRetry.
// 3. Receive the updated delivery.
// 4. Return retry information to the HTTP handler.
// The actual database transaction and outbox insertion are already handled by the repository.
package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/pkg/notiflog"
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

	// Step 3: Atomically prepare the retry.
	//
	// One repository transaction moves the delivery to a new retry cycle
	// and inserts the DELIVERY_RETRY_REQUESTED outbox event. The previous
	// cycle comes from the same UPDATE, so a concurrent retry request
	// cannot slip in between a read and the update.
	retriedDelivery, previousCycle, err := uc.notificationRepository.PrepareDeadLetterRetry(
		ctx,
		input.DeliveryID,
		input.AdminID,
	)
	lg := notiflog.From(ctx).With(
		notiflog.KeyDeliveryID, input.DeliveryID,
		notiflog.KeyAdminID, input.AdminID,
	)

	if err != nil {
		lg.Warn("dlq_retry.request_rejected", notiflog.Err(err))
		return nil, fmt.Errorf("failed to prepare delivery retry: %w", err)
	}

	if retriedDelivery == nil {
		lg.Warn("dlq_retry.request_rejected", "reason", "not dead-lettered")
		return nil, domain.ErrDeliveryNotDeadLettered
	}

	lg.Info("dlq_retry.requested",
		notiflog.KeyNotificationID, retriedDelivery.NotificationID,
		"previous_retry_cycle", previousCycle,
		notiflog.KeyRetryCycle, retriedDelivery.RetryCycle,
		"lifetime_attempts", retriedDelivery.AttemptCount,
	)

	// Step 4: Build the response.
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
