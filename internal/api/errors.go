package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service"
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
		errors.Is(err, auth.ErrWrongTokenType),
		errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized

	// Authorization errors
	case errors.Is(err, card_review.ErrCardNotOwned),
		errors.Is(err, service.ErrNotOwned):
		return http.StatusForbidden

	// Not found errors
	case errors.Is(err, store.ErrUserNotFound),
		errors.Is(err, store.ErrCardNotFound),
		errors.Is(err, store.ErrMemoNotFound),
		errors.Is(err, store.ErrNotFound),
		errors.Is(err, card_review.ErrCardNotFound),
		errors.Is(err, card_review.ErrCardStatsNotFound):
		return http.StatusNotFound

	// Conflict errors
	case errors.Is(err, store.ErrEmailExists),
		errors.Is(err, store.ErrDuplicate):
		return http.StatusConflict

	// Bad request errors - validation errors and invalid entities
	case errors.Is(err, store.ErrInvalidEntity),
		errors.Is(err, card_review.ErrInvalidAnswer),
		errors.Is(err, domain.ErrValidation),
		errors.Is(err, domain.ErrInvalidFormat),
		errors.Is(err, domain.ErrInvalidID),
		errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPassword),
		errors.Is(err, domain.ErrEmptyContent),
		errors.Is(err, domain.ErrInvalidReviewOutcome),
		errors.Is(err, domain.ErrInvalidCardContent),
		errors.Is(err, domain.ErrInvalidMemoStatus):
		return http.StatusBadRequest

	// Special cases
	case errors.Is(err, card_review.ErrNoCardsDue):
		return http.StatusNoContent

	// Default: internal server error
	default:
		// Check if the error is a wrapped validation error
		var validationErr *domain.ValidationError
		if errors.As(err, &validationErr) {
			return http.StatusBadRequest
		}

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

	// First check for custom error types with additional context
	// These checks need to come before the errors.Is checks to ensure
	// we get the most specific and helpful error messages
	var validationErr *domain.ValidationError
	if errors.As(err, &validationErr) {
		if validationErr.Field != "" {
			return fmt.Sprintf("Invalid %s: %s", validationErr.Field, validationErr.Message)
		}
		return validationErr.Message
	}

	// Handle service errors with wrapped errors
	var serviceErr *card_review.ServiceError
	if errors.As(err, &serviceErr) {
		// Check if the service error wraps a specific error we have a better message for
		if serviceErr.Err != nil {
			// Try to get a message for the wrapped error
			innerMessage := GetSafeErrorMessage(serviceErr.Err)
			if innerMessage != "An unexpected error occurred" {
				return innerMessage
			}
		}
		return "Card review operation failed"
	}

	// Handle store errors with wrapped errors
	var storeErr *store.StoreError
	if errors.As(err, &storeErr) {
		// Try to get a message for the wrapped error
		if storeErr.Err != nil {
			innerMessage := GetSafeErrorMessage(storeErr.Err)
			if innerMessage != "An unexpected error occurred" {
				return innerMessage
			}
		}
		return fmt.Sprintf("Operation failed: %s", storeErr.Message)
	}

	// Map specific sentinel error types to user-friendly messages
	switch {
	// Authentication errors
	case errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, auth.ErrExpiredToken):
		return "Invalid token"

	case errors.Is(err, auth.ErrInvalidRefreshToken),
		errors.Is(err, auth.ErrExpiredRefreshToken),
		errors.Is(err, auth.ErrWrongTokenType):
		return "Invalid refresh token"

	case errors.Is(err, domain.ErrUnauthorized):
		return "Unauthorized operation"

	// Authorization errors
	case errors.Is(err, card_review.ErrCardNotOwned),
		errors.Is(err, service.ErrNotOwned):
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

	case errors.Is(err, store.ErrNotFound):
		return "Resource not found"

	// Conflict errors
	case errors.Is(err, store.ErrEmailExists):
		return "Email already exists"

	case errors.Is(err, store.ErrDuplicate):
		return "Resource already exists"

	// Bad request errors - domain validation errors
	case errors.Is(err, domain.ErrValidation):
		return "Validation failed"

	case errors.Is(err, domain.ErrInvalidFormat):
		return "Invalid format"

	case errors.Is(err, domain.ErrInvalidID):
		return "Invalid ID"

	case errors.Is(err, domain.ErrInvalidEmail):
		return "Invalid email format"

	case errors.Is(err, domain.ErrInvalidPassword):
		return "Invalid password"

	case errors.Is(err, domain.ErrEmptyContent):
		return "Content cannot be empty"

	case errors.Is(err, domain.ErrInvalidReviewOutcome):
		return "Invalid review outcome"

	case errors.Is(err, domain.ErrInvalidCardContent):
		return "Invalid card content"

	case errors.Is(err, domain.ErrInvalidMemoStatus):
		return "Invalid memo status"

	// Store/service specific errors
	case errors.Is(err, store.ErrInvalidEntity):
		return "Invalid entity data"

	case errors.Is(err, card_review.ErrInvalidAnswer):
		return "Invalid answer"

	// Card review related errors
	case errors.Is(err, card_review.ErrNoCardsDue):
		// This should not happen as we return StatusNoContent, but for completeness
		return "No cards due for review"

	// Default case for unknown errors
	default:
		return "An unexpected error occurred"
	}
}

