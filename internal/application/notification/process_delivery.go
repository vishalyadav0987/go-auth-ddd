// This use case will be responsible for processing an individual notification delivery.

// It retrieves the notification and delivery details, selects the correct provider, sends the message, and updates the delivery status.

package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

const maxDeliveryAttempts = 5

type ProcessDeliveryUseCase struct {
	notificationRepository domain.NotificationRepository
	emailProvider          domain.EmailProvider
	pushProvider           domain.PushProvider
	attemptRepository      domain.DeliveryAttemptRepository
	deviceRepository       domain.DeviceRepository
}

func NewProcessDeliveryUseCase(
	notificationRepository domain.NotificationRepository,
	emailProvider domain.EmailProvider,
	pushProvider domain.PushProvider,
	attemptRepository domain.DeliveryAttemptRepository,
	deviceRepository domain.DeviceRepository,
) *ProcessDeliveryUseCase {
	return &ProcessDeliveryUseCase{
		notificationRepository: notificationRepository,
		emailProvider:          emailProvider,
		pushProvider:           pushProvider,
		attemptRepository:      attemptRepository,
		deviceRepository:       deviceRepository,
	}
}

// Execute processes one notification delivery.
//
// Processing lifecycle:
//
// 1. Load the delivery.
// 2. Atomically claim the delivery.
// 3. Load the parent notification.
// 4. Create a transactional delivery attempt.
// 5. Start a heartbeat to maintain processing ownership.
// 6. Send through the appropriate provider.
// 7. Classify provider errors.
// 8. Schedule retries or move exhausted deliveries to DLQ.
// 9. Update attempt and delivery status.
//
// It returns domain.ErrStaleDeliveryTask when the task belongs to an older
// retry cycle; callers should acknowledge such tasks without retrying.
func (uc *ProcessDeliveryUseCase) Execute(
	ctx context.Context,
	task DeliveryTask,
) error {

	if err := task.Validate(); err != nil {
		return fmt.Errorf("invalid delivery task: %w", err)
	}

	// ---------------------------------------------------------
	// STEP 1: Retrieve the delivery and reject stale tasks.
	// ---------------------------------------------------------

	delivery, err := uc.notificationRepository.GetDeliveryByID(
		ctx,
		task.DeliveryID,
	)
	if err != nil {
		return fmt.Errorf("get notification delivery: %w", err)
	}

	if err := ValidateDeliveryTask(delivery, task); err != nil {
		return err
	}

	// ---------------------------------------------------------
	// STEP 2: Atomically claim the delivery.
	//
	// Every worker generates a unique processing token.
	// Only the worker whose token is stored in PostgreSQL
	// can update this delivery.
	// ---------------------------------------------------------

	processingToken := uuid.NewString()

	// The claim only succeeds if the delivery is still in the task's
	// retry cycle, so a DLQ retry racing with this task cannot let a
	// stale task process the new cycle.
	claimed, err := uc.notificationRepository.ClaimDelivery(
		ctx,
		delivery.ID,
		processingToken,
		task.RetryCycle,
	)
	if err != nil {
		return fmt.Errorf("claim notification delivery: %w", err)
	}

	if !claimed {
		// Another worker owns the delivery, or the delivery
		// is not currently eligible for processing.
		return nil
	}

	// ---------------------------------------------------------
	// STEP 3: Retrieve the parent notification.
	// ---------------------------------------------------------

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

	// ---------------------------------------------------------
	// STEP 4: Create a transactional delivery attempt.
	//
	// This repository operation:
	// - Locks the delivery row.
	// - Verifies the processing token.
	// - Checks the maximum attempt limit.
	// - Increments attempt_count.
	// - Inserts a PROCESSING attempt record.
	//
	// All operations happen in one database transaction.
	// ---------------------------------------------------------

	attempt := &domain.NotificationDeliveryAttempt{
		ID:     uuid.NewString(),
		Status: "PROCESSING",
	}

	attemptNumber, err := uc.notificationRepository.CreateDeliveryAttempt(
		ctx,
		delivery.ID,
		processingToken,
		attempt,
	)

	if err != nil {

		// This is a defensive path. Normally, the worker should
		// not claim a delivery whose attempt limit is exhausted.
		//
		// If the repository confirms that the limit has been
		// reached, move the delivery into the dead-letter queue.

		if errors.Is(err, domain.ErrMaxRetriesExceeded) {

			if updateErr := uc.notificationRepository.MarkDeliveryDeadLettered(
				ctx,
				delivery.ID,
				processingToken,
				"Maximum delivery attempts exceeded",
			); updateErr != nil {
				return fmt.Errorf(
					"mark exhausted delivery as dead-lettered: %w",
					updateErr,
				)
			}

			return nil
		}

		return fmt.Errorf(
			"create transactional delivery attempt: %w",
			err,
		)
	}

	// ---------------------------------------------------------
	// STEP 5: Start delivery heartbeat.
	//
	// The heartbeat periodically refreshes the processing
	// lease in PostgreSQL.
	//
	// If this worker loses ownership, the provider context
	// is cancelled to prevent further processing.
	// ---------------------------------------------------------

	sendCtx, stopHeartbeat := uc.startDeliveryHeartbeat(
		ctx,
		delivery.ID,
		processingToken,
		task.RetryCycle,
	)

	// ---------------------------------------------------------
	// STEP 6: Send through the appropriate provider.
	//
	// Email and push have independent provider timeouts.
	// ---------------------------------------------------------

	var providerID string
	var sendErr error

	switch delivery.Channel {

	case domain.ChannelEmail:

		if uc.emailProvider == nil {
			sendErr = fmt.Errorf("email provider is not configured")
			break
		}

		providerCtx, cancel := context.WithTimeout(
			sendCtx,
			30*time.Second,
		)

		sendErr = uc.emailProvider.Send(
			providerCtx,
			delivery.Recipient,
			notification.Title,
			notification.HTMLBody,
		)

		cancel()

	case domain.ChannelPush:

		if uc.pushProvider == nil {
			sendErr = fmt.Errorf("push provider is not configured")
			break
		}

		providerCtx, cancel := context.WithTimeout(
			sendCtx,
			20*time.Second,
		)

		providerID, sendErr = uc.pushProvider.Send(
			providerCtx,
			domain.PushMessage{
				Token: delivery.Recipient,
				Title: notification.Title,
				Body:  notification.Body,
			},
		)

		cancel()

	default:
		sendErr = domain.ErrInvalidChannel
	}

	// ---------------------------------------------------------
	// STEP 7: Stop heartbeat and verify ownership.
	//
	// The heartbeat is stopped before updating delivery
	// history or status.
	// ---------------------------------------------------------

	ownershipLost := stopHeartbeat()

	if ownershipLost {
		return fmt.Errorf(
			"delivery processing ownership lost: %s",
			delivery.ID,
		)
	}

	// ---------------------------------------------------------
	// STEP 8: Handle provider failure.
	//
	// First update the current attempt history.
	// Then classify the failure and decide what happens next.
	// ---------------------------------------------------------

	if sendErr != nil {

		decision := ClassifyProviderError(sendErr)

		// The provider attempt has ended with an error.
		// Preserve the error in the attempt history.
		if err := uc.attemptRepository.UpdateAttempt(
			ctx,
			delivery.ID,
			attemptNumber,
			"FAILED",
			"",
			sendErr.Error(),
			processingToken,
		); err != nil {
			return fmt.Errorf(
				"provider failed: %v; update failed attempt: %w",
				sendErr,
				err,
			)
		}

		switch decision {

		// -------------------------------------------------
		// STEP 8A: Transient failure.
		//
		// Examples:
		// - SMTP temporary server error.
		// - Firebase unavailable.
		// - Firebase quota exceeded.
		//
		// Retry unless the maximum attempt count is reached.
		// -------------------------------------------------

		case FailureRetry:

			if attemptNumber >= maxDeliveryAttempts {

				// All allowed attempts have been exhausted.
				// Move the delivery into the dead-letter queue.
				if err := uc.notificationRepository.MarkDeliveryDeadLettered(
					ctx,
					delivery.ID,
					processingToken,
					sendErr.Error(),
				); err != nil {
					return fmt.Errorf(
						"provider failed: %v; dead-letter delivery: %w",
						sendErr,
						err,
					)
				}

				return nil
			}

			// Calculate exponential backoff with jitter.
			// Provider Retry-After is respected when available.
			retryDelay := calculateProviderRetryDelay(
				attemptNumber,
				sendErr,
			)

			nextRetryAt := time.Now().
				Add(retryDelay).
				Unix()

			if err := uc.notificationRepository.UpdateDeliveryRetry(
				ctx,
				delivery.ID,
				processingToken,
				domain.DeliveryRetrying,
				nextRetryAt,
				sendErr.Error(),
			); err != nil {
				return fmt.Errorf(
					"provider failed: %v; schedule retry: %w",
					sendErr,
					err,
				)
			}

			return nil

		// -------------------------------------------------
		// STEP 8B: Permanent failure.
		//
		// Examples:
		// - Invalid email recipient.
		// - FCM token is permanently unregistered.
		//
		// Do not retry permanent provider failures.
		// -------------------------------------------------

		case FailurePermanent:

			// An unregistered FCM token should no longer
			// receive push notifications.
			//
			// Do not deactivate tokens for every permanent
			// FCM error. Only FCM_UNREGISTERED proves that
			// this registration token is no longer valid.

			if delivery.Channel == domain.ChannelPush &&
				isFCMUnregisteredError(sendErr) {

				if uc.deviceRepository == nil {
					return fmt.Errorf(
						"FCM token is unregistered but device repository is not configured",
					)
				}

				if err := uc.deviceRepository.DeactivateDeviceByToken(
					ctx,
					delivery.Recipient,
				); err != nil {
					return fmt.Errorf(
						"deactivate unregistered FCM device: %w",
						err,
					)
				}
			}

			// Permanent failures are FAILED, but they are
			// not automatically dead-lettered.
			return uc.notificationRepository.UpdateDeliveryRetry(
				ctx,
				delivery.ID,
				processingToken,
				domain.DeliveryFailed,
				0,
				sendErr.Error(),
			)

		// -------------------------------------------------
		// STEP 8C: Ambiguous provider outcome.
		//
		// Examples:
		// - Network timeout during SMTP DATA.
		// - Connection lost after provider submission.
		//
		// The provider may already have accepted the message.
		// Retrying blindly could send a duplicate.
		// -------------------------------------------------

		case FailureAmbiguous:

			return uc.notificationRepository.UpdateDeliveryStatus(
				ctx,
				delivery.ID,
				processingToken,
				domain.DeliveryUnknown,
			)

		default:

			return fmt.Errorf(
				"unsupported provider failure decision: %s",
				decision,
			)
		}
	}

	// ---------------------------------------------------------
	// STEP 9: Update the attempt to ACCEPTED.
	//
	// This means the provider accepted the submission.
	// It does not mean the recipient received or read it.
	// ---------------------------------------------------------

	if err := uc.attemptRepository.UpdateAttempt(
		ctx,
		delivery.ID,
		attemptNumber,
		"ACCEPTED",
		providerID,
		"",
		processingToken,
	); err != nil {
		return fmt.Errorf(
			"update successful delivery attempt: %w",
			err,
		)
	}

	// ---------------------------------------------------------
	// STEP 10: Save provider message ID.
	//
	// Push providers may return a message ID.
	// The current SMTP provider does not return one.
	// ---------------------------------------------------------

	if providerID != "" {

		if err := uc.notificationRepository.UpdateDeliveryProviderID(
			ctx,
			delivery.ID,
			processingToken,
			providerID,
		); err != nil {
			return fmt.Errorf(
				"save provider message ID: %w",
				err,
			)
		}
	}

	// ---------------------------------------------------------
	// STEP 11: Mark delivery as ACCEPTED.
	//
	// This is the final state for successful provider
	// submission. Delivery confirmation, if supported,
	// should be handled separately through provider events.
	// ---------------------------------------------------------

	if err := uc.notificationRepository.UpdateDeliveryStatus(
		ctx,
		delivery.ID,
		processingToken,
		domain.DeliveryAccepted,
	); err != nil {
		return fmt.Errorf(
			"mark delivery accepted: %w",
			err,
		)
	}

	return nil
}

