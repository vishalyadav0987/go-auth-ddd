package auth

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotVerified    = errors.New("user not verified")
	ErrInvalidOTP         = errors.New("invalid otp")
	ErrOTPExpired         = errors.New("otp expired")
	ErrInvalidMPIN        = errors.New("invalid mpin")
	ErrInvalidToken       = errors.New("invalid token")
)
