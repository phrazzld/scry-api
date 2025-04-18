package store

import "errors"

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
)
