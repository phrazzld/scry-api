# T028 Inventory: Direct Error Response Generation

The following locations directly generate error responses instead of using the centralized error handling helpers `HandleAPIError` and `HandleValidationError`.

## auth_handler.go

### 1. Register Method (Lines 119-121)
**Type**: Format error (validation)
**Current**:
```go
if err := shared.DecodeJSON(r, &req); err != nil {
    shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, "Invalid request format", err)
    return
}
```
**Should use**: `HandleValidationError`

### 2. Register Method (Lines 125-128)
**Type**: Validation error
**Current**:
```go
if err := shared.Validate.Struct(req); err != nil {
    sanitizedError := SanitizeValidationError(err)
    shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, sanitizedError, err)
    return
}
```
**Should use**: `HandleValidationError`

### 3. Register Method (Lines 132-141)
**Type**: Domain error
**Current**:
```go
if err != nil {
    // Map domain error to appropriate message and status
    statusCode := MapErrorToStatusCode(err)
    safeMessage := GetSafeErrorMessage(err)
    if safeMessage == "An unexpected error occurred" {
        safeMessage = "Invalid user data"
    }
    shared.RespondWithErrorAndLog(w, r, statusCode, safeMessage, err)
    return
}
```
**Should use**: `HandleAPIError` with default message "Invalid user data"

### 4. Register Method (Lines 145-149)
**Type**: Storage error
**Current**:
```go
if err := h.userStore.Create(r.Context(), user); err != nil {
    statusCode := MapErrorToStatusCode(err)
    safeMessage := GetSafeErrorMessage(err)
    shared.RespondWithErrorAndLog(w, r, statusCode, safeMessage, err)
    return
}
```
**Should use**: `HandleAPIError` with default message

### 5. Register Method (Lines 154-157)
**Type**: Token generation error
**Current**:
```go
if err != nil {
    shared.RespondWithErrorAndLog(w, r, http.StatusInternalServerError,
        "Failed to generate authentication tokens", err)
    return
}
```
**Should use**: `HandleAPIError` with default message "Failed to generate authentication tokens"

### 6. RefreshToken Method (Lines 175-177)
**Type**: Format error (validation)
**Current**:
```go
if err := shared.DecodeJSON(r, &req); err != nil {
    shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, "Invalid request format", err)
    return
}
```
**Should use**: `HandleValidationError`

### 7. RefreshToken Method (Lines 181-184)
**Type**: Validation error
**Current**:
```go
if err := shared.Validate.Struct(req); err != nil {
    sanitizedError := SanitizeValidationError(err)
    shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, sanitizedError, err)
    return
}
```
**Should use**: `HandleValidationError`

### 8. RefreshToken Method (Lines 189-194)
**Type**: Token validation error
**Current**:
```go
if err != nil {
    // Map different error types to appropriate status codes and messages
    statusCode := MapErrorToStatusCode(err)
    safeMessage := GetSafeErrorMessage(err)
    shared.RespondWithErrorAndLog(w, r, statusCode, safeMessage, err)
    return
}
```
**Should use**: `HandleAPIError` with default message

### 9. RefreshToken Method (Lines 207-210)
**Type**: Token generation error
**Current**:
```go
if err != nil {
    shared.RespondWithErrorAndLog(w, r, http.StatusInternalServerError,
        "Failed to generate new authentication tokens", err)
    return
}
```
**Should use**: `HandleAPIError` with default message "Failed to generate new authentication tokens"

### 10. Login Method (Lines 226-228)
**Type**: Format error (validation)
**Current**:
```go
if err := shared.DecodeJSON(r, &req); err != nil {
    shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, "Invalid request format", err)
    return
}
```
**Should use**: `HandleValidationError`

### 11. Login Method (Lines 232-235)
**Type**: Validation error
**Current**:
```go
if err := shared.Validate.Struct(req); err != nil {
    sanitizedError := SanitizeValidationError(err)
    shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, sanitizedError, err)
    return
}
```
**Should use**: `HandleValidationError`

### 12. Login Method (Lines 240-256)
**Type**: Authentication error
**Current**:
```go
if err != nil {
    if errors.Is(err, store.ErrUserNotFound) {
        // Use generic error message for security (don't reveal if email exists)
        // Elevate to WARN level as repeated auth failures are operationally important
        shared.RespondWithErrorAndLog(
            w,
            r,
            http.StatusUnauthorized,
            "Invalid credentials",
            err,
            shared.WithElevatedLogLevel(),
        )
        return
    }
    shared.RespondWithErrorAndLog(w, r, http.StatusInternalServerError,
        "Failed to authenticate user", err)
    return
}
```
**Should use**:
- For ErrUserNotFound: `HandleAPIError` with default message "Invalid credentials" and `WithElevatedLogLevel()` option
- For other errors: `HandleAPIError` with default message "Failed to authenticate user"

