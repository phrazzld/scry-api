package api

import (
	"github.com/google/uuid"
)

// Common request/response structures

// RegisterRequest defines the payload for the user registration endpoint.
type RegisterRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=12,max=72"`
}

// LoginRequest defines the payload for the user login endpoint.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=1"`
}

// AuthResponse defines the successful response for authentication endpoints.
type AuthResponse struct {
	// UserID is the unique identifier for the authenticated user
	UserID uuid.UUID `json:"user_id"`

	// AccessToken is the JWT token used for API authorization
	// Field renamed from Token for clarity but JSON field name kept for backward compatibility
	AccessToken string `json:"token"`

	// RefreshToken is the JWT token used to obtain new access tokens
	RefreshToken string `json:"refresh_token,omitempty"`

	// ExpiresAt is the ISO 8601 timestamp when the access token expires
	ExpiresAt string `json:"expires_at,omitempty"`
}

// RefreshTokenRequest defines the payload for the token refresh endpoint.
type RefreshTokenRequest struct {
	// RefreshToken is the JWT refresh token to be used to obtain a new token pair
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshTokenResponse defines the successful response for the token refresh endpoint.
type RefreshTokenResponse struct {
	// AccessToken is the new JWT token used for API authorization
	AccessToken string `json:"access_token"`

	// RefreshToken is the new JWT token used to obtain future access tokens
	RefreshToken string `json:"refresh_token"`

	// ExpiresAt is the ISO 8601 timestamp when the access token expires
	ExpiresAt string `json:"expires_at"`
}