// SanitizeValidationError extracts validation details from structured validation errors
// or uses type checking to provide a user-friendly message.
//
// For go-playground/validator errors, it attempts to parse the field and tag from
// the validator's structured error format.
//
// For domain.ValidationError types, it uses the field and message directly.
func SanitizeValidationError(err error) string {
	// First, check if we have a domain.ValidationError
	var validationErr *domain.ValidationError
	if errors.As(err, &validationErr) {
		if validationErr.Field != "" {
			return fmt.Sprintf("Invalid %s: %s", validationErr.Field, validationErr.Message)
		}
		return validationErr.Message
	}

	// Try to extract field and tag information from the go-playground/validator error format
	// Example: "Key: 'LoginRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag"
	errStr := err.Error()

	// Look for validator's structured error format
	if field, tag, ok := extractValidatorFieldAndTag(errStr); ok {
		if tag != "" {
			return fmt.Sprintf("Invalid %s: %s", field, getValidationTagMessage(tag))
		}
		return fmt.Sprintf("Invalid %s", field)
	}

	// Fall back to a generic validation error message
	return "Validation error"
}

// extractValidatorFieldAndTag attempts to extract the field name and validation tag
// from a go-playground/validator error message.
// Returns the field name, tag, and whether extraction was successful.
func extractValidatorFieldAndTag(errStr string) (string, string, bool) {
	// Check for the typical validator error format
	if !strings.Contains(errStr, "Field validation") {
		return "", "", false
	}

	// Example: "Key: 'LoginRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag"
	parts := strings.Split(errStr, "Error:")
	if len(parts) < 2 {
		return "", "", false
	}

	// Extract just the field validation part
	fieldParts := strings.Split(parts[1], "'")
	if len(fieldParts) < 3 {
		return "", "", false
	}

	field := fieldParts[1]
	var tag string
	if len(fieldParts) >= 5 {
		tag = fieldParts[3]
	}

	return field, tag, true
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

// HandleAPIError is a centralized helper function that handles API errors consistently.
// It maps the error to an HTTP status code, generates a user-friendly message,
// and responds with an appropriate HTTP error.
//
// Parameters:
// - w: The HTTP response writer
// - r: The HTTP request
// - err: The error to handle
// - defaultMsg: An optional default message to use for internal server errors
// - opts: Optional response options (like WithElevatedLogLevel)
//
// This centralized function reduces duplication across handlers and ensures
// consistent error handling throughout the API.
func HandleAPIError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	defaultMsg string,
	opts ...shared.ResponseOption,
) {
	// Map error to appropriate HTTP status code
	statusCode := MapErrorToStatusCode(err)

	// Get a safe, user-friendly message
	safeMessage := GetSafeErrorMessage(err)

	// For internal server errors, use the default message if provided
	if statusCode == http.StatusInternalServerError && defaultMsg != "" {
		safeMessage = defaultMsg
	}

	// Respond with error using centralized shared function
	shared.RespondWithErrorAndLog(w, r, statusCode, safeMessage, err, opts...)
}

// HandleValidationError is a specialized version of HandleAPIError for validation errors.
// It sanitizes the validation error and responds with a BadRequest status.
//
// Parameters:
// - w: The HTTP response writer
// - r: The HTTP request
// - err: The validation error to handle
// - opts: Optional response options (like WithElevatedLogLevel)
func HandleValidationError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	opts ...shared.ResponseOption,
) {
	// Sanitize the validation error message
	sanitizedError := SanitizeValidationError(err)

	// Always use BadRequest status for validation errors
	shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, sanitizedError, err, opts...)
}
