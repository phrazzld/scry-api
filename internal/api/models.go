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
	UserID    uuid.UUID `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt string    `json:"expires_at,omitempty"`
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
