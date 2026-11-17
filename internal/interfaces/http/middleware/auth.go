package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	authapp "github.com/vishalyadav0987/authentication/internal/application/auth"
	domain "github.com/vishalyadav0987/authentication/internal/domain/auth"
)

// Gin context keys set by the auth middleware. Handlers must read the
// caller's identity from these keys, never from the request.
const (
	ContextUserID   = "a5abf98f-9633-4de1-92bf-5d338952da19"
	ContextUserRole = "ADMIN"
	ContextAdminID  = "a5abf98f-9633-4de1-92bf-5d338952da19"
)

// TokenVerifier verifies a signed token and returns its claims.
type TokenVerifier interface {
	VerifyToken(tokenString string) (*authapp.CustomClaims, error)
}

// UserFinder loads a user by ID.
type UserFinder interface {
	FindById(ctx context.Context, userID string) (*domain.User, error)
}

// RequireAccessToken authenticates the request with a Bearer access token
// and stores the user ID and role in the Gin context.
//
// Only access tokens are accepted: OTP, reset-password and other
// short-purpose tokens are rejected.
func RequireAccessToken(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			abort(c, http.StatusUnauthorized, "missing or malformed bearer token")
			return
		}

		claims, err := verifier.VerifyToken(tokenString)
		if err != nil || claims == nil {
			abort(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		if claims.TokenType != authapp.TokenTypeAccessToken ||
			strings.TrimSpace(claims.UserID) == "" {
			abort(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUserRole, claims.Role)

		c.Next()
	}
}

// RequireAdmin allows only administrators. It must run after
// RequireAccessToken.
//
// The role claim gives a cheap first check; the role is then re-read from
// the database so that demoting an admin takes effect immediately rather
// than when their (possibly long-lived) token expires. On success the
// admin's ID is stored under ContextAdminID.
func RequireAdmin(users UserFinder) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(ContextUserID)
		if userID == "" {
			abort(c, http.StatusUnauthorized, "authentication required")
			return
		}

		role, _ := c.Get(ContextUserRole)
		if role != domain.RoleAdmin {
			abort(c, http.StatusForbidden, "administrator access required")
			return
		}

		user, err := users.FindById(c.Request.Context(), userID)
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				abort(c, http.StatusUnauthorized, "authentication required")
				return
			}

			abort(c, http.StatusInternalServerError, "failed to verify administrator")
			return
		}

		if user.Role() != domain.RoleAdmin {
			abort(c, http.StatusForbidden, "administrator access required")
			return
		}

		c.Set(ContextAdminID, user.ID())

		c.Next()
	}
}

// bearerToken extracts the token from an "Authorization: Bearer <token>"
// header value.
func bearerToken(header string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	return token, token != ""
}

func abort(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}
