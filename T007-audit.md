# Transaction Error Handling Audit (T007)

This document provides a comprehensive audit of error handling patterns in service transaction callbacks throughout the codebase, as required by ticket T007.

## Overview

The application uses `store.RunInTransaction` to execute database operations atomically. Each service has its own pattern for error handling within these transactions. This audit identifies the current patterns to inform standardization efforts.

## Error Handling Patterns Found

### 1. CardService

**File:** `internal/service/card_service.go`

| Method | Error Handling Pattern |
|--------|------------------------|
| `CreateCards` | **Wrapped errors** - All store errors are wrapped using `NewCardServiceError` with operation name and message |
| `PostponeCard` | **Mixed** - Within transaction: wraps errors with `NewCardServiceError`. Outside transaction: returns the error directly (`return nil, err`) |

**Example (CreateCards):**
```go
return NewCardServiceError("create_cards", "failed to save cards", err)
```

**Example (PostponeCard):**
```go
// Inside transaction
return NewCardServiceError("postpone_card", "failed to calculate postponed review", err)

// Outside transaction
if err != nil {
    return nil, err // Error already wrapped and logged in transaction
}
```

### 2. CardReviewService

**File:** `internal/service/card_review/service_impl.go`

| Method | Error Handling Pattern |
|--------|------------------------|
| `SubmitAnswer` | **Mixed** - Returns sentinel errors directly (e.g., `ErrCardNotFound`), but wraps store errors with `NewSubmitAnswerError`. Does another check outside the transaction but returns the error directly without wrapping. |

**Example:**
```go
// Inside transaction: return sentinel error directly
if errors.Is(err, store.ErrCardNotFound) {
    return ErrCardNotFound
}
// Inside transaction: wrap error
return NewSubmitAnswerError("failed to retrieve stats", err)

// Outside transaction: return unwrapped error
if err != nil {
    // If the error is already one of our service errors, pass it through
    if errors.Is(err, ErrCardNotFound) ||
        errors.Is(err, ErrCardNotOwned) ||
        errors.Is(err, ErrInvalidAnswer) {
        return nil, err
    }
    return nil, err // No need to wrap, we're using CustomErrors now
}
```

### 3. MemoService

**File:** `internal/service/memo_service.go`

| Method | Error Handling Pattern |
|--------|------------------------|
| `CreateMemoAndEnqueueTask` | **Simple wrapper** - Uses generic `fmt.Errorf("failed to create memo: %w", err)` for wrapping errors |
| `UpdateMemoStatus` | **Simple wrapper** - Uses generic `fmt.Errorf("failed to update memo status to %s: %w", status, err)` with context |

**Example:**
```go
// Inside transaction
return fmt.Errorf("failed to save memo status %s: %w", status, err)

// Outside transaction
return nil, fmt.Errorf("failed to create memo: %w", err)
```

### 4. UserService

**File:** `internal/service/user_service.go`

| Method | Error Handling Pattern |
|--------|------------------------|
| `CreateUser` | **Simple wrapper** - Uses `fmt.Errorf("failed to create user: %w", err)` both inside and outside transaction |
| `UpdateUserEmail` | **Simple wrapper** - Uses `fmt.Errorf("failed to update user email: %w", err)` with specific context |
| `UpdateUserPassword` | **Simple wrapper** - Uses `fmt.Errorf("failed to update user password: %w", err)` with specific context |
| `DeleteUser` | **Simple wrapper** - Uses `fmt.Errorf("failed to delete user: %w", err)` with specific context |

**Example:**
```go
// Inside transaction
return fmt.Errorf("failed to retrieve user for update: %w", err)

// Outside transaction
return nil, fmt.Errorf("failed to create user: %w", err)
```

## Sentinel Error Usage

| Service | Sentinel Error Usage |
|---------|----------------------|
| CardService | Returns service-specific sentinel errors directly (e.g., `ErrNotOwned`) when appropriate |
| CardReviewService | Returns service-specific sentinel errors directly (e.g., `ErrCardNotFound`, `ErrCardNotOwned`) |
| MemoService | Does not appear to use sentinel errors |
| UserService | Does not directly return sentinel errors, but checks for store-level sentinel errors like `store.ErrEmailExists` |

## Inconsistencies and Observations

1. **Custom Error Types vs fmt.Errorf**:
   - `CardService` and `CardReviewService` use custom error types (`CardServiceError`, `SubmitAnswerError`)
   - `MemoService` and `UserService` use `fmt.Errorf` with `%w` for wrapping

2. **Sentinel Error Handling**:
   - Services handle sentinel errors differently:
     - Some return sentinel errors directly
     - Some wrap sentinel errors with custom error types
     - Some check for sentinel errors with `errors.Is` but then wrap them anyway

3. **Error Return Patterns from Transactions**:
   - `CardService.PostponeCard` returns the transaction error directly: `return nil, err`
   - `CardReviewService.SubmitAnswer` has conditional logic for different error types outside the transaction
   - `MemoService` and `UserService` consistently wrap errors from transactions

4. **Error Context**:
   - Some services include detailed context in error messages (operation name, memo ID, status, etc.)
   - Others use more generic messages

## Recommended Standardization Approach

Based on this audit, a standardized approach should:

1. Choose between custom error types vs `fmt.Errorf` for error wrapping
2. Establish clear rules for when to return sentinel errors directly vs wrapped
3. Standardize error logging patterns
4. Ensure consistent error handling both inside and outside transactions

This will be addressed in the next task (T008).
