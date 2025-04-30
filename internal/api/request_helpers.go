package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/redact"
)

// getUserIDFromContext extracts the authenticated user's UUID from the request context.
// The user ID is expected to be placed in the context by the authentication middleware.
//
// Parameters:
//   - r: The HTTP request containing the context
//
// Returns:
//   - (uuid.UUID, true): The user's UUID if successfully extracted
//   - (uuid.UUID{}, false): A zero UUID and false if user ID not found or invalid
func getUserIDFromContext(r *http.Request) (uuid.UUID, bool) {
	userID, ok := r.Context().Value(shared.UserIDContextKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, false
	}
	return userID, true
}

// getPathUUID extracts a UUID from the URL path parameters.
// It parses and validates the UUID, handling common error cases.
//
// Parameters:
//   - r: The HTTP request
//   - paramName: The name of the path parameter to extract
//
// Returns:
//   - (uuid.UUID, nil): The parsed UUID if valid
//   - (uuid.UUID{}, error): A zero UUID and appropriate error if parameter is missing or invalid
func getPathUUID(r *http.Request, paramName string) (uuid.UUID, error) {
	// Extract parameter from URL path using chi router
	pathParam := chi.URLParam(r, paramName)
	if pathParam == "" {
		return uuid.Nil, domain.NewValidationError(paramName, "is required", domain.ErrValidation)
	}

	// Parse parameter as UUID
	id, err := uuid.Parse(pathParam)
	if err != nil {
		return uuid.Nil, domain.NewValidationError(paramName, "has invalid format", domain.ErrInvalidID)
	}

	return id, nil
}

// handleUserIDAndPathUUID is a composite helper that extracts both the user ID from context
// and a UUID from the path parameters. It writes an error response if either extraction fails.
//
// Parameters:
//   - w: The HTTP response writer
//   - r: The HTTP request
//   - paramName: The name of the path parameter to extract
//   - log: The logger to use
//
// Returns:
//   - (userID, pathID, true): The user UUID and path UUID if both were extracted successfully
//   - (uuid.UUID{}, uuid.UUID{}, false): Zero UUIDs and false if extraction failed and an error was written
func handleUserIDAndPathUUID(
	w http.ResponseWriter,
	r *http.Request,
	paramName string,
	log *slog.Logger,
) (uuid.UUID, uuid.UUID, bool) {
	// Get logger from context if not provided
	if log == nil {
		log = logger.FromContextOrDefault(r.Context(), slog.Default())
	}

	// Extract user ID from context
	userID, ok := getUserIDFromContext(r)
	if !ok {
		log.Warn("user ID not found or invalid in request context")
		HandleAPIError(w, r, domain.ErrUnauthorized, "User ID not found or invalid")
		return uuid.Nil, uuid.Nil, false
	}

	// Extract path UUID
	pathID, err := getPathUUID(r, paramName)
	if err != nil {
		logMsg := "invalid path parameter"
		if paramName != "" {
			logMsg = "invalid " + paramName
		}
		log.Warn(logMsg, slog.String("param_name", paramName), slog.String("value", chi.URLParam(r, paramName)))
		HandleAPIError(w, r, err, "")
		return uuid.Nil, uuid.Nil, false
	}

	return userID, pathID, true
}

// handleUserIDFromContext is a helper function to extract the user ID from the request context
// and write an error response if extraction fails.
//
// Parameters:
//   - w: The HTTP response writer
//   - r: The HTTP request
//   - log: The logger to use
//
// Returns:
//   - (uuid.UUID, true): The user UUID if successfully extracted
//   - (uuid.UUID{}, false): A zero UUID and false if extraction failed and an error was written
func handleUserIDFromContext(
	w http.ResponseWriter,
	r *http.Request,
	log *slog.Logger,
) (uuid.UUID, bool) {
	// Get logger from context if not provided
	if log == nil {
		log = logger.FromContextOrDefault(r.Context(), slog.Default())
	}

	// Extract user ID from context
	userID, ok := getUserIDFromContext(r)
	if !ok {
		log.Warn("user ID not found or invalid in request context")
		HandleAPIError(w, r, domain.ErrUnauthorized, "User ID not found or invalid")
		return uuid.Nil, false
	}

	return userID, true
}

// parseAndValidateRequest is a helper function to decode and validate a request body.
// It writes an error response if parsing or validation fails.
//
// Parameters:
//   - w: The HTTP response writer
//   - r: The HTTP request
//   - req: Pointer to the struct to decode into
//   - log: The logger to use
//   - logFields: Optional list of slog fields to include in log messages
//
// Returns:
//   - true if parsing and validation succeeded
//   - false if parsing or validation failed and an error was written
func parseAndValidateRequest(
	w http.ResponseWriter,
	r *http.Request,
	req interface{},
	log *slog.Logger,
	logFields ...slog.Attr,
) bool {
	// Get logger from context if not provided
	if log == nil {
		log = logger.FromContextOrDefault(r.Context(), slog.Default())
	}

	// Parse request body
	if err := shared.DecodeJSON(r, req); err != nil {
		// Create a slice of log args with the error first
		logArgs := []any{slog.String("error", redact.Error(err))}

		// Convert slog.Attr to any for the log.Warn call
		for _, field := range logFields {
			logArgs = append(logArgs, field)
		}

		log.Warn("invalid request format", logArgs...)
		HandleValidationError(w, r, err)
		return false
	}

	// Validate request
	if err := shared.Validate.Struct(req); err != nil {
		// Create a slice of log args with the error first
		logArgs := []any{slog.String("error", redact.Error(err))}

		// Convert slog.Attr to any for the log.Warn call
		for _, field := range logFields {
			logArgs = append(logArgs, field)
		}

		log.Warn("validation error", logArgs...)
		HandleValidationError(w, r, err)
		return false
	}

	return true
}
