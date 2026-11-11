package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hibiken/asynq"

	apporder "github.com/vishalyadav0987/authentication/internal/application/order"
)

const TaskTypeOrderEvent = "order:event"

type OrderEventPayload struct {
	EventID     string          `json:"event_id"`
	AggregateID string          `json:"aggregate_id"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
}

type OrderEventPublisher struct {
	client *asynq.Client
}

func NewOrderEventPublisher(
	client *asynq.Client,
) *OrderEventPublisher {
	return &OrderEventPublisher{
		client: client,
	}
}

func (p *OrderEventPublisher) Publish(
	ctx context.Context,
	event apporder.OutboxEvent,
) error {

	payload := OrderEventPayload{
		EventID:     event.ID,
		AggregateID: event.AggregateID,
		EventType:   event.EventType,
		Payload:     json.RawMessage(event.Payload),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal order event: %w", err)
	}

	task := asynq.NewTask(
		TaskTypeOrderEvent,
		payloadBytes,
		asynq.TaskID(event.ID),
		asynq.Queue("critical"),
	)

	_, err = p.client.EnqueueContext(ctx, task)
	if err != nil {
		if errors.Is(err, asynq.ErrTaskIDConflict) {
			return nil
		}

		return fmt.Errorf("enqueue order event: %w", err)
	}

	return nil
}
