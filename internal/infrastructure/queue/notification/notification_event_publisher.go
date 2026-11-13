package notificationQueue

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

type Publisher struct {
	client *asynq.Client
}

func NewPublisher(client *asynq.Client) *Publisher {
	return &Publisher{
		client: client,
	}
}

func (p *Publisher) PublishDelivery(
	ctx context.Context,
	delivery *domain.NotificationDelivery,
	priority domain.NotificationPriority,
) error {
	if delivery == nil {
		return fmt.Errorf("notification delivery is required")
	}

	if delivery.ID == "" || delivery.NotificationID == "" ||
		delivery.UserID == "" {
		return fmt.Errorf("invalid notification delivery")
	}

	queue := priority.QueueName()
	if queue == "" {
		return domain.ErrInvalidPriority
	}

	payload, err := MarshalDeliveryPayload(DeliveryTaskPayload{
		DeliveryID:     delivery.ID,
		NotificationID: delivery.NotificationID,
		UserID:         delivery.UserID,
		Channel:        string(delivery.Channel),
	})
	if err != nil {
		return fmt.Errorf("marshal delivery payload: %w", err)
	}

	taskType := TaskTypeSendEmail
	if delivery.Channel == domain.ChannelPush {
		taskType = TaskTypeSendPush
	} else if delivery.Channel != domain.ChannelEmail {
		return domain.ErrInvalidChannel
	}

	task := asynq.NewTask(
		taskType,
		payload,
		asynq.Queue(queue),
		asynq.TaskID(delivery.ID),
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
	)

	_, err = p.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("enqueue notification delivery: %w", err)
	}

	return nil
}
