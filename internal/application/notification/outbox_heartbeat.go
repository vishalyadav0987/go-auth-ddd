package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vishalyadav0987/authentication/pkg/notiflog"
)

// OutboxHeartbeatRepository renews an outbox event's processing lease.
type OutboxHeartbeatRepository interface {
	Heartbeat(
		ctx context.Context,
		eventID string,
		processingToken string,
	) (bool, error)
}

// OutboxHeartbeat maintains ownership of a claimed outbox event.
type OutboxHeartbeat struct {
	repository OutboxHeartbeatRepository
	interval   time.Duration
}

// NewOutboxHeartbeat creates the heartbeat manager.
func NewOutboxHeartbeat(
	repository OutboxHeartbeatRepository,
) *OutboxHeartbeat {
	return &OutboxHeartbeat{
		repository: repository,
		interval:   time.Minute,
	}
}

// Start begins renewing the event's processing lease.
//
// The returned context must be used for the event's processing work: it
// is cancelled as soon as ownership is lost or a renewal fails, so the
// worker stops acting on an event another worker may now own.
//
// The returned function stops the heartbeat and waits for its
// goroutine to finish. It returns an error if ownership was lost
// or if a heartbeat operation failed.
func (h *OutboxHeartbeat) Start(
	ctx context.Context,
	eventID string,
	processingToken string,
) (context.Context, func() error, error) {

	if h.repository == nil {
		return nil, nil, fmt.Errorf("outbox heartbeat repository is nil")
	}

	if eventID == "" || processingToken == "" {
		return nil, nil, fmt.Errorf("event ID and processing token are required")
	}

	if h.interval <= 0 {
		return nil, nil, fmt.Errorf("heartbeat interval must be positive")
	}

	heartbeatCtx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})

	var (
		mu            sync.Mutex
		heartbeatErr  error
		ownershipLost bool
		stopOnce      sync.Once
	)

	go func() {
		defer close(done)

		ticker := time.NewTicker(h.interval)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				return

			case <-ticker.C:

				owned, err := h.repository.Heartbeat(
					heartbeatCtx,
					eventID,
					processingToken,
				)

				if err != nil {
					notiflog.From(ctx).Error("outbox.heartbeat_failed", notiflog.Err(err))

					mu.Lock()
					heartbeatErr = err
					mu.Unlock()

					cancel()
					return
				}

				if !owned {
					notiflog.From(ctx).Warn("outbox.heartbeat_ownership_lost")

					mu.Lock()
					ownershipLost = true
					mu.Unlock()

					cancel()
					return
				}

				notiflog.From(ctx).Debug("outbox.heartbeat_renewed")
			}
		}
	}()

	stop := func() error {
		stopOnce.Do(cancel)
		<-done

		mu.Lock()
		defer mu.Unlock()

		if ownershipLost {
			return fmt.Errorf(
				"outbox event processing ownership lost: %s",
				eventID,
			)
		}

		if heartbeatErr != nil {
			return fmt.Errorf(
				"outbox heartbeat failed: %w",
				heartbeatErr,
			)
		}

		return nil
	}

	return heartbeatCtx, stop, nil
}
