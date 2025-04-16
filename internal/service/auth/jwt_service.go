package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// JWTService defines operations for managing JWT authentication tokens.
type JWTService interface {
	// GenerateToken creates a signed JWT access token containing the user's information.
	// Returns the token string or an error if token generation fails.
	GenerateToken(ctx context.Context, userID uuid.UUID) (string, error)

	// ValidateToken validates the provided access token string and extracts the claims.
	// Returns the claims containing user information if the token is valid,
	// or an error if validation fails (expired, invalid signature, etc.).
	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)

	// GenerateRefreshToken creates a signed JWT refresh token containing the user's information.
	// Refresh tokens have a longer lifetime and are used to obtain new access tokens.
	// Returns the refresh token string or an error if token generation fails.
	GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error)

	// ValidateRefreshToken validates the provided refresh token string and extracts the claims.
	// Returns the claims containing user information if the refresh token is valid,
	// or an error if validation fails (expired, invalid signature, wrong token type, etc.).
	ValidateRefreshToken(ctx context.Context, tokenString string) (*Claims, error)
}

// Claims represents the custom claims structure for the JWT tokens.
// It extends standard JWT registered claims with application-specific fields.
type Claims struct {
	// UserID is the unique identifier of the user the token was issued for.
	UserID uuid.UUID `json:"uid,omitempty"`

	// TokenType indicates the purpose of the token ("access" or "refresh").
	// Used to prevent token misuse across different contexts.
	TokenType string `json:"type,omitempty"`

	// Standard registered JWT claims
	Subject   string    `json:"sub,omitempty"`
	IssuedAt  time.Time `json:"iat,omitempty"`
	ExpiresAt time.Time `json:"exp,omitempty"`
	ID        string    `json:"jti,omitempty"`
}
