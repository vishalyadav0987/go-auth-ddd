// The notification outbox worker claims events (ClaimPending), and this
// use case dispatches one claimed event and records the result.

package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/pkg/notiflog"
)

// MaxOutboxAttempts is how many times a notification outbox event is
// dispatched before it is moved to the terminal FAILED state.
const MaxOutboxAttempts = 10

// OutboxDispatcher handles the business processing of an outbox event.
type OutboxDispatcher interface {
	Dispatch(
		ctx context.Context,
		event *notification.NotificationOutboxEvent,
	) error
}

// ProcessOutboxEventUseCase processes one claimed outbox event.
//
// The outbox repository is responsible for claiming events and
// assigning a unique processing token.
//
// The heartbeat keeps the processing lease alive while the
// dispatcher performs the actual event processing.
type ProcessOutboxEventUseCase struct {
	outboxRepository notification.NotificationOutboxRepository
	dispatcher       OutboxDispatcher
	heartbeat        *OutboxHeartbeat
}

// NewProcessOutboxEventUseCase creates the outbox processor.
func NewProcessOutboxEventUseCase(
	outboxRepository notification.NotificationOutboxRepository,
	dispatcher OutboxDispatcher,
	heartbeat *OutboxHeartbeat,
) *ProcessOutboxEventUseCase {
	return &ProcessOutboxEventUseCase{
		outboxRepository: outboxRepository,
		dispatcher:       dispatcher,
		heartbeat:        heartbeat,
	}
}

// Execute processes one previously claimed outbox event.
//
// Important:
// This method does not claim the event itself.
// ClaimPending must have already claimed the event and assigned
// its processing token.
func (uc *ProcessOutboxEventUseCase) Execute(
	ctx context.Context,
	event *notification.NotificationOutboxEvent,
) error {

	// Step 1: Validate the event.
	if event == nil {
		return errors.New("outbox event cannot be nil")
	}

	if event.ID == "" {
		return errors.New("outbox event ID cannot be empty")
	}

	if event.ProcessingToken == "" {
		return errors.New("outbox event processing token is missing")
	}

	// Step 2: Validate dependencies.
	if uc.outboxRepository == nil {
		return errors.New("outbox repository is not configured")
	}

	if uc.dispatcher == nil {
		return errors.New("outbox dispatcher is not configured")
	}

	if uc.heartbeat == nil {
		return errors.New("outbox heartbeat is not configured")
	}

	start := time.Now()

	// Every log below, and in the dispatched handler, carries these.
	ctx = notiflog.With(ctx,
		notiflog.KeyOutboxEventID, event.ID,
		notiflog.KeyEventType, event.EventType,
		"aggregate_id", event.AggregateID,
		"outbox_attempt", event.Attempts+1,
	)
	lg := notiflog.From(ctx)

	lg.Info("outbox.event_processing_started",
		"max_outbox_attempts", MaxOutboxAttempts,
		"previous_error", event.LastError,
	)

	// Step 3-4: Start the heartbeat.
	//
	// dispatchCtx is cancelled by the heartbeat if ownership is lost or a
	// lease renewal fails, which aborts the in-flight dispatch.
	dispatchCtx, stopHeartbeat, err := uc.heartbeat.Start(
		ctx,
		event.ID,
		event.ProcessingToken,
	)
	if err != nil {
		lg.Error("outbox.heartbeat_start_failed", notiflog.Err(err))
		return fmt.Errorf("start outbox heartbeat: %w", err)
	}
	defer stopHeartbeat()

	// Step 5: Dispatch the event.
	dispatchErr := uc.dispatcher.Dispatch(dispatchCtx, event)

	// Step 6: Stop heartbeat and check ownership.
	heartbeatErr := stopHeartbeat()

	if heartbeatErr != nil {
		// Do not update event state with a potentially stale token.
		lg.Error("outbox.ownership_lost",
			"note", "event state not updated; it will be reclaimed after its lease expires",
			"dispatch_succeeded", dispatchErr == nil,
			notiflog.Err(heartbeatErr),
		)
		return fmt.Errorf(
			"outbox processing ownership or heartbeat failure: %w",
			heartbeatErr,
		)
	}

	// Step 7: Handle dispatch failure.
	if dispatchErr != nil {

		// Shutdown interrupted the dispatch. This is not a real failure,
		// so do not consume an attempt; the event is reclaimed after its
		// lease expires.
		if ctx.Err() != nil {
			lg.Warn("outbox.event_interrupted",
				"note", "shutdown; attempt not counted, event reclaimed after lease",
				notiflog.Err(dispatchErr),
			)
			return fmt.Errorf(
				"outbox event %s processing interrupted: %w",
				event.ID,
				ctx.Err(),
			)
		}

		return uc.handleDispatchFailure(ctx, event, dispatchErr, start)
	}

	// Step 8: Mark the event processed.
	//
	// The dispatch already succeeded, so record it even if shutdown
	// started meanwhile; otherwise the event would be dispatched again
	// after its lease expires. A short detached context bounds the write.
	finalizeCtx, cancelFinalize := context.WithTimeout(
		context.WithoutCancel(ctx),
		5*time.Second,
	)
	defer cancelFinalize()

	// The repository verifies the processing token before updating.
	if err := uc.outboxRepository.MarkProcessed(
		finalizeCtx,
		event.ID,
		event.ProcessingToken,
	); err != nil {
		lg.Error("outbox.mark_processed_failed",
			"note", "dispatch succeeded; event will be re-dispatched after lease (deduplicated by task ID)",
			notiflog.Err(err),
		)
		return fmt.Errorf("mark outbox event processed: %w", err)
	}

	lg.Info("outbox.event_processed", notiflog.Since(start))

	return nil
}

