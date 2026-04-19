package auth

import (
	"context"
	"time"
)

type OTPRepository interface {
	Save(ctx context.Context, userID, otpHash string, expiresAt time.Time) error
	Find(ctx context.Context, userID string) (otpHash string, expiresAt time.Time, err error)
	Delete(ctx context.Context, userID string) error
}
