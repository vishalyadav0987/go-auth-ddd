package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	authapp "github.com/vishalyadav0987/authentication/internal/application/auth"
	domain "github.com/vishalyadav0987/authentication/internal/domain/auth"
)

var errInvalidToken = errors.New("invalid token")

type jwtClaims struct {
	UserID    string            `json:"sub"`
	Phone     string            `json:"phone"`
	TokenType authapp.TokenType `json:"type"`
	Role      domain.Role       `json:"role,omitempty"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey string
}

var _ authapp.TokenManager = (*JWTManager)(nil)

func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{
		secretKey: secret,
	}
}

// GenerateToken issues a non-access token (OTP, reset password, ...).
// These tokens carry no role.
func (j *JWTManager) GenerateToken(
	userID string,
	phone string,
	tokenType authapp.TokenType,
	duration time.Duration,
) (string, error) {
	return j.sign(userID, phone, tokenType, "", duration)
}

// GenerateAccessToken issues an access token carrying the user's role.
func (j *JWTManager) GenerateAccessToken(
	userID string,
	phone string,
	role domain.Role,
	duration time.Duration,
) (string, error) {
	if !role.IsValid() {
		return "", domain.ErrInvalidRole
	}

	return j.sign(userID, phone, authapp.TokenTypeAccessToken, role, duration)
}

func (j *JWTManager) sign(
	userID string,
	phone string,
	tokenType authapp.TokenType,
	role domain.Role,
	duration time.Duration,
) (string, error) {
	now := time.Now()

	claims := jwtClaims{
		UserID:    userID,
		Phone:     phone,
		TokenType: tokenType,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(j.secretKey))
}

func (j *JWTManager) VerifyToken(tokenString string) (*authapp.CustomClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwtClaims{},
		func(t *jwt.Token) (any, error) {
			return []byte(j.secretKey), nil
		},
		// Only accept the algorithm we sign with.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, errInvalidToken
	}

	return &authapp.CustomClaims{
		UserID:    claims.UserID,
		Phone:     claims.Phone,
		TokenType: claims.TokenType,
		Role:      claims.Role,
	}, nil
}
