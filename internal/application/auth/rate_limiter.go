package authapp

import "context"

type MPINRateLimiter interface {
	Allow(ctx context.Context, userID, otpToken string) error
	RegisterFailure(ctx context.Context, userID, otpToken string) error
	Reset(ctx context.Context, userID, otpToken string) error
}
