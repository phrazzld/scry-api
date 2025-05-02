# Service Package - Error Handling Guidelines

This document outlines the standardized approach to error handling in the service layer, particularly for transaction callbacks and service interactions. Following these guidelines ensures consistent error handling across the application and provides a clear way for callers to detect and respond to different error conditions.

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
    // Check for specific sentinel error types first using errors.Is
    if errors.Is(err, store.ErrCardNotFound) {
        // Handle not found case
        return NotFoundResponse()
    }

    if errors.Is(err, service.ErrNotOwned) {
        // Handle authorization error
        return ForbiddenResponse()
    }

    if errors.Is(err, service.ErrStatsNotFound) {
        // Handle stats not found case
        return NotFoundResponse()
    }

    // Then check for custom error types using errors.As to access fields
    var cardErr *service.CardServiceError
    if errors.As(err, &cardErr) {
        // Access fields of the specific error type
        log.Error("Card service error",
            "operation", cardErr.Operation,
            "message", cardErr.Message)

        // Optionally check wrapped error
        if cardErr.Err != nil {
            // Handle based on the wrapped error
        }
    }

    // Default error handling for unexpected errors
    return InternalErrorResponse()
}
```

Real example from `api/card_handler.go`:

```go
func (h *Handler) GetCard(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := api.UserIDFromContext(ctx)
    cardID, err := api.GetCardIDFromRequest(r)
    if err != nil {
        api.RespondWithError(w, err)
        return
    }

    card, err := h.cardService.GetCard(ctx, cardID)
    if err != nil {
        // Check for specific error conditions using errors.Is
        if errors.Is(err, store.ErrCardNotFound) {
            api.RespondWithError(w, api.NewNotFoundError("card not found"))
            return
        }

        api.RespondWithError(w, err)
        return
    }

    // Verify ownership
    if card.UserID != userID {
        api.RespondWithError(w, api.NewForbiddenError("not authorized to access this card"))
        return
    }

    api.RespondWithJSON(w, http.StatusOK, card)
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
6. **Use meaningful operation names**: Operation names in error types should be descriptive and consistent (e.g., "create_cards", "update_content")
7. **Standardize error messages**: Keep error messages concise and consistent across similar operations

## Using errors.Is/errors.As vs. String Comparison

Always use `errors.Is` and `errors.As` rather than string comparison or direct type assertions:

### ✅ DO: Use errors.Is for sentinel errors

```go
// Good: Using errors.Is to check for sentinel errors
if errors.Is(err, service.ErrNotOwned) {
    return api.NewForbiddenError("not authorized to access this resource")
}
```

### ❌ DON'T: Compare error strings

```go
// Bad: String comparison is brittle and breaks the error wrapping chain
if err.Error() == "resource is owned by another user" {
    return api.NewForbiddenError("not authorized to access this resource")
}
```

### ✅ DO: Use errors.As for custom error types

```go
// Good: Using errors.As to extract custom error type details
var cardErr *service.CardServiceError
if errors.As(err, &cardErr) {
    log.Error("Card service error", "operation", cardErr.Operation)
    // Handle specific error types based on operation or other fields
}
```

### ❌ DON'T: Use direct type assertions

```go
// Bad: Direct type assertion doesn't work with wrapped errors
cardErr, ok := err.(*service.CardServiceError)
if ok {
    log.Error("Card service error", "operation", cardErr.Operation)
}
```

## Real-world Examples

### Error Type Definitions

The CardService defines its error type pattern as follows:

```go
// CardServiceError is a custom error type for card service errors.
type CardServiceError struct {
    Operation string
    Message   string
    Err       error
}

// Error implements the error interface for CardServiceError.
func (e *CardServiceError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("card service %s failed: %s: %v", e.Operation, e.Message, e.Err)
    }
    return fmt.Sprintf("card service %s failed: %s", e.Operation, e.Message)
}

// Unwrap returns the wrapped error to support errors.Is/errors.As.
func (e *CardServiceError) Unwrap() error {
    return e.Err
}

// NewCardServiceError creates a new CardServiceError.
func NewCardServiceError(operation, message string, err error) *CardServiceError {
    return &CardServiceError{
        Operation: operation,
        Message:   message,
        Err:       err,
    }
}
```

### Error Handling in Transactions

The CardService creates cards with proper transaction error handling:

```go
// CreateCards implements CardService.CreateCards
func (s *cardServiceImpl) CreateCards(ctx context.Context, cards []*domain.Card) error {
    log := logger.FromContextOrDefault(ctx, s.logger)

    // Input validation
    if len(cards) == 0 {
        log.Debug("no cards to create")
        return nil
    }

    // Run all operations in a single transaction for atomicity
    return store.RunInTransaction(
        ctx,
        s.cardRepo.DB(),
        func(ctx context.Context, tx *sql.Tx) error {
            // Get transactional repositories
            txCardRepo := s.cardRepo.WithTx(tx)
            txStatsRepo := s.statsRepo.WithTx(tx)

            // 1. Create the cards within the transaction
            err := txCardRepo.CreateMultiple(ctx, cards)
            if err != nil {
                log.Error("failed to create cards in transaction",
                    slog.String("error", err.Error()))
                return NewCardServiceError("create_cards", "failed to save cards", err)
            }

            // 2. Create stats for each card
            for _, card := range cards {
                stats, err := domain.NewUserCardStats(card.UserID, card.ID)
                if err != nil {
                    return NewCardServiceError("create_cards", "failed to create stats object", err)
                }

                err = txStatsRepo.Create(ctx, stats)
                if err != nil {
                    return NewCardServiceError("create_cards", "failed to save stats", err)
                }
            }

            return nil
        },
    )
}
```

### Repository Error Handling

The GetCard method shows proper repository error handling:

```go
// GetCard implements CardService.GetCard
func (s *cardServiceImpl) GetCard(ctx context.Context, cardID uuid.UUID) (*domain.Card, error) {
    log := logger.FromContextOrDefault(ctx, s.logger)

    card, err := s.cardRepo.GetByID(ctx, cardID)
    if err != nil {
        // Check for specific error types
        if store.IsNotFoundError(err) {
            return nil, NewCardServiceError("get_card", "card not found", store.ErrCardNotFound)
        }

        return nil, NewCardServiceError("get_card", "failed to retrieve card", err)
    }

    return card, nil
}
```
