package postgres

import (
	"context"
	"database/sql"

	apporder "github.com/vishalyadav0987/authentication/internal/application/order"
)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) FetchPending(
	ctx context.Context,
	limit int,
) ([]apporder.OutboxEvent, error) {

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, aggregate_id, event_type, payload, created_at
		FROM outbox_events
		WHERE status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := tx.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	var events []apporder.OutboxEvent

	for rows.Next() {
		var event apporder.OutboxEvent

		if err := rows.Scan(
			&event.ID,
			&event.AggregateID,
			&event.EventType,
			&event.Payload,
			&event.CreatedAt,
		); err != nil {
			rows.Close()
			return nil, err
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}

	rows.Close()

	for _, event := range events {
		_, err := tx.ExecContext(
			ctx,
			`UPDATE outbox_events SET status = 'PROCESSING' WHERE id = $1`,
			event.ID,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *OutboxRepository) MarkProcessed(
	ctx context.Context,
	eventID string,
) error {

	query := `
		UPDATE outbox_events
		SET status = 'PROCESSED',
		    processed_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, eventID)
	return err
}

func (r *OutboxRepository) MarkFailed(
	ctx context.Context,
	eventID string,
) error {

	query := `
		UPDATE outbox_events
		SET status = 'PENDING',
		    retry_count = retry_count + 1
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, eventID)
	return err
}
