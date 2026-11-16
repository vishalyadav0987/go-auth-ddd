// Now we'll implement the repository for tracking every email and push delivery attempt.

// This is important for production debugging because the current notification_deliveries table stores only the latest status and failure reason, whereas this table preserves the complete attempt history.

package postgres

import (
	"context"
	"database/sql"
	"fmt"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

type DeliveryAttemptRepository struct {
	db *sql.DB
}

func NewDeliveryAttemptRepository(
	db *sql.DB,
) *DeliveryAttemptRepository {
	return &DeliveryAttemptRepository{
		db: db,
	}
}

func (r *DeliveryAttemptRepository) CreateAttempt(
	ctx context.Context,
	attempt *domain.NotificationDeliveryAttempt,
) error {
	if attempt == nil {
		return fmt.Errorf("delivery attempt cannot be nil")
	}

	const query = `
		INSERT INTO notification_delivery_attempts (
			id,
			delivery_id,
			attempt_number,
			status,
			provider_id,
			error_message,
			attempted_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		attempt.ID,
		attempt.DeliveryID,
		attempt.AttemptNumber,
		attempt.Status,
		attempt.ProviderID,
		attempt.ErrorMessage,
		attempt.AttemptedAt,
	)
	if err != nil {
		return fmt.Errorf("create delivery attempt: %w", err)
	}

	return nil
}

func (r *DeliveryAttemptRepository) UpdateAttempt(
	ctx context.Context,
	deliveryID string,
	attemptNumber int,
	status string,
	providerID string,
	errorMessage string,
	processingToken string,
) error {
	if processingToken == "" {
		return fmt.Errorf("processing token is required")
	}

	result, err := r.db.ExecContext(
		ctx,
		`
        UPDATE notification_delivery_attempts
        SET status = $3,
            provider_id = $4,
            error_message = $5
        WHERE delivery_id = $1
          AND attempt_number = $2
          AND EXISTS (
              SELECT 1
              FROM notification_deliveries
              WHERE id = $1
                AND status = 'PROCESSING'
                AND processing_token = $6
          )
        `,
		deliveryID,
		attemptNumber,
		status,
		providerID,
		errorMessage,
		processingToken,
	)
	if err != nil {
		return fmt.Errorf("update delivery attempt: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated attempt rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf(
			"attempt not found or processing ownership lost: delivery=%s attempt=%d",
			deliveryID,
			attemptNumber,
		)
	}

	return nil
}

// GetDeliveryAttemptsByDeliveryID retrieves all attempts associated
// with a delivery, ordered by attempt number.
func (r *DeliveryAttemptRepository) GetDeliveryAttemptsByDeliveryID(
	ctx context.Context,
	deliveryID string,
) ([]*domain.NotificationDeliveryAttempt, error) {

	if deliveryID == "" {
		return nil, fmt.Errorf("delivery ID is required")
	}

	query := `
		SELECT
			id,
			delivery_id,
			attempt_number,
			status,
			COALESCE(provider_id, ''),
			COALESCE(error_message, ''),
		FROM notification_delivery_attempts
		WHERE delivery_id = $1
		ORDER BY attempt_number ASC
	`

	rows, err := r.db.QueryContext(ctx, query, deliveryID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve delivery attempts: %w", err)
	}
	defer rows.Close()

	attempts := make([]*domain.NotificationDeliveryAttempt, 0)

	for rows.Next() {
		attempt := &domain.NotificationDeliveryAttempt{}

		err := rows.Scan(
			&attempt.ID,
			&attempt.DeliveryID,
			&attempt.AttemptNumber,
			&attempt.Status,
			&attempt.ProviderID,
			&attempt.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delivery attempt: %w", err)
		}

		attempts = append(attempts, attempt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating delivery attempts: %w", err)
	}

	return attempts, nil
}

var _ domain.DeliveryAttemptRepository = (*DeliveryAttemptRepository)(nil)
