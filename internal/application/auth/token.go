package authapp

import (
	"time"
)

type TokenType string

type CustomClaims struct {
	UserID    string
	Phone     string
	TokenType TokenType
}

const (
	TokenTypeOtpAccessToken            TokenType = "otp_access_token"
	TokenTypeAccessToken               TokenType = "access_token"
	TokenTypeOtpVerificationToken      TokenType = "otp_verification_token"
	TokenTypeResetOtpVerificationToken TokenType = "otp_reset_verification_token"
	TokenTypeResetPasswordToken        TokenType = "reset_password_token"
)

type TokenManager interface {
	GenerateToken(userID string, phone string, tokenType TokenType, duration time.Duration) (string, error)
	VerifyToken(tokenString string) (*CustomClaims, error)
}
