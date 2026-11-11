package postgres

import (
	"context"
	"database/sql"
	"errors"

	domainorder "github.com/vishalyadav0987/authentication/internal/domain/order"
	domainpayment "github.com/vishalyadav0987/authentication/internal/domain/payment"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) CreatePayment(
	ctx context.Context,
	payment *domainpayment.Payment,
) error {
	return r.Create(ctx, payment)
}

func (r *PaymentRepository) GetPaymentByID(
	ctx context.Context,
	paymentID string,
) (*domainpayment.Payment, error) {
	return r.GetByPaymentID(ctx, paymentID)
}

func (r *PaymentRepository) GetByCashFreeOrderID(
	ctx context.Context,
	cashFreeOrderID string,
) (*domainpayment.Payment, error) {
	return r.GetByCashfreeOrderID(ctx, cashFreeOrderID)
}

func (r *PaymentRepository) UpdatePaymentStatus(
	ctx context.Context,
	paymentID string,
	status domainpayment.PaymentStatus,
) error {
	return r.UpdateStatus(ctx, paymentID, status)
}

func (r *PaymentRepository) Create(
	ctx context.Context,
	payment *domainpayment.Payment,
) error {

	query := `
		INSERT INTO payments (
			payment_id,
			order_id,
			user_id,
			idempotency_key,
			cashfree_order_id,
			cashfree_payment_id,
			payment_session_id,
			amount_paise,
			currency,
			status,
			created_at,
			updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12
		)
		RETURNING id
	`
	return r.db.QueryRowContext(
		ctx,
		query,
		payment.PaymentID,
		payment.OrderID,
		payment.UserID,
		payment.IdempotencyKey,
		payment.CashfreeOrderID,
		payment.CashfreePaymentID,
		payment.PaymentSessionID,
		payment.AmountPaise,
		payment.Currency,
		payment.Status,
		payment.CreatedAt,
		payment.UpdatedAt,
	).Scan(&payment.ID)
}

func (r *PaymentRepository) GetByPaymentID(
	ctx context.Context,
	paymentID string,
) (*domainpayment.Payment, error) {
	query := `
		SELECT
			id,
			payment_id,
			order_id,
			user_id,
			idempotency_key,
			cashfree_order_id,
			cashfree_payment_id,
			payment_session_id,
			amount_paise,
			currency,
			status,
			created_at,
			updated_at
		FROM payments
		WHERE payment_id = $1
	`

	return r.getPayment(ctx, query, paymentID)
}

func (r *PaymentRepository) GetByOrderID(
	ctx context.Context,
	orderID string,
) (*domainpayment.Payment, error) {
	query := `
		SELECT
			id,
			payment_id,
			order_id,
			user_id,
			idempotency_key,
			cashfree_order_id,
			cashfree_payment_id,
			payment_session_id,
			amount_paise,
			currency,
			status,
			created_at,
			updated_at
		FROM payments
		WHERE order_id = $1
	`

	return r.getPayment(ctx, query, orderID)
}

func (r *PaymentRepository) GetByCashfreeOrderID(
	ctx context.Context,
	cashfreeOrderID string,
) (*domainpayment.Payment, error) {
	query := `
		SELECT
			id,
			payment_id,
			order_id,
			user_id,
			idempotency_key,
			cashfree_order_id,
			cashfree_payment_id,
			payment_session_id,
			amount_paise,
			currency,
			status,
			created_at,
			updated_at
		FROM payments
		WHERE cashfree_order_id = $1
	`

	return r.getPayment(ctx, query, cashfreeOrderID)
}

func (r *PaymentRepository) GetByIdempotencyKey(
	ctx context.Context,
	userID string,
	idempotencyKey string,
) (*domainpayment.Payment, error) {
	query := `
		SELECT
			id,
			payment_id,
			order_id,
			user_id,
			idempotency_key,
			cashfree_order_id,
			cashfree_payment_id,
			payment_session_id,
			amount_paise,
			currency,
			status,
			created_at,
			updated_at
		FROM payments
		WHERE user_id = $1
		  AND idempotency_key = $2
	`

	return r.getPayment(ctx, query, userID, idempotencyKey)
}