### 13. Login Method (Lines 260-271)
**Type**: Password verification error
**Current**:
```go
if err := h.passwordVerifier.Compare(user.HashedPassword, req.Password); err != nil {
    // Use same generic error message as above for security
    // Elevate to WARN level as repeated auth failures are operationally important
    shared.RespondWithErrorAndLog(
        w,
        r,
        http.StatusUnauthorized,
        "Invalid credentials",
        err,
        shared.WithElevatedLogLevel(),
    )
    return
}
```
**Should use**: `HandleAPIError` with default message "Invalid credentials" and `WithElevatedLogLevel()` option

### 14. Login Method (Lines 276-279)
**Type**: Token generation error
**Current**:
```go
if err != nil {
    shared.RespondWithErrorAndLog(w, r, http.StatusInternalServerError,
        "Failed to generate authentication tokens", err)
    return
}
```
**Should use**: `HandleAPIError` with default message "Failed to generate authentication tokens"

## card_handler.go

### 1. GetNextReviewCard Method (Lines 61-64)
**Type**: Authentication error
**Current**:
```go
if !ok || userID == uuid.Nil {
    log.Warn("user ID not found or invalid in request context")
    shared.RespondWithError(w, r, http.StatusUnauthorized, "User ID not found or invalid")
    return
}
```
**Should use**: `HandleAPIError` with default message "User ID not found or invalid"

### 2. GetNextReviewCard Method (Lines 80-93)
**Type**: General error
**Current**:
```go
if err != nil {
    // Use our new error handling helper methods
    statusCode := MapErrorToStatusCode(err)
    safeMessage := GetSafeErrorMessage(err)

    // For generic server errors in GetNextReviewCard, use a specific message
    if statusCode == http.StatusInternalServerError &&
        !errors.Is(err, card_review.ErrNoCardsDue) {
        safeMessage = "Failed to get next review card"
    }

    // Log the full error details but only send sanitized message to client
    shared.RespondWithErrorAndLog(w, r, statusCode, safeMessage, err)
    return
}
```
**Should use**: `HandleAPIError` with default message "Failed to get next review card"

### 3. SubmitAnswer Method (Lines 131-134)
**Type**: Validation error (missing ID)
**Current**:
```go
if pathCardID == "" {
    log.Warn("card ID not found in URL path")
    shared.RespondWithError(w, r, http.StatusBadRequest, "Card ID is required")
    return
}
```
**Should use**: `HandleAPIError` with default message "Card ID is required"

### 4. SubmitAnswer Method (Lines 139-142)
**Type**: Validation error (invalid UUID)
**Current**:
```go
if err != nil {
    log.Warn("invalid card ID format", slog.String("card_id", pathCardID))
    shared.RespondWithError(w, r, http.StatusBadRequest, "Invalid card ID format")
    return
}
```
**Should use**: `HandleAPIError` with default message "Invalid card ID format"

### 5. SubmitAnswer Method (Lines 147-150)
**Type**: Authentication error
**Current**:
```go
if !ok || userID == uuid.Nil {
    log.Warn("user ID not found or invalid in request context")
    shared.RespondWithError(w, r, http.StatusUnauthorized, "User ID not found or invalid")
    return
}
```
**Should use**: `HandleAPIError` with default message "User ID not found or invalid"

### 6. SubmitAnswer Method (Lines 155-161)
**Type**: Format error (validation)
**Current**:
```go
if err := shared.DecodeJSON(r, &req); err != nil {
    log.Warn("invalid request format",
        slog.String("error", redact.Error(err)),
        slog.String("user_id", userID.String()),
        slog.String("card_id", cardID.String()))
    shared.RespondWithError(w, r, http.StatusBadRequest, "Invalid request format")
    return
}
```
**Should use**: `HandleValidationError`

