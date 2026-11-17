// OutboxEventDispatcher routes notification outbox events to the handler
// registered for their event type. It contains no business logic itself:
// supporting a new event type means implementing a handler and
// registering it at startup, without changing the dispatcher.
//
// Order and payment events are not routed here. They live in the
// separate order outbox (outbox_events) and are published by the order
// outbox worker to the order event queue.
package notification

import (
	"context"
	"fmt"

	"github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// OutboxEventHandler handles one type of notification outbox event.
type OutboxEventHandler interface {
	Handle(
		ctx context.Context,
		event *notification.NotificationOutboxEvent,
	) error
}

// OutboxEventHandlerFunc adapts a plain function to OutboxEventHandler.
type OutboxEventHandlerFunc func(
	ctx context.Context,
	event *notification.NotificationOutboxEvent,
) error

// Handle calls f(ctx, event).
func (f OutboxEventHandlerFunc) Handle(
	ctx context.Context,
	event *notification.NotificationOutboxEvent,
) error {
	return f(ctx, event)
}

// OutboxEventDispatcher is an in-memory registry of event handlers.
//
// Register every handler during application initialisation, before the
// outbox worker starts. Register is not safe to call concurrently with
// Dispatch.
type OutboxEventDispatcher struct {
	handlers map[string]OutboxEventHandler
}

var _ OutboxDispatcher = (*OutboxEventDispatcher)(nil)

// NewOutboxEventDispatcher creates a dispatcher with no handlers.
func NewOutboxEventDispatcher() *OutboxEventDispatcher {
	return &OutboxEventDispatcher{
		handlers: make(map[string]OutboxEventHandler),
	}
}

// Register adds the handler for an event type.
//
// Each event type has exactly one handler; registering a second one
// returns an error.
func (d *OutboxEventDispatcher) Register(
	eventType string,
	handler OutboxEventHandler,
) error {

	if eventType == "" {
		return fmt.Errorf("event type is required")
	}

	if handler == nil {
		return fmt.Errorf("handler is required for event type %s", eventType)
	}

	if _, exists := d.handlers[eventType]; exists {
		return fmt.Errorf("handler already registered for event type %s", eventType)
	}

	d.handlers[eventType] = handler

	return nil
}

// Dispatch delegates the event to the handler registered for its type.
//
// An unknown event type is returned as an error rather than silently
// dropped, so the processor retries it and eventually moves it to FAILED
// for operator attention.
func (d *OutboxEventDispatcher) Dispatch(
	ctx context.Context,
	event *notification.NotificationOutboxEvent,
) error {

	if event == nil {
		return fmt.Errorf("outbox event cannot be nil")
	}

	if event.ID == "" {
		return fmt.Errorf("outbox event ID cannot be empty")
	}

	if event.EventType == "" {
		return fmt.Errorf("outbox event %s has no event type", event.ID)
	}

	handler, exists := d.handlers[event.EventType]
	if !exists {
		return fmt.Errorf(
			"no handler registered for outbox event type %s",
			event.EventType,
		)
	}

	return handler.Handle(ctx, event)
}
