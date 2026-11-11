package order

import (
	"context"
	"log"
	"time"
)

type OutboxWorker struct {
	processor *OutboxProcessor
	interval  time.Duration
}

func NewOutboxWorker(
	processor *OutboxProcessor,
) *OutboxWorker {
	return &OutboxWorker{
		processor: processor,
		interval:  2 * time.Second,
	}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Println("Outbox worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Outbox worker stopped")
			return

		case <-ticker.C:
			if err := w.processor.ProcessBatch(ctx); err != nil {
				log.Printf("Outbox processing error: %v", err)
			}
		}
	}
}