### 7. SubmitAnswer Method (Lines 165-186)
**Type**: Validation error
**Current**:
```go
if err := shared.Validate.Struct(req); err != nil {
    log.Warn("validation error",
        slog.String("error", redact.Error(err)),
        slog.String("user_id", userID.String()),
        slog.String("card_id", cardID.String()))

    // Use our sanitized validation error format
    sanitizedError := SanitizeValidationError(err)

    // For the validation error test cases, ensure we use "Validation error" as the message
    if strings.Contains(r.URL.Path, "/answer") &&
        (req.Outcome == "" ||
            (req.Outcome != "" &&
                req.Outcome != "again" &&
                req.Outcome != "hard" &&
                req.Outcome != "good" &&
                req.Outcome != "easy")) {
        sanitizedError = "Validation error"
    }

    shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, sanitizedError, err)
    return
}
```
**Should use**: `HandleValidationError`

### 8. SubmitAnswer Method (Lines 201-213)
**Type**: General error
**Current**:
```go
if err != nil {
    // Map to appropriate status code and get sanitized message
    statusCode := MapErrorToStatusCode(err)
    safeMessage := GetSafeErrorMessage(err)

    // For generic server errors in SubmitAnswer, use a specific message
    if statusCode == http.StatusInternalServerError {
        safeMessage = "Failed to submit answer"
    }

    // Log the full error but only send sanitized message to client
    shared.RespondWithErrorAndLog(w, r, statusCode, safeMessage, err)
    return
}
```
**Should use**: `HandleAPIError` with default message "Failed to submit answer"

## memo_handler.go

### 1. CreateMemo Method (Lines 56-59)
**Type**: Authentication error
**Current**:
```go
if !ok || userID == uuid.Nil {
    log.Warn("user ID not found or invalid in request context")
    shared.RespondWithError(w, r, http.StatusUnauthorized, "Authentication required")
    return
}
```
**Should use**: `HandleAPIError` with default message "Authentication required"

### 2. CreateMemo Method (Lines 64-66)
**Type**: Format error (validation)
**Current**:
```go
if err := shared.DecodeJSON(r, &req); err != nil {
    shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, "Invalid request format", err)
    return
}
```
**Should use**: `HandleValidationError`

### 3. CreateMemo Method (Lines 70-74)
**Type**: Validation error
**Current**:
```go
if err := shared.Validate.Struct(req); err != nil {
    // Sanitize validation error message
    sanitizedError := SanitizeValidationError(err)
    shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, sanitizedError, err)
    return
}
```
**Should use**: `HandleValidationError`

### 4. CreateMemo Method (Lines 79-86)
**Type**: General error
**Current**:
```go
if err != nil {
    // Map error to appropriate status code and get sanitized message
    statusCode := MapErrorToStatusCode(err)
    safeMessage := GetSafeErrorMessage(err)

    // Log the full error details but only send sanitized message to client
    shared.RespondWithErrorAndLog(w, r, statusCode, safeMessage, err)
    return
}
```
**Should use**: `HandleAPIError` with default message

## middleware/auth.go

### 1. Authenticate Method (Lines 33-35)
**Type**: Authentication error
**Current**:
```go
if authHeader == "" {
    shared.RespondWithError(w, r, http.StatusUnauthorized, "Authorization header required")
    return
}
```
**Should use**: `HandleAPIError` with default message "Authorization header required"

### 2. Authenticate Method (Lines 40-42)
**Type**: Authentication error
**Current**:
```go
if len(parts) != 2 || parts[0] != "Bearer" {
    shared.RespondWithError(w, r, http.StatusUnauthorized, "Invalid authorization format")
    return
}
```
**Should use**: `HandleAPIError` with default message "Invalid authorization format"

### 3. Authenticate Method (Lines 50-63)
**Type**: Authentication error
**Current**:
```go
if err != nil {
    switch err {
    case auth.ErrExpiredToken:
        shared.RespondWithError(w, r, http.StatusUnauthorized, "Token expired")
    case auth.ErrInvalidToken:
        shared.RespondWithError(w, r, http.StatusUnauthorized, "Invalid token")
    default:
        slog.Error("failed to validate token", "error", redact.Error(err))
        shared.RespondWithError(
            w,
            r,
            http.StatusInternalServerError,
            "Authentication error",
        )
    }
    return
}
```
**Should use**:
- For ErrExpiredToken: `HandleAPIError` with default message "Token expired"
- For ErrInvalidToken: `HandleAPIError` with default message "Invalid token"
- For other errors: `HandleAPIError` with default message "Authentication error"

## Summary

Total instances found: 26

These instances need to be updated to use:
- `HandleValidationError` for validation-related errors
- `HandleAPIError` for general errors, with appropriate default messages

This will ensure consistent error handling across the codebase and leverage the centralized error handling mechanism.
