# Authentication Package

This package provides JWT-based authentication services for the Scry API.

## Components

- `jwt_service.go` - Defines the JWTService interface and Claims struct
- `jwt_service_impl.go` - Provides the implementation of the JWTService interface
- `jwt_service_mock.go` - Provides a mock implementation for testing
- `test_helpers.go` - Provides helper functions for testing JWT functionality
- `errors.go` - Defines authentication-related errors
- `password.go` - Provides password hashing and verification

## Usage in Tests

When writing tests that require JWT functionality, use the helpers from `test_helpers.go`:

```go
// Create a JWT service for testing
jwtService := auth.RequireTestJWTService(t)

// Generate an auth header with Bearer prefix
authHeader := auth.GenerateAuthHeaderForTestingT(t, userID)

// Create a mock JWT service
mockService := auth.NewMockJWTService()

// Configure the mock to return custom values
mockService.WithTokenError(auth.ErrExpiredToken)
```

## Implementation Notes

1. The JWT service uses HMAC-SHA256 for signing tokens
2. Tokens include standard JWT claims plus application-specific claims
3. Both access tokens and refresh tokens are supported
4. The service supports token validation with configurable clock skew

## Testing

For testing with JWT, we recommend using the following best practices:

1. Use `RequireTestJWTService(t)` to get a real JWT service with default test configuration
2. Use `auth.NewMockJWTService()` when you need to mock specific behaviors
3. Use the helper functions like `GenerateAuthHeaderForTestingT` when you just need tokens

## Deprecation Notice

Some legacy JWT helpers in other packages are now deprecated in favor of these consolidated implementations:

- `internal/testutils/mock_jwt_service.go` - Replaced by `auth.MockJWTService`
- `internal/mocks/jwt_service.go` - Replaced by `auth.MockJWTService`
- `internal/testutils/auth_helpers.go:CreateTestJWTService` - Replaced by `auth.NewTestJWTService`
