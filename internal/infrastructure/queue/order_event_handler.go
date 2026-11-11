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
			"Processing order created event: OrderID=%s EventID=%s",
			event.AggregateID,
			event.EventID,
		)

	default:
		return fmt.Errorf("unsupported order event type: %s", event.EventType)
	}

	return nil
}
