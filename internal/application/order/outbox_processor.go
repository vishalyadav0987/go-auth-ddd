package order

import (
	"context"
	"errors"
	"fmt"
	"log"

	domainorder "github.com/vishalyadav0987/authentication/internal/domain/order"
)

type EventPublisher interface {
	Publish(
		ctx context.Context,
		event OutboxEvent,
	) error
}

type OutboxProcessor struct {
	repository domainorder.OrderOutboxRepository
	publisher  EventPublisher
	batchSize  int
}

func NewOutboxProcessor(
	repository domainorder.OrderOutboxRepository,
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

	events, err := p.repository.ClaimPending(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("fetch outbox events: %w", err)
	}

	var processingErrors []error

	for _, event := range events {
		// Use the token issued by ClaimPending; it proves this worker
		// still owns the event.
		token := event.ProcessingToken

		if err := p.publisher.Publish(ctx, event); err != nil {
			log.Printf(
				"Failed to publish event %s: %v",
				event.ID,
				err,
			)

			if markErr := p.repository.MarkFailed(ctx, event.ID, token); markErr != nil {
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

		if err := p.repository.MarkProcessed(ctx, event.ID, token); err != nil {
			processingErrors = append(
				processingErrors,
				fmt.Errorf("mark event %s processed: %w", event.ID, err),
			)
		}
	}

	return errors.Join(processingErrors...)
}
