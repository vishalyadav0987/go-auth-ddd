package order

import (
	"context"
	"errors"
	"fmt"
	"log"
)

type OutboxRepository interface {
	FetchPending(
		ctx context.Context,
		limit int,
	) ([]OutboxEvent, error)

	MarkProcessed(
		ctx context.Context,
		eventID string,
	) error

	MarkFailed(
		ctx context.Context,
		eventID string,
	) error
}

type EventPublisher interface {
	Publish(
		ctx context.Context,
		event OutboxEvent,
	) error
}

type OutboxProcessor struct {
	repository OutboxRepository
	publisher  EventPublisher
	batchSize  int
}

func NewOutboxProcessor(
	repository OutboxRepository,
	publisher EventPublisher,
) *OutboxProcessor {
	return &OutboxProcessor{
		repository: repository,
		publisher:  publisher,
		batchSize:  100,
	}
}

func (p *OutboxProcessor) ProcessBatch(
	ctx context.Context,
) error {

	events, err := p.repository.FetchPending(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("fetch outbox events: %w", err)
	}

	var processingErrors []error

	for _, event := range events {
		if err := p.publisher.Publish(ctx, event); err != nil {
			log.Printf(
				"Failed to publish event %s: %v",
				event.ID,
				err,
			)

			if markErr := p.repository.MarkFailed(ctx, event.ID); markErr != nil {
				processingErrors = append(
					processingErrors,
					fmt.Errorf("mark event %s failed: %w", event.ID, markErr),
				)
			}

			processingErrors = append(
				processingErrors,
				fmt.Errorf("publish event %s: %w", event.ID, err),
			)

			continue
		}

		if err := p.repository.MarkProcessed(ctx, event.ID); err != nil {
			processingErrors = append(
				processingErrors,
				fmt.Errorf("mark event %s processed: %w", event.ID, err),
			)
		}
	}

	return errors.Join(processingErrors...)
}
