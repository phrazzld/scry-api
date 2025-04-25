# T026 Inventory: Direct Error Logging Locations

The following locations in the API handlers directly log error objects without using `redact.Error()`. These instances need to be updated in T027.

## internal/api/auth_handler.go

1. **Line 42-44**:
```go
log.Error("failed to generate access token",
    slog.String("error", err.Error()),
    slog.String("token_type", "access"),
    slog.Int("lifetime_minutes", h.authConfig.TokenLifetimeMinutes))
```

2. **Line 52-55**:
```go
log.Error("failed to generate refresh token",
    slog.String("error", err.Error()),
    slog.String("token_type", "refresh"),
    slog.Int("lifetime_minutes", h.authConfig.RefreshTokenLifetimeMinutes))
```

## internal/api/card_handler.go

1. **Line 155-158**:
```go
log.Warn("invalid request format",
    slog.String("error", err.Error()),
    slog.String("user_id", userID.String()),
    slog.String("card_id", cardID.String()))
```

2. **Line 165-168**:
```go
log.Warn("validation error",
    slog.String("error", err.Error()),
    slog.String("user_id", userID.String()),
    slog.String("card_id", cardID.String()))
```

## internal/api/middleware/auth.go

1. **Line 55**:
```go
slog.Error("failed to validate token", "error", err)
```

## Summary

Total instances found: 5

These instances need to be updated to use `redact.Error(err)` instead of directly logging `err.Error()` or passing `err` directly to the logger.
