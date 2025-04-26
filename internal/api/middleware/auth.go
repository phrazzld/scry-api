package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/redact"
	"github.com/phrazzld/scry-api/internal/service/auth"
)

// AuthMiddleware provides JWT authentication for routes.
type AuthMiddleware struct {
	jwtService auth.JWTService
}

// NewAuthMiddleware creates a new AuthMiddleware with the given dependencies.
func NewAuthMiddleware(jwtService auth.JWTService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
	}
}

// Authenticate validates JWT tokens from the Authorization header and
// adds the user ID to the request context for authorized requests.
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			api.HandleAPIError(w, r, auth.ErrInvalidToken, "Authorization header required")
			return
		}

		// Check Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			api.HandleAPIError(w, r, auth.ErrInvalidToken, "Invalid authorization format")
			return
		}

		token := parts[1]

		// Validate token
		claims, err := m.jwtService.ValidateToken(r.Context(), token)
		if err != nil {
			switch err {
			case auth.ErrExpiredToken:
				api.HandleAPIError(w, r, err, "Token expired")
			case auth.ErrInvalidToken:
				api.HandleAPIError(w, r, err, "Invalid token")
			default:
				slog.Error("failed to validate token", "error", redact.Error(err))
				api.HandleAPIError(w, r, err, "Authentication error")
			}
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), shared.UserIDContextKey, claims.UserID)

		// Continue with the authenticated request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts the user ID from the request context.
// Returns the user ID and a boolean indicating if it was found.
func GetUserID(r *http.Request) (uuid.UUID, bool) {
	userID, ok := r.Context().Value(shared.UserIDContextKey).(uuid.UUID)
	return userID, ok
}
