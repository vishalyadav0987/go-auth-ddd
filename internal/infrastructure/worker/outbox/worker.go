package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/pkg/notiflog"
)

// NotificationOutboxClaimer claims due events from the notification outbox
// (notification_outbox_events). Order events live in a separate table and
// are handled by the order outbox worker.
type NotificationOutboxClaimer interface {
	ClaimPending(
		ctx context.Context,
		limit int,
	) ([]*notification.NotificationOutboxEvent, error)
}

// NotificationOutboxEventProcessor processes one claimed event.
//
// ProcessOutboxEventUseCase implements this interface.
type NotificationOutboxEventProcessor interface {
	Execute(
		ctx context.Context,
		event *notification.NotificationOutboxEvent,
	) error
}

// Worker continuously polls the notification outbox and processes events.
type Worker struct {
	claimer   NotificationOutboxClaimer
	processor NotificationOutboxEventProcessor

	batchSize    int
	pollInterval time.Duration
}

// NewWorker creates a notification outbox polling worker.
func NewWorker(
	claimer NotificationOutboxClaimer,
	processor NotificationOutboxEventProcessor,
	batchSize int,
	pollInterval time.Duration,
) (*Worker, error) {

	if claimer == nil {
		return nil, fmt.Errorf("outbox claimer is required")
	}

	if processor == nil {
		return nil, fmt.Errorf("outbox event processor is required")
	}

	if batchSize <= 0 {
		return nil, fmt.Errorf("batch size must be greater than zero")
	}

	if pollInterval <= 0 {
		return nil, fmt.Errorf("poll interval must be greater than zero")
	}

	return &Worker{
		claimer:      claimer,
		processor:    processor,
		batchSize:    batchSize,
		pollInterval: pollInterval,
	}, nil
}

// Run starts the polling loop and blocks until ctx is cancelled.
//
// Individual event failures do not stop the worker. While events are
// available it drains them back-to-back; it only sleeps when the outbox
// is empty or the database is failing.
func (w *Worker) Run(ctx context.Context) error {

	lg := notiflog.L().With("component", "outbox_worker")

	lg.Info("outbox.worker_started",
		"batch_size", w.batchSize,
		"poll_interval", w.pollInterval.String(),
	)
	defer lg.Info("outbox.worker_stopped")

	for {
		// Stop before starting another database operation on shutdown.
		if ctx.Err() != nil {
			return nil
		}

		// ClaimPending moves events to PROCESSING and assigns each a
		// processing token.
		events, err := w.claimer.ClaimPending(ctx, w.batchSize)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			lg.Error("outbox.claim_failed",
				"retry_in", w.pollInterval.String(),
				notiflog.Err(err),
			)

			// Avoid hammering PostgreSQL while it is unavailable.
			if !wait(ctx, w.pollInterval) {
				return nil
			}

			continue
		}

		// Nothing is due. Sleep before polling again.
		if len(events) == 0 {
			if !wait(ctx, w.pollInterval) {
				return nil
			}

			continue
		}

		lg.Info("outbox.events_claimed", "count", len(events))

		// Process sequentially for now; controlled concurrency can be
		// added once the full flow is validated.
		for _, event := range events {

			// Do not start another event during shutdown. Events claimed
			// but not started are reclaimed once their lease expires.
			if ctx.Err() != nil {
				return nil
			}

			if err := w.processor.Execute(ctx, event); err != nil {

				// Shutdown caused the error: leave the event for
				// lease-based recovery.
				if ctx.Err() != nil {
					return nil
				}

				// The processor already logged the details and recorded
				// the failure (retry scheduled / dead / lease lost).
				lg.Debug("outbox.event_failed",
					notiflog.KeyOutboxEventID, event.ID,
					notiflog.KeyEventType, event.EventType,
				)

				continue
			}

		}

		// A batch was processed; poll again immediately.
	}
}

// wait pauses the worker without blocking shutdown.
//
// It returns false when ctx is cancelled.
func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
