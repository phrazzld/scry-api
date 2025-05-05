//go:build test || integration || test_without_external_deps

package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/stretchr/testify/require"
)

// DefaultJWTConfig returns a standard configuration for JWT authentication suitable for testing.
// This is the single source of truth for JWT test config.
func DefaultJWTConfig() config.AuthConfig {
	return config.AuthConfig{
		JWTSecret:                   "test-jwt-secret-that-is-32-chars-long", // At least 32 chars
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	}
}

// NewTestJWTService creates a JWT service with default configuration for testing.
// This is the recommended way to create a JWT service for tests.
func NewTestJWTService() (JWTService, error) {
	return NewJWTService(DefaultJWTConfig())
}

// MustCreateTestJWTService creates a test JWT service and panics if it fails.
// Useful for test setup where error handling would be verbose.
func MustCreateTestJWTService() JWTService {
	service, err := NewTestJWTService()
	if err != nil {
		// ALLOW-PANIC
		panic(fmt.Sprintf("failed to create test JWT service: %v", err))
	}
	return service
}

// RequireTestJWTService creates a test JWT service and uses require to handle errors.
// This is the recommended way to create a JWT service in tests using testify.
func RequireTestJWTService(t *testing.T) JWTService {
	t.Helper()
	service, err := NewTestJWTService()
	require.NoError(t, err, "Failed to create test JWT service")
	return service
}

// GenerateTokenForTesting creates a JWT token for the specified user ID.
// This is a utility function for tests that need to create tokens without
// having to instantiate a JWT service.
func GenerateTokenForTesting(userID uuid.UUID) (string, error) {
	svc, err := NewTestJWTService()
	if err != nil {
		return "", fmt.Errorf("failed to create JWT service: %w", err)
	}
	return svc.GenerateToken(context.Background(), userID)
}

// GenerateRefreshTokenForTesting creates a refresh token for the specified user ID.
// This is a utility function for tests that need to create refresh tokens without
// having to instantiate a JWT service.
func GenerateRefreshTokenForTesting(userID uuid.UUID) (string, error) {
	svc, err := NewTestJWTService()
	if err != nil {
		return "", fmt.Errorf("failed to create JWT service: %w", err)
	}
	return svc.GenerateRefreshToken(context.Background(), userID)
}

// GenerateAuthHeaderForTesting creates an Authorization header value with Bearer prefix
// containing a valid JWT token for the specified user ID.
func GenerateAuthHeaderForTesting(userID uuid.UUID) (string, error) {
	token, err := GenerateTokenForTesting(userID)
	if err != nil {
		return "", err
	}
	return "Bearer " + token, nil
}

// GenerateAuthHeaderForTestingT is a test helper that creates an Authorization header
// and fails the test if token generation fails.
func GenerateAuthHeaderForTestingT(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	header, err := GenerateAuthHeaderForTesting(userID)
	require.NoError(t, err, "Failed to generate auth header")
	return header
}

// GenerateExpiredRefreshTokenForTesting creates an expired refresh token for testing.
func GenerateExpiredRefreshTokenForTesting(userID uuid.UUID) (string, error) {
	svc, err := NewTestJWTService()
	if err != nil {
		return "", fmt.Errorf("failed to create JWT service: %w", err)
	}
	expiry := time.Now().Add(-1 * time.Hour) // 1 hour in the past
	return svc.GenerateRefreshTokenWithExpiry(context.Background(), userID, expiry)
}
