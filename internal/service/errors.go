// Package service provides application-level services for managing cards, memos, and users.
package service

import "errors"

// Common service errors - sentinel errors used across service implementations.
// These errors represent common conditions that callers may want to check for with errors.Is().
// For details on the standardized error handling approach, see README.md.
//
// Error handling principles:
// 1. Service methods return sentinel errors for expected error conditions
// 2. Unexpected errors are wrapped in service-specific error types
// 3. Callers use errors.Is/errors.As to check for specific error conditions
// 4. The API layer maps service errors to appropriate HTTP status codes
var (
	// ErrNotOwned indicates a resource is owned by a different user than the one making the request.
	// This is typically returned when a user attempts to modify a resource they don't own.
	// API layer should map this to HTTP 403 Forbidden.
	ErrNotOwned = errors.New("resource is owned by another user")

	// ErrStatsNotFound indicates that user card statistics were not found.
	// This is typically returned when stats for a user-card pair don't exist.
	// API layer should map this to HTTP 404 Not Found.
	ErrStatsNotFound = errors.New("user card statistics not found")
)
