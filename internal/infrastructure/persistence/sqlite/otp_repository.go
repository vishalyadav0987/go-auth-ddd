package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	domain "github.com/vishalyadav0987/authentication/internal/domain/auth"
)

type OtpRepository struct {
	db *sql.DB
}

// constructor
func NewOtpRepository(db *sql.DB) *OtpRepository {
	return &OtpRepository{db: db}
}

func (r *OtpRepository) Save(ctx context.Context, userID, otpHash string, expiresAt time.Time) error {

	id := uuid.NewString()

	query := `
	INSERT INTO otps (id, user_id, code, expires_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(user_id) DO UPDATE SET
		code = excluded.code,
		expires_at = excluded.expires_at
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		id,
		userID,
		otpHash,
		expiresAt,
	)

	if err != nil {
		return fmt.Errorf("save otp: %w", err)
	}

	return nil
}

func (r *OtpRepository) Find(ctx context.Context, userID string) (string, time.Time, error) {
	query := `SELECT code, expires_at FROM otps WHERE user_id = ?`

	row := r.db.QueryRowContext(ctx, query, userID)

	var (
		otpHash   string
		expiresAt time.Time
	)

	err := row.Scan(&otpHash, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", time.Time{}, domain.ErrUserNotFound
	}

	if err != nil {
		return "", time.Time{}, err
	}

	return otpHash, expiresAt, nil
}

func (r *OtpRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM otps WHERE user_id = ?`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}