// handleDispatchFailure records a failed attempt. Once MaxOutboxAttempts
// is reached the event moves to the terminal FAILED state; otherwise it
// returns to PENDING with an exponential backoff.
//
// Both updates are guarded by the processing token, so they only apply
// if this worker still owns the event.
func (uc *ProcessOutboxEventUseCase) handleDispatchFailure(
	ctx context.Context,
	event *notification.NotificationOutboxEvent,
	dispatchErr error,
	start time.Time,
) error {

	lg := notiflog.From(ctx)
	attempt := event.Attempts + 1

	if attempt >= MaxOutboxAttempts {
		if err := uc.outboxRepository.MarkDead(
			ctx,
			event.ID,
			event.ProcessingToken,
			dispatchErr.Error(),
		); err != nil {
			lg.Error("outbox.mark_dead_failed", notiflog.Err(err))
			return fmt.Errorf(
				"dispatch failed: %v; mark event dead: %w",
				dispatchErr,
				err,
			)
		}

		lg.Error("outbox.event_dead",
			"note", "terminal FAILED; needs operator attention",
			"attempts", attempt,
			notiflog.Err(dispatchErr),
			notiflog.Since(start),
		)

		return fmt.Errorf(
			"outbox event %s failed permanently after %d attempts: %w",
			event.ID,
			attempt,
			dispatchErr,
		)
	}

	retryDelay := calculateOutboxRetryDelay(attempt)
	nextRetryAt := time.Now().UTC().Add(retryDelay)

	if err := uc.outboxRepository.MarkFailed(
		ctx,
		event.ID,
		event.ProcessingToken,
		dispatchErr.Error(),
		nextRetryAt,
	); err != nil {
		lg.Error("outbox.mark_failed_failed", notiflog.Err(err))
		return fmt.Errorf(
			"dispatch failed: %v; mark event failed: %w",
			dispatchErr,
			err,
		)
	}

	lg.Warn("outbox.event_retry_scheduled",
		notiflog.KeyRetryIn, retryDelay.String(),
		notiflog.KeyNextRetryAt, nextRetryAt,
		"attempts_left", MaxOutboxAttempts-attempt,
		notiflog.Err(dispatchErr),
		notiflog.Since(start),
	)

	return fmt.Errorf("dispatch outbox event: %w", dispatchErr)
}

// calculateOutboxRetryDelay returns an exponential retry delay
// (10s, 20s, 40s, ...) capped at 15 minutes.
func calculateOutboxRetryDelay(attempt int) time.Duration {

	const (
		baseDelay = 10 * time.Second
		maxDelay  = 15 * time.Minute
		maxPower  = 10
	)

	if attempt < 1 {
		attempt = 1
	}

	power := attempt - 1
	if power > maxPower {
		power = maxPower
	}

	delay := baseDelay * time.Duration(1<<power)

	if delay > maxDelay {
		return maxDelay
	}

	return delay
}
