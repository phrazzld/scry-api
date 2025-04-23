package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
)

// MapErrorToStatusCode maps internal errors to appropriate HTTP status codes
// based on the error type. This prevents leaking internal error types or
// messages to clients.
func MapErrorToStatusCode(err error) int {
	switch {
	// Authentication errors
	case errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, auth.ErrExpiredToken),
		errors.Is(err, auth.ErrInvalidRefreshToken),
		errors.Is(err, auth.ErrExpiredRefreshToken),
		errors.Is(err, auth.ErrWrongTokenType):
		return http.StatusUnauthorized

	// Authorization errors
	case errors.Is(err, card_review.ErrCardNotOwned):
		return http.StatusForbidden

	// Not found errors
	case errors.Is(err, store.ErrUserNotFound),
		errors.Is(err, store.ErrCardNotFound),
		errors.Is(err, store.ErrMemoNotFound),
		errors.Is(err, card_review.ErrCardNotFound),
		errors.Is(err, card_review.ErrCardStatsNotFound):
		return http.StatusNotFound

	// Conflict errors
	case errors.Is(err, store.ErrEmailExists):
		return http.StatusConflict

	// Bad request errors
	case errors.Is(err, store.ErrInvalidEntity),
		errors.Is(err, card_review.ErrInvalidAnswer):
		return http.StatusBadRequest

	// Special cases
	case errors.Is(err, card_review.ErrNoCardsDue):
		return http.StatusNoContent

	// Default: internal server error
	default:
		return http.StatusInternalServerError
	}
}

// GetSafeErrorMessage returns a sanitized, user-friendly error message
// based on the error type. This prevents leaking sensitive internal details.
func GetSafeErrorMessage(err error) string {
	// Handle nil error
	if err == nil {
		return "An unexpected error occurred"
	}

	// Map specific error types to user-friendly messages
	switch {
	// Authentication errors
	case errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, auth.ErrExpiredToken):
		return "Invalid token"

	case errors.Is(err, auth.ErrInvalidRefreshToken),
		errors.Is(err, auth.ErrExpiredRefreshToken),
		errors.Is(err, auth.ErrWrongTokenType):
		return "Invalid refresh token"

	// Authorization errors
	case errors.Is(err, card_review.ErrCardNotOwned):
		return "You do not own this card"

	// Not found errors
	case errors.Is(err, store.ErrUserNotFound):
		return "User not found"

	case errors.Is(err, store.ErrCardNotFound),
		errors.Is(err, card_review.ErrCardNotFound):
		return "Card not found"

	case errors.Is(err, store.ErrMemoNotFound):
		return "Memo not found"

	case errors.Is(err, card_review.ErrCardStatsNotFound):
		return "Card statistics not found"

	// Conflict errors
	case errors.Is(err, store.ErrEmailExists):
		return "Email already exists"

	// Bad request errors
	case errors.Is(err, store.ErrInvalidEntity):
		return "Invalid entity data"

	case errors.Is(err, card_review.ErrInvalidAnswer):
		return "Invalid answer"

	// No cards due is handled separately with StatusNoContent

	// Default case for unknown errors
	default:
		// Check if we're in a card review context by looking at the error string
		if strings.Contains(err.Error(), "submit answer") {
			return "Failed to submit answer"
		} else if strings.Contains(err.Error(), "get next") {
			return "Failed to get next review card"
		}
		return "An unexpected error occurred"
	}
}

// SanitizeValidationError removes sensitive details from validation errors
// and returns a user-friendly message.
func SanitizeValidationError(err error) string {
	errMsg := err.Error()

	// Check if this is likely a validation error message
	if strings.Contains(errMsg, "Field validation") {
		// Extract the field name and validation tag
		// Example format: "Key: 'LoginRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag"
		parts := strings.Split(errMsg, "Error:")
		if len(parts) >= 2 {
			// Further split to get just the field validation part
			fieldParts := strings.Split(parts[1], "'")
			if len(fieldParts) >= 3 {
				field := fieldParts[1]
				var tag string
				if len(fieldParts) >= 5 {
					tag = fieldParts[3]
				}

				// Create a cleaner error message
				if tag != "" {
					return fmt.Sprintf("Invalid %s: %s", field, getValidationTagMessage(tag))
				}
				return fmt.Sprintf("Invalid %s", field)
			}
		}
	}

	// Fall back to a generic validation error message
	return "Validation error"
}

// getValidationTagMessage maps validation tags to user-friendly error messages
func getValidationTagMessage(tag string) string {
	switch tag {
	case "required":
		return "required field"
	case "email":
		return "invalid email format"
	case "min":
		return "too short"
	case "max":
		return "too long"
	case "oneof":
		return "invalid value"
	default:
		return "validation failed"
	}
}
