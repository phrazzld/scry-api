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
	"strings"
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

// AssertNoErrorLeakage checks that the error does not leak internal database details.
// This is particularly important for testing postgres error handling to ensure sensitive
// database implementation details are not exposed to users.
func AssertNoErrorLeakage(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		return
	}

	errMsg := err.Error()

	// Database specific terms that should not be leaked to users
	sensitiveTerms := []string{
		// PostgreSQL specific
		"postgres", "postgresql", "pq:", "pg:", "pgx:",
		"23505", "23503", "23502", "23514", // PostgreSQL error codes
		"duplicate key", "violates unique constraint",
		"violates foreign key constraint",
		"violates not-null constraint",
		"constraint", "table", "column",

		// SQL specific
		"sql:", "sql.ErrNoRows", "database/sql",
		"query", "syntax error",

		// Internal details
		"position:", "line:", "file:", "detail:", "hint:",
		"internal query:", "where:", "schema",
	}

	for _, term := range sensitiveTerms {
		if strings.Contains(errMsg, term) {
			t.Errorf("Error message leaks internal detail: %q. Full error: %q", term, errMsg)
		}
	}

	// In a production app, also verify it doesn't leak too much technical information
	// by keeping error messages to a reasonable length
	if len(errMsg) >= 200 {
		t.Errorf("Error message is suspiciously long which may indicate leakage of internal details: %q", errMsg)
	}
}
