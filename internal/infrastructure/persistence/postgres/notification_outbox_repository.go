package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// notificationOutboxLease is how long a claimed notification event stays
// owned by a worker before another worker may reclaim it.
const notificationOutboxLease = "5 minutes"

// sqlExecer is satisfied by both *sql.DB and *sql.Tx.
type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// NotificationOutboxRepository is the PostgreSQL implementation of the
// notification outbox, backed by the notification_outbox_events table.
type NotificationOutboxRepository struct {
	db *sql.DB
}

// NewNotificationOutboxRepository creates the notification outbox repository.
func NewNotificationOutboxRepository(db *sql.DB) *NotificationOutboxRepository {
	return &NotificationOutboxRepository{db: db}
}

// Create stores a new PENDING notification outbox event.
func (r *NotificationOutboxRepository) Create(
	ctx context.Context,
	event *domain.NotificationOutboxEvent,
) error {
	return insertNotificationOutboxEvent(ctx, r.db, event)
}

// insertNotificationOutboxEvent inserts a PENDING event using the given
// executor. Pass a *sql.Tx to write the event atomically with other
// changes (see NotificationRepository.PrepareDeadLetterRetry).
//
// Missing ID / timestamps are filled in on the event.
func insertNotificationOutboxEvent(
	ctx context.Context,
	exec sqlExecer,
	event *domain.NotificationOutboxEvent,
) error {

	if event == nil {
		return fmt.Errorf("notification outbox event is required")
	}

	if event.EventType == "" || event.AggregateID == "" {
		return fmt.Errorf("notification outbox event type and aggregate ID are required")
	}

	if len(event.Payload) == 0 {
		return fmt.Errorf("notification outbox event payload is required")
	}

	now := time.Now().UTC()

	if event.ID == "" {
		event.ID = uuid.NewString()
	}

	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}

	event.UpdatedAt = now
	event.Status = domain.OutboxPending
	event.Attempts = 0

	nextRetryAt := now
	if event.NextRetryAt != nil {
		nextRetryAt = event.NextRetryAt.UTC()
	}
	event.NextRetryAt = &nextRetryAt

	_, err := exec.ExecContext(
		ctx,
		`
		INSERT INTO notification_outbox_events (
			id, event_type, aggregate_id, payload,
			status, attempts, next_retry_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'PENDING', 0, $5, $6, $7)
		`,
		event.ID,
		event.EventType,
		event.AggregateID,
		event.Payload,
		nextRetryAt,
		event.CreatedAt,
		event.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert notification outbox event: %w", err)
	}

	return nil
}