func (r *PaymentRepository) UpdateStatus(
	ctx context.Context,
	paymentID string,
	status domainpayment.PaymentStatus,
) error {
	query := `
		UPDATE payments
		SET status = $1,
		    updated_at = NOW()
		WHERE payment_id = $2
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		status,
		paymentID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domainpayment.ErrPaymentNotFound
	}

	return nil
}

func (r *PaymentRepository) getPayment(
	ctx context.Context,
	query string,
	args ...any,
) (*domainpayment.Payment, error) {
	var p domainpayment.Payment

	err := r.db.QueryRowContext(
		ctx,
		query,
		args...,
	).Scan(
		&p.ID,
		&p.PaymentID,
		&p.OrderID,
		&p.UserID,
		&p.IdempotencyKey,
		&p.CashfreeOrderID,
		&p.CashfreePaymentID,
		&p.PaymentSessionID,
		&p.AmountPaise,
		&p.Currency,
		&p.Status,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainpayment.ErrPaymentNotFound
		}

		return nil, err
	}

	return &p, nil
}

func (r *PaymentRepository) UpdateFromWebhook(
	ctx context.Context,
	paymentID string,
	cashfreePaymentID string,
	status domainpayment.PaymentStatus,
) error {
	query := `
		UPDATE payments
		SET
			cashfree_payment_id = $1,
			status = $2,
			updated_at = NOW()
		WHERE payment_id = $3
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		cashfreePaymentID,
		status,
		paymentID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domainpayment.ErrPaymentNotFound
	}

	return nil
}

func (r *PaymentRepository) ApplyWebhook(
	ctx context.Context,
	settlement *domainpayment.WebhookSettlement,
) error {
	if settlement == nil {
		return domainpayment.ErrInvalidWebhook
	}

	if settlement.PaymentID == "" ||
		settlement.CashfreePaymentID == "" ||
		settlement.EventID == "" ||
		settlement.EventType == "" ||
		len(settlement.EventPayload) == 0 {
		return domainpayment.ErrInvalidWebhook
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Lock payment row.
	var (
		orderID       string
		paymentStatus domainpayment.PaymentStatus
		amountPaise   int64
		currency      string
	)

	err = tx.QueryRowContext(
		ctx,
		`
		SELECT order_id, status, amount_paise, currency
		FROM payments
		WHERE payment_id = $1
		FOR UPDATE
		`,
		settlement.PaymentID,
	).Scan(
		&orderID,
		&paymentStatus,
		&amountPaise,
		&currency,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return domainpayment.ErrPaymentNotFound
	}
	if err != nil {
		return err
	}

	// 2. Lock associated order row.
	var orderStatus domainorder.OrderStatus

	err = tx.QueryRowContext(
		ctx,
		`
		SELECT status
		FROM orders
		WHERE order_id = $1
		FOR UPDATE
		`,
		orderID,
	).Scan(&orderStatus)

	if errors.Is(err, sql.ErrNoRows) {
		return domainorder.ErrOrderNotFound
	}
	if err != nil {
		return err
	}

	// 3. Handle duplicate and late webhook events.
	if paymentStatus == domainpayment.PaymentStatusSuccess {
		return tx.Commit()
	}

	if paymentStatus == domainpayment.PaymentStatusFailed ||
		paymentStatus == domainpayment.PaymentStatusCancelled {
		if settlement.Status != domainpayment.PaymentStatusSuccess {
			return tx.Commit()
		}
	}

	// 4. A successful payment can only settle a pending order.
	if settlement.Status == domainpayment.PaymentStatusSuccess &&
		orderStatus != domainorder.OrderStatusPending &&
		orderStatus != domainorder.OrderStatusProcessing {
		return domainorder.ErrInvalidOrderStatus
	}

	// 5. Update payment.
	result, err := tx.ExecContext(
		ctx,
		`
		UPDATE payments
		SET cashfree_payment_id = $1,
		    status = $2,
		    updated_at = NOW()
		WHERE payment_id = $3
		`,
		settlement.CashfreePaymentID,
		settlement.Status,
		settlement.PaymentID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domainpayment.ErrPaymentNotFound
	}

	// 6. On successful payment, move order to PROCESSING.
	if settlement.Status == domainpayment.PaymentStatusSuccess &&
		orderStatus == domainorder.OrderStatusPending {

		_, err = tx.ExecContext(
			ctx,
			`
			UPDATE orders
			SET status = $1,
			    updated_at = NOW()
			WHERE order_id = $2
			`,
			domainorder.OrderStatusProcessing,
			orderID,
		)
		if err != nil {
			return err
		}
	}

	// 7. Insert outbox event only for successful payment.
	if settlement.Status == domainpayment.PaymentStatusSuccess {
		_, err = tx.ExecContext(
			ctx,
			`
			INSERT INTO outbox_events (
				id,
				aggregate_id,
				event_type,
				payload,
				status,
				retry_count,
				created_at
			)
			VALUES ($1, $2, $3, $4, 'PENDING', 0, $5)
			ON CONFLICT (id) DO NOTHING
			`,
			settlement.EventID,
			orderID,
			settlement.EventType,
			settlement.EventPayload,
			settlement.EventCreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
