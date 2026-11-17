package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	domainorder "github.com/vishalyadav0987/authentication/internal/domain/order"
)

// orderOutboxLease is how long a claimed order event stays owned by a
// worker before another worker may reclaim it.
const orderOutboxLease = "5 minutes"

// OrderOutboxRepository is the PostgreSQL implementation of the order
// outbox, backed by the outbox_events table.
type OrderOutboxRepository struct {
	db *sql.DB
}

var _ domainorder.OrderOutboxRepository = (*OrderOutboxRepository)(nil)

// NewOrderOutboxRepository creates the order outbox repository.
func NewOrderOutboxRepository(db *sql.DB) *OrderOutboxRepository {
	return &OrderOutboxRepository{db: db}
}

// ClaimPending selects and claims order outbox events in one transaction.
//
// FOR UPDATE SKIP LOCKED ensures concurrent workers never select the same
// row. PROCESSING rows whose lease has expired (crashed worker) are
// reclaimed with a new token, so the stale worker can no longer update them.
func (r *OrderOutboxRepository) ClaimPending(
	ctx context.Context,
	limit int,
) ([]domainorder.OutboxEvent, error) {

	if limit <= 0 {
		return nil, fmt.Errorf("order outbox claim limit must be greater than zero")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin order outbox claim transaction: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(
		ctx,
		`
		SELECT id, aggregate_id, event_type, payload, created_at
		FROM outbox_events
		WHERE status = 'PENDING'
		   OR (
				status = 'PROCESSING'
				AND processing_started_at < NOW() - $2::INTERVAL
		   )
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
		`,
		limit,
		orderOutboxLease,
	)
	if err != nil {
		return nil, fmt.Errorf("select claimable order outbox events: %w", err)
	}

	var events []domainorder.OutboxEvent

	for rows.Next() {
		var event domainorder.OutboxEvent

		if err := rows.Scan(
			&event.ID,
			&event.AggregateID,
			&event.EventType,
			&event.Payload,
			&event.CreatedAt,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan order outbox event: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate order outbox events: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close order outbox rows: %w", err)
	}

	// The rows are locked by this transaction, so a plain UPDATE by id
	// cannot race with another worker.
	for i := range events {
		token := uuid.NewString()

		if _, err := tx.ExecContext(
			ctx,
			`
			UPDATE outbox_events
			SET status = 'PROCESSING',
			    processing_started_at = NOW(),
			    processing_token = $2
			WHERE id = $1
			`,
			events[i].ID,
			token,
		); err != nil {
			return nil, fmt.Errorf("claim order outbox event %s: %w", events[i].ID, err)
		}

		events[i].ProcessingToken = token
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit order outbox claims: %w", err)
	}

	return events, nil
}

// MarkProcessed finalises an event, provided the caller still owns it.
func (r *OrderOutboxRepository) MarkProcessed(
	ctx context.Context,
	eventID string,
	processingToken string,
) error {

	result, err := r.db.ExecContext(
		ctx,
		`
		UPDATE outbox_events
		SET status = 'PROCESSED',
		    processed_at = NOW(),
		    processing_started_at = NULL,
		    processing_token = NULL
		WHERE id = $1
		  AND status = 'PROCESSING'
		  AND processing_token = $2
		`,
		eventID,
		processingToken,
	)
	if err != nil {
		return fmt.Errorf("mark order outbox event processed: %w", err)
	}

	return requireOneRow(result, eventID, domainorder.ErrOutboxLeaseLost)
}

// MarkFailed returns an event to PENDING and increments retry_count,
// provided the caller still owns it.
func (r *OrderOutboxRepository) MarkFailed(
	ctx context.Context,
	eventID string,
	processingToken string,
) error {

	result, err := r.db.ExecContext(
		ctx,
		`
		UPDATE outbox_events
		SET status = 'PENDING',
		    retry_count = retry_count + 1,
		    processing_started_at = NULL,
		    processing_token = NULL
		WHERE id = $1
		  AND status = 'PROCESSING'
		  AND processing_token = $2
		`,
		eventID,
		processingToken,
	)
	if err != nil {
		return fmt.Errorf("mark order outbox event failed: %w", err)
	}

	return requireOneRow(result, eventID, domainorder.ErrOutboxLeaseLost)
}

// requireOneRow returns leaseErr (wrapped with the event ID) when a
// token-guarded update did not touch exactly one row.
func requireOneRow(result sql.Result, eventID string, leaseErr error) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected for outbox event %s: %w", eventID, err)
	}

	if affected != 1 {
		return fmt.Errorf("outbox event %s: %w", eventID, leaseErr)
	}

	return nil
}