// ClaimPending selects and claims due notification events in one
// transaction, using FOR UPDATE SKIP LOCKED so concurrent workers never
// claim the same row.
//
// Eligible events are PENDING events whose next_retry_at has passed and
// PROCESSING events whose lease expired (the worker crashed).
func (r *NotificationOutboxRepository) ClaimPending(
	ctx context.Context,
	limit int,
) ([]*domain.NotificationOutboxEvent, error) {

	if limit <= 0 {
		return nil, fmt.Errorf("notification outbox claim limit must be greater than zero")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin notification outbox claim transaction: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(
		ctx,
		`
		SELECT
			id, event_type, aggregate_id, payload,
			attempts, next_retry_at, COALESCE(last_error, ''),
			created_at
		FROM notification_outbox_events
		WHERE (status = 'PENDING' AND next_retry_at <= NOW())
		   OR (
				status = 'PROCESSING'
				AND (
					processing_started_at IS NULL
					OR processing_started_at < NOW() - $2::INTERVAL
				)
		   )
		ORDER BY next_retry_at ASC, created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
		`,
		limit,
		notificationOutboxLease,
	)
	if err != nil {
		return nil, fmt.Errorf("select claimable notification outbox events: %w", err)
	}

	var events []*domain.NotificationOutboxEvent

	for rows.Next() {
		event := &domain.NotificationOutboxEvent{}

		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.AggregateID,
			&event.Payload,
			&event.Attempts,
			&event.NextRetryAt,
			&event.LastError,
			&event.CreatedAt,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan notification outbox event: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate notification outbox events: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close notification outbox rows: %w", err)
	}

	// The rows are locked by this transaction, so a plain UPDATE by id
	// cannot race with another worker.
	for _, event := range events {
		token := uuid.NewString()

		if err := tx.QueryRowContext(
			ctx,
			`
			UPDATE notification_outbox_events
			SET status = 'PROCESSING',
			    processing_started_at = NOW(),
			    processing_token = $2,
				attempts = attempts + 1,
			    updated_at = NOW()
			WHERE id = $1
			RETURNING updated_at
			`,
			event.ID,
			token,
		).Scan(&event.UpdatedAt); err != nil {
			return nil, fmt.Errorf("claim notification outbox event %s: %w", event.ID, err)
		}

		event.Status = domain.OutboxProcessing
		event.ProcessingToken = token
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit notification outbox claims: %w", err)
	}

	return events, nil
}

// MarkProcessed finalises an event, provided the caller still owns it.
func (r *NotificationOutboxRepository) MarkProcessed(
	ctx context.Context,
	eventID string,
	processingToken string,
) error {

	result, err := r.db.ExecContext(
		ctx,
		`
		UPDATE notification_outbox_events
		SET status = 'PROCESSED',
		    last_error = NULL,
		    processed_at = NOW(),
		    processing_started_at = NULL,
		    processing_token = NULL,
		    updated_at = NOW()
		WHERE id = $1
		  AND status = 'PROCESSING'
		  AND processing_token = $2
		`,
		eventID,
		processingToken,
	)
	if err != nil {
		return fmt.Errorf("mark notification outbox event processed: %w", err)
	}

	return requireOneRow(result, eventID, domain.ErrOutboxLeaseLost)
}

// MarkFailed records a failed attempt and reschedules the event.
func (r *NotificationOutboxRepository) MarkFailed(
	ctx context.Context,
	eventID string,
	processingToken string,
	reason string,
	nextRetryAt time.Time,
) error {

	result, err := r.db.ExecContext(
		ctx,
		`
		UPDATE notification_outbox_events
		SET status = 'PENDING',
		    last_error = $3,
		    next_retry_at = $4,
		    processing_started_at = NULL,
		    processing_token = NULL,
		    updated_at = NOW()
		WHERE id = $1
		  AND status = 'PROCESSING'
		  AND processing_token = $2
		`,
		eventID,
		processingToken,
		reason,
		nextRetryAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("mark notification outbox event failed: %w", err)
	}

	return requireOneRow(result, eventID, domain.ErrOutboxLeaseLost)
}

// MarkDead records the final failed attempt and moves the event to the
// terminal FAILED state. It will not be claimed again.
func (r *NotificationOutboxRepository) MarkDead(
	ctx context.Context,
	eventID string,
	processingToken string,
	reason string,
) error {

	result, err := r.db.ExecContext(
		ctx,
		`
		UPDATE notification_outbox_events
		SET status = 'FAILED',
		    last_error = $3,
		    processing_started_at = NULL,
		    processing_token = NULL,
		    updated_at = NOW()
		WHERE id = $1
		  AND status = 'PROCESSING'
		  AND processing_token = $2
		`,
		eventID,
		processingToken,
		reason,
	)
	if err != nil {
		return fmt.Errorf("mark notification outbox event dead: %w", err)
	}

	return requireOneRow(result, eventID, domain.ErrOutboxLeaseLost)
}

// Heartbeat renews the processing lease for an outbox event.
//
// The processing token ensures that only the worker currently owning
// the event can renew its lease.
//
// Returns false when the worker has lost ownership.
func (r *NotificationOutboxRepository) Heartbeat(
	ctx context.Context,
	eventID string,
	processingToken string,
) (bool, error) {

	query := `
		UPDATE notification_outbox_events
	SET processing_started_at = NOW(),
		updated_at = NOW()
	WHERE id = $1
	AND status = 'PROCESSING'
	AND processing_token = $2
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		eventID,
		processingToken,
	)
	if err != nil {
		return false, fmt.Errorf(
			"renew outbox processing lease: %w",
			err,
		)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"check outbox heartbeat result: %w",
			err,
		)
	}

	return affected == 1, nil
}

var _ domain.NotificationOutboxRepository = (*NotificationOutboxRepository)(nil)