// isFCMUnregisteredError checks whether Firebase explicitly
// reported that the registration token is no longer valid.
//
// Only this error should trigger automatic device deactivation.
func isFCMUnregisteredError(err error) bool {

	providerErr, ok := domain.IsProviderError(err)
	if !ok {
		return false
	}

	return providerErr.Code == "FCM_UNREGISTERED"
}

// flowchart TD
//     A[Load Delivery]
//     B[Increment Attempt Count]
//     C[Create PROCESSING Attempt]
//     D[Send Email or Push]
//     A --> B --> C --> D

// In Step 90, we added the PostgreSQL heartbeat method. Now we need to actually call it while an email or push notification is being sent.

// Our agreed configuration:
// Setting	Value
// Processing lease	5 minutes
// Heartbeat interval	1 minute
// Heartbeat database timeout	10 seconds

// How it works:
// - A goroutine starts alongside the provider operation.
// - Every minute, it refreshes processing_started_at.
// - If the database says the worker no longer owns the delivery, it cancels the processing context.
// - If the heartbeat fails, it also cancels processing rather than allowing the worker to continue without a confirmed lease.
// - stop() waits for the heartbeat goroutine to finish before reading its result.
// 2. Integrate it into Execute
// Find the section in Execute where you send the email or push notification.

func (uc *ProcessDeliveryUseCase) startDeliveryHeartbeat(
	ctx context.Context,
	deliveryID string,
	processingToken string,
	retryCycle int,
) (context.Context, func() bool) {
	heartbeatCtx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})
	ownershipLost := false

	go func() {
		defer close(done)

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				return

			case <-ticker.C:
				heartbeatTimeoutCtx, heartbeatCancel :=
					context.WithTimeout(heartbeatCtx, 10*time.Second)

				owned, err := uc.notificationRepository.HeartbeatDelivery(
					heartbeatTimeoutCtx,
					deliveryID,
					processingToken,
					retryCycle,
				)

				heartbeatCancel()

				if err != nil || !owned {
					ownershipLost = true
					cancel()
					return
				}
			}
		}
	}()

	stop := func() bool {
		cancel()
		<-done
		return ownershipLost
	}

	return heartbeatCtx, stop
}
