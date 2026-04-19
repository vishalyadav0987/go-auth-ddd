package authapp

import "context"

type OTPService interface {
	SendOtp(ctx context.Context, userID string) (string, error)
	VerifyOtp(ctx context.Context, userID, otp string) error
}
