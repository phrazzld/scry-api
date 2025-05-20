// Package service provides application-level services for managing cards, memos, and users.
package service

import (
	"errors"
	"fmt"
)

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

// ServiceError represents an error that occurred in a service.
// It provides context about the service and operation where the error occurred.
type ServiceError struct {
	Service string // The service where the error occurred (e.g., "user", "card")
	Op      string // The operation that failed (e.g., "create", "get")
	Err     error  // The underlying error
}

// Error implements the error interface for ServiceError.
func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s service %s operation failed: %v", e.Service, e.Op, e.Err)
	}
	return fmt.Sprintf("%s service %s operation failed", e.Service, e.Op)
}

// Unwrap returns the underlying error for compatibility with errors.Is/As.
func (e *ServiceError) Unwrap() error {
	return e.Err
}

// NewServiceError creates a new ServiceError.
func NewServiceError(service, op string, err error) error {
	return &ServiceError{
		Service: service,
		Op:      op,
		Err:     err,
	}
}
