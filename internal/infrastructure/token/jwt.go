package token

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	authapp "github.com/vishalyadav0987/authentication/internal/application/auth"
)

type jwtClaims struct {
	UserID    string            `json:"sub"`
	Phone     string            `json:"phone"`
	TokenType authapp.TokenType `json:"type"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey string
}

func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{
		secretKey: secret,
	}
}

func (j *JWTManager) GenerateToken(
	userID string,
	phone string,
	tokenType authapp.TokenType,
	duration time.Duration) (string, error) {
	now := time.Now()

	claims := jwtClaims{
		UserID:    userID,
		Phone:     phone,
		TokenType: tokenType,
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
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, err
	}

	return &authapp.CustomClaims{
		UserID:    claims.UserID,
		Phone:     claims.Phone,
		TokenType: claims.TokenType,
	}, nil

}
