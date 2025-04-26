package store

import (
	"errors"
	"fmt"
)

// Common store errors used across all store implementations.
var (
	// ErrNotFound is returned when a requested entity does not exist in the store.
	// This is a generic version of the entity-specific not found errors
	// (e.g., ErrUserNotFound, ErrMemoNotFound).
	ErrNotFound = errors.New("entity not found")

	// ErrDuplicate is returned when an operation would create a duplicate
	// of a unique entity (e.g., a user with the same email).
	ErrDuplicate = errors.New("entity already exists")

	// ErrNotImplemented is returned when a store method is not yet implemented.
	// This is particularly useful for stub implementations.
	ErrNotImplemented = errors.New("method not implemented")

	// ErrInvalidEntity is returned when an entity fails validation before
	// being stored. Check the wrapped error for specific validation details.
	ErrInvalidEntity = errors.New("invalid entity")

	// ErrUpdateFailed is returned when an update operation fails, for example
	// because the entity does not exist or the update violates constraints.
	ErrUpdateFailed = errors.New("update failed")

	// ErrDeleteFailed is returned when a delete operation fails, for example
	// because the entity does not exist or is referenced by other entities.
	ErrDeleteFailed = errors.New("delete failed")

	// ErrTransactionFailed is returned when a database transaction fails
	// to commit or when an operation within a transaction fails.
	ErrTransactionFailed = errors.New("transaction failed")

	// Entity-specific "not found" errors

	// ErrUserNotFound indicates that the requested user does not exist in the store.
	ErrUserNotFound = fmt.Errorf("%w: user", ErrNotFound)

	// ErrMemoNotFound indicates that the requested memo does not exist in the store.
	ErrMemoNotFound = fmt.Errorf("%w: memo", ErrNotFound)

	// ErrCardNotFound indicates that the requested card does not exist in the store.
	ErrCardNotFound = fmt.Errorf("%w: card", ErrNotFound)

	// ErrUserCardStatsNotFound indicates that the requested user card stats do not exist in the store.
	ErrUserCardStatsNotFound = fmt.Errorf("%w: user card stats", ErrNotFound)

	// Entity-specific "duplicate" errors

	// ErrEmailExists indicates that a user with the given email already exists.
	// This is returned when attempting to create a user with an email that's already in use.
	ErrEmailExists = fmt.Errorf("%w: email", ErrDuplicate)
)

// IsNotFoundError checks if the error is any kind of "not found" error.
// This includes the generic ErrNotFound and all entity-specific not found errors.
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrUserNotFound) ||
		errors.Is(err, ErrMemoNotFound) ||
		errors.Is(err, ErrCardNotFound) ||
		errors.Is(err, ErrUserCardStatsNotFound)
}

// IsDuplicateError checks if the error is any kind of "duplicate" error.
// This includes the generic ErrDuplicate and all entity-specific duplicate errors.
func IsDuplicateError(err error) bool {
	return errors.Is(err, ErrDuplicate) ||
		errors.Is(err, ErrEmailExists)
}

// StoreError is a custom error type for store-specific errors with additional context.
type StoreError struct {
	Entity    string // The entity type (e.g., "user", "card")
	Operation string // The operation that failed (e.g., "create", "update")
	Message   string // Error message
	Err       error  // Original error
}

// Error implements the error interface for StoreError.
func (e *StoreError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf(
			"%s operation on %s failed: %s: %v",
			e.Operation,
			e.Entity,
			e.Message,
			e.Err,
		)
	}
	return fmt.Sprintf("%s operation on %s failed: %s", e.Operation, e.Entity, e.Message)
}

// Unwrap returns the wrapped error to support errors.Is/errors.As.
func (e *StoreError) Unwrap() error {
	return e.Err
}

// NewStoreError creates a new StoreError with the given entity, operation, message, and wrapped error.
func NewStoreError(entity, operation, message string, err error) *StoreError {
	return &StoreError{
		Entity:    entity,
		Operation: operation,
		Message:   message,
		Err:       err,
	}
}
