# Service Package - Error Handling Guidelines

This document outlines the standardized approach to error handling in the service layer, particularly for transaction callbacks and service interactions.

## Core Principles

1. Service errors should provide context about operations that failed
2. Error wrapping should preserve the original error type for `errors.Is`/`errors.As` checks
3. Sentinel errors should be used for common, expected error cases
4. Callers should use `errors.Is`/`errors.As` to check for specific error conditions

## Error Wrapping in Transaction Callbacks

When using `store.RunInTransaction`, follow these practices:

```go
err = store.RunInTransaction(
    ctx,
    repo.DB(),
    func(ctx context.Context, tx *sql.Tx) error {
        // Get transactional repositories
        txRepo := repo.WithTx(tx)

        // Call repository methods
        result, err := txRepo.SomeOperation(ctx, param)
        if err != nil {
            // 1. Check for specific error types first
            if store.IsNotFoundError(err) {
                return NewServiceError("operation_name", "resource not found", ErrResourceNotFound)
            }

            // 2. Otherwise, wrap the error with context for general errors
            return NewServiceError("operation_name", "failed to perform operation", err)
        }

        // 3. For validation or business rule failures, return sentinel errors directly
        if !isValid(result) {
            return ErrInvalidOperation
        }

        return nil
    },
)

// After transaction returns, pass the error upward - it's already wrapped appropriately
if err != nil {
    return nil, err
}
```

## Service Error Types

Each service should define its own custom error type with an `Unwrap()` method to support `errors.Is`/`errors.As`:

```go
// SomeServiceError is a custom error type for some service errors
type SomeServiceError struct {
    Operation string
    Message   string
    Err       error
}

// Error implements the error interface
func (e *SomeServiceError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("service %s failed: %s: %v", e.Operation, e.Message, e.Err)
    }
    return fmt.Sprintf("service %s failed: %s", e.Operation, e.Message)
}

// Unwrap returns the wrapped error to support errors.Is/errors.As
func (e *SomeServiceError) Unwrap() error {
    return e.Err
}

// NewSomeServiceError creates a new SomeServiceError
func NewSomeServiceError(operation, message string, err error) *SomeServiceError {
    return &SomeServiceError{
        Operation: operation,
        Message:   message,
        Err:       err,
    }
}
```

## Common Sentinel Errors

Common sentinel errors should be defined at the appropriate level:

1. **Service package level**: For errors common across multiple services
2. **Individual service level**: For errors specific to one service

Example of service-level sentinel errors:

```go
// Common service errors
var (
    // ErrNotOwned indicates a resource is owned by a different user than the one making the request
    ErrNotOwned = errors.New("resource is owned by another user")

    // ErrStatsNotFound indicates that user card statistics were not found
    ErrStatsNotFound = errors.New("user card statistics not found")
)
```

## Error Checking for Callers

Callers of service methods should use `errors.Is` or `errors.As` to check for specific error conditions:

```go
// In API handlers or other service consumers
card, err := cardService.GetCard(ctx, cardID)
if err != nil {
    // Check for specific error types
    if errors.Is(err, store.ErrCardNotFound) {
        // Handle not found case
        return NotFoundResponse()
    }

    if errors.Is(err, service.ErrNotOwned) {
        // Handle authorization error
        return ForbiddenResponse()
    }

    // Check for custom error types
    var cardErr *service.CardServiceError
    if errors.As(err, &cardErr) {
        // Access fields of the specific error type
        log.Error("Card service error", "operation", cardErr.Operation)
    }

    // Default error handling
    return InternalErrorResponse()
}
```

## Error Mapping in API Layer

The API layer should map service errors to appropriate HTTP status codes and user-friendly messages:

- Map `ErrNotOwned` to 403 Forbidden
- Map `ErrCardNotFound` to 404 Not Found
- Map validation errors to 400 Bad Request

The `MapErrorToStatusCode` and `GetSafeErrorMessage` functions in the API layer handle this mapping.

## Best Practices

1. **Be consistent**: Follow the same pattern in all services
2. **Add context**: Error messages should describe what operation failed
3. **Preserve error types**: Always wrap errors using `fmt.Errorf("...%w", err)` or proper error types
4. **Avoid leaking implementation details**: Internal errors should be wrapped/translated before returning to callers
5. **Log detailed errors**: Log internal errors at appropriate level while returning safe error messages to clients
