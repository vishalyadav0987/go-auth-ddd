package postgres

import (
	"context"
	"database/sql"
	"errors"

	domainorder "github.com/vishalyadav0987/authentication/internal/domain/order"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) GetByIdempotencyKey(
	ctx context.Context,
	userID string,
	idempotencyKey string,
) (*domainorder.Order, error) {

	query := `
		SELECT id, order_id, user_id, scheme_code,
		       amount_paise, investment_type, idempotency_key,
		       status, created_at, updated_at
		FROM orders
		WHERE user_id = $1 AND idempotency_key = $2
	`

	order := &domainorder.Order{}

	err := r.db.QueryRowContext(
		ctx, query, userID, idempotencyKey,
	).Scan(
		&order.ID,
		&order.OrderID,
		&order.UserID,
		&order.SchemeCode,
		&order.AmountPaise,
		&order.InvestmentType,
		&order.IdempotencyKey,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domainorder.ErrOrderNotFound
	}

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (r *OrderRepository) GetByOrderID(
	ctx context.Context,
	orderID string,
) (*domainorder.Order, error) {

	query := `
		SELECT id, order_id, user_id, scheme_code,
		       amount_paise, investment_type, idempotency_key,
		       status, created_at, updated_at
		FROM orders
		WHERE order_id = $1
	`

	order := &domainorder.Order{}

	err := r.db.QueryRowContext(ctx, query, orderID).Scan(
		&order.ID,
		&order.OrderID,
		&order.UserID,
		&order.SchemeCode,
		&order.AmountPaise,
		&order.InvestmentType,
		&order.IdempotencyKey,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domainorder.ErrOrderNotFound
	}

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (r *OrderRepository) Create(
	ctx context.Context,
	order *domainorder.Order,
) error {

	query := `
		INSERT INTO orders (
			order_id, user_id, scheme_code, amount_paise,
			investment_type, idempotency_key, status,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	return r.db.QueryRowContext(
		ctx,
		query,
		order.OrderID,
		order.UserID,
		order.SchemeCode,
		order.AmountPaise,
		order.InvestmentType,
		order.IdempotencyKey,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	).Scan(&order.ID)
}

func (r *OrderRepository) UpdateStatus(
	ctx context.Context,
	orderID string,
	status domainorder.OrderStatus,
) error {

	query := `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE order_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, orderID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return domainorder.ErrOrderNotFound
	}

	return nil
}

func (r *OrderRepository) CreateWithOutbox(
	ctx context.Context,
	order *domainorder.Order,
	event *domainorder.OutboxEvent,
) error {

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	orderQuery := `
		INSERT INTO orders (
			order_id, user_id, scheme_code, amount_paise,
			investment_type, idempotency_key, status,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	err = tx.QueryRowContext(
		ctx,
		orderQuery,
		order.OrderID,
		order.UserID,
		order.SchemeCode,
		order.AmountPaise,
		order.InvestmentType,
		order.IdempotencyKey,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	).Scan(&order.ID)

	if err != nil {
		return err
	}

	outboxQuery := `
		INSERT INTO outbox_events (
			id, aggregate_id, event_type, payload, created_at
		)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = tx.ExecContext(
		ctx,
		outboxQuery,
		event.ID,
		event.AggregateID,
		event.EventType,
		event.Payload,
		event.CreatedAt,
	)

	if err != nil {
		return err
	}

	return tx.Commit()
}
