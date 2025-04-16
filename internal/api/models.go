package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
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

// ErrorResponse defines the standard error response structure.
type ErrorResponse struct {
	Error string `json:"error"`
}

// Helper functions

// RespondWithJSON writes a JSON response with the given status code and data.
func RespondWithJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// RespondWithError writes a JSON error response with the given status code and message.
func RespondWithError(w http.ResponseWriter, r *http.Request, status int, message string) {
	RespondWithJSON(w, r, status, ErrorResponse{Error: message})
}

// DecodeJSON decodes the request body into the given struct.
func DecodeJSON(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return err
	}
	return nil
}

// ValidateRequest validates the given struct using the validator package.
func ValidateRequest(v interface{}) error {
	validate := validator.New()
	return validate.Struct(v)
}
