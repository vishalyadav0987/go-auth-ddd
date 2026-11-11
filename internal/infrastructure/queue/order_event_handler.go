package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

type OrderEventHandler struct {
}

func NewOrderEventHandler() *OrderEventHandler {
	return &OrderEventHandler{}
}

func (h *OrderEventHandler) HandleOrderEvent(
	ctx context.Context,
	task *asynq.Task,
) error {

	var event OrderEventPayload

	if err := json.Unmarshal(task.Payload(), &event); err != nil {
		return fmt.Errorf("decode order event: %w", err)
	}

	if event.EventID == "" || event.AggregateID == "" {
		return fmt.Errorf("invalid order event payload")
	}

	switch event.EventType {

	case "ORDER_CREATED":

		log.Printf(
			"Processing ORDER_CREATED event: OrderID=%s EventID=%s",
			event.AggregateID,
			event.EventID,
		)

		return h.handleOrderCreated(ctx, event)

	case "PAYMENT_SUCCEEDED":

		log.Printf(
			"Processing PAYMENT_SUCCEEDED event: OrderID=%s EventID=%s",
			event.AggregateID,
			event.EventID,
		)

		return h.handlePaymentSucceeded(ctx, event)

	default:

		return fmt.Errorf(
			"unsupported order event type: %s",
			event.EventType,
		)
	}
}

func (h *OrderEventHandler) handleOrderCreated(
	ctx context.Context,
	event OrderEventPayload,
) error {

	log.Printf(
		"Order created event received successfully: %s",
		event.AggregateID,
	)

	return nil
}

func (h *OrderEventHandler) handlePaymentSucceeded(
	ctx context.Context,
	event OrderEventPayload,
) error {

	log.Printf(
		"Payment succeeded event received successfully: OrderID=%s Payload=%s",
		event.AggregateID,
		string(event.Payload),
	)

	// Investment processing will be connected here.

	return nil
}
