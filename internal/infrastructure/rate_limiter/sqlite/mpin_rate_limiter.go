package sqliteRate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// implements to rate limiter interface
func (r *MPINRateLimiter) Allow(ctx context.Context, userID, otpAccessToken string) error {
	// check block / attempts
	var blockUntil sql.NullTime

	query := `SELECT blocked_until FROM mpin_attempts WHERE user_id = ? AND otp_access_token = ?`

	err := r.db.QueryRowContext(ctx, query, userID, otpAccessToken).Scan(&blockUntil)

	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	fmt.Println("working")

	if err != nil {
		return err
	}

	if blockUntil.Valid && blockUntil.Time.After(time.Now()) {
		return errors.New("too many attempts, try again later")
	}

	return nil

}

func (r *MPINRateLimiter) RegisterFailure(ctx context.Context, userID, otpAccessToken string) error {
	// increment attempts
	var attempts int

	query1 := `SELECT attempts FROM mpin_attempts WHERE user_id = ? AND otp_access_token = ?`

	err := r.db.QueryRowContext(ctx, query1, userID, otpAccessToken).Scan(&attempts)

	now := time.Now()

	if errors.Is(err, sql.ErrNoRows) {
		// if attemps is not their it means these is first time, so we first insert
		query2 := `INSERT INTO mpin_attempts (user_id, otp_access_token, attempts, updated_at) VALUES (?, ?, 1, ?)`
		_, err := r.db.ExecContext(ctx, query2, userID, otpAccessToken, now)
		return err
	}

	if err != nil {
		return err
	}

	attempts++

	var blockedUntil *time.Time

	// checking attemsps >= maxAttempts
	if attempts >= maxAttempts {
		t := now.Add(blockTime)
		blockedUntil = &t
	}

	// Update the DB
	query3 := `UPDATE mpin_attempts SET attempts = ?, blocked_until = ?, updated_at = ?
	WHERE user_id = ? AND otp_access_token = ?`

	_, err = r.db.ExecContext(ctx, query3, attempts, blockedUntil, now, userID, otpAccessToken)

	return err

}

func (r *MPINRateLimiter) Reset(ctx context.Context, userID, otpAccessToken string) error {
	// clear attempts
	_, err := r.db.ExecContext(
		ctx,
		`DELETE FROM mpin_attempts 
		 WHERE user_id = ? AND otp_access_token = ?`,
		userID, otpAccessToken,
	)

	return err
}
