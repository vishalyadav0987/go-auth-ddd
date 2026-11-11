package postgresRate

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type MPINRateLimiter struct {
	db *sql.DB
}

// constructor
func NewMPINRateLimiter(db *sql.DB) *MPINRateLimiter {
	return &MPINRateLimiter{
		db: db,
	}
}

const (
	maxAttempts = 5
	blockTime   = 10 * time.Minute
)

// Allow implements rate limiter interface
func (r *MPINRateLimiter) Allow(ctx context.Context, userID, otpAccessToken string) error {
	var blockUntil sql.NullTime

	query := `SELECT blocked_until FROM mpin_attempts WHERE user_id = $1 AND otp_access_token = $2`

	err := r.db.QueryRowContext(ctx, query, userID, otpAccessToken).Scan(&blockUntil)

	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return err
	}

	if blockUntil.Valid && blockUntil.Time.After(time.Now()) {
		return errors.New("too many attempts, try again later")
	}

	return nil
}

func (r *MPINRateLimiter) RegisterFailure(ctx context.Context, userID, otpAccessToken string) error {
	var attempts int

	query1 := `SELECT attempts FROM mpin_attempts WHERE user_id = $1 AND otp_access_token = $2`

	err := r.db.QueryRowContext(ctx, query1, userID, otpAccessToken).Scan(&attempts)

	now := time.Now().UTC()

	if errors.Is(err, sql.ErrNoRows) {
		query2 := `INSERT INTO mpin_attempts (user_id, otp_access_token, attempts, updated_at) VALUES ($1, $2, 1, $3)`
		_, err := r.db.ExecContext(ctx, query2, userID, otpAccessToken, now)
		return err
	}

	if err != nil {
		return err
	}

	attempts++

	var blockedUntil *time.Time

	if attempts >= maxAttempts {
		t := now.Add(blockTime)
		blockedUntil = &t
	}

	query3 := `
		UPDATE mpin_attempts
		SET attempts = $1, blocked_until = $2, updated_at = $3
		WHERE user_id = $4 AND otp_access_token = $5
	`

	_, err = r.db.ExecContext(ctx, query3, attempts, blockedUntil, now, userID, otpAccessToken)
	return err
}

func (r *MPINRateLimiter) Reset(ctx context.Context, userID, otpAccessToken string) error {
	query := `DELETE FROM mpin_attempts WHERE user_id = $1 AND otp_access_token = $2`
	_, err := r.db.ExecContext(ctx, query, userID, otpAccessToken)
	return err
}
