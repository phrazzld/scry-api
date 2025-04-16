# T052: Improve Time Handling in Auth Handler

## Task Description
Inject a time source into `AuthHandler` for calculating response `ExpiresAt` instead of using `time.Now()` directly.

## Context

Currently, in the `generateTokenResponse` method of `AuthHandler`, we use `time.Now()` directly to calculate the expiration time for tokens:

```go
// Calculate access token expiration time
expiresAtTime := time.Now().Add(time.Duration(h.authConfig.TokenLifetimeMinutes) * time.Minute)

// Format expiration time in RFC3339 format (standard for JSON API responses)
expiresAt = expiresAtTime.Format(time.RFC3339)
```

This makes testing this method difficult, since we can't easily mock the time in tests.

However, the JWT service already injects a time function for testability:

```go
type hmacJWTService struct {
    signingKey           []byte
    tokenLifetime        time.Duration    // Access token lifetime
    refreshTokenLifetime time.Duration    // Refresh token lifetime
    timeFunc             func() time.Time // Injectable for testing
    clockSkew            time.Duration    // Allowed time difference for validation to handle clock drift
}
```

We need to apply the same pattern to `AuthHandler` for testability and consistency.
