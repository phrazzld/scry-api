//go:build integration && !test_without_external_deps && !integration_test_internal

// Package testutils provides test utilities and helpers for the application.
// This file provides critical functions needed for postgres integration tests
// while avoiding conflicts with other implementations.
//
// IMPORTANT: This file uses a specific build tag combination to ensure these functions
// are available for integration tests but do not conflict with other files.
package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/testdb"
)

// IsIntegrationTestEnvironment returns true if the environment is configured
// for running integration tests with a database connection.
// Integration tests should check this and skip if not in an integration test environment.
func IsIntegrationTestEnvironment() bool {
	return testdb.IsIntegrationTestEnvironment()
}

// MustGetTestDatabaseURL returns the database URL for tests
// This implementation is for backward compatibility
func MustGetTestDatabaseURL() string {
	dbURL := testdb.GetTestDatabaseURL()
	if dbURL == "" {
		// ALLOW-PANIC
		panic("DATABASE_URL environment variable is required for integration tests")
	}
	return dbURL
}

// WithTx executes a test function within a transaction, automatically rolling back
// after the test completes. This implementation is specifically for integration tests.
func WithTx(t *testing.T, dbConn *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()
	testdb.WithTx(t, dbConn, fn)
}

// SetupTestDatabaseSchema initializes the database schema using project migrations.
// This implementation is specifically for postgres integration tests.
func SetupTestDatabaseSchema(dbConn *sql.DB) error {
	return testdb.ApplyMigrations(dbConn, "./internal/platform/postgres/migrations")
}

// GetTestDBWithT returns a database connection for testing.
// This implementation is specifically for postgres integration tests.
func GetTestDBWithT(t *testing.T) *sql.DB {
	t.Helper()
	return testdb.GetTestDBWithT(t)
}

// CreateTestJWTService creates a real JWT service for testing with a pre-configured secret and expiration.
func CreateTestJWTService() (auth.JWTService, error) {
	// Create minimal auth config with values valid for testing
	authConfig := config.AuthConfig{
		JWTSecret:                   "test-jwt-secret-that-is-32-chars-long", // At least 32 chars
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	}

	return auth.NewJWTService(authConfig)
}

// GenerateAuthHeader creates an Authorization header value with a valid JWT token for testing.
func GenerateAuthHeader(userID uuid.UUID) (string, error) {
	jwtService, err := CreateTestJWTService()
	if err != nil {
		return "", fmt.Errorf("failed to create test JWT service: %w", err)
	}

	token, err := jwtService.GenerateToken(context.Background(), userID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return "Bearer " + token, nil
}

// GenerateRefreshTokenWithExpiry generates a refresh token with a custom expiration time.
// This is useful for testing token expiration scenarios.
func GenerateRefreshTokenWithExpiry(t *testing.T, userID uuid.UUID, expiry time.Time) (string, error) {
	t.Helper()

	jwtService, err := CreateTestJWTService()
	if err != nil {
		return "", fmt.Errorf("failed to create test JWT service: %w", err)
	}

	token, err := jwtService.GenerateRefreshTokenWithExpiry(context.Background(), userID, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token with custom expiry: %w", err)
	}

	return token, nil
}
