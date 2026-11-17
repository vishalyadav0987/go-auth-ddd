package authapp

import (
	"time"

	domain "github.com/vishalyadav0987/authentication/internal/domain/auth"
)

type TokenType string

type CustomClaims struct {
	UserID    string
	Phone     string
	TokenType TokenType

	// Role is set only on access tokens.
	Role domain.Role
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

	// GenerateAccessToken issues an access token carrying the user's role.
	GenerateAccessToken(userID string, phone string, role domain.Role, duration time.Duration) (string, error)
	VerifyToken(tokenString string) (*CustomClaims, error)
}
