//go:build compatibility

// This file provides a compatibility layer to ease migration to the new
// package structure. It should only be used during the migration period
// and will be removed once all tests are updated to use the new structure.

package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/testutils/api"
	"github.com/phrazzld/scry-api/internal/testutils/db"
)

//
// COMPATIBILITY LAYER
//
// These functions provide a compatibility layer to ease migration to the new
// package structure. They proxy calls to the appropriate sub-packages.
// This allows for gradual migration of tests to use the new package structure
// without breaking existing tests.
//

// GetTestDBWithT returns a database connection for testing.
// Compatibility function that delegates to db.GetTestDBWithT.
func GetTestDBWithT(t *testing.T) *sql.DB {
	t.Helper()
	return db.GetTestDBWithT(t)
}

// GetTestDB is the original version that returns an error rather than using t.Helper
// Compatibility function that delegates to db.GetTestDB.
func GetTestDB() (*sql.DB, error) {
	return db.GetTestDB()
}

// SetupTestDatabaseSchema initializes the database schema using project migrations.
// Compatibility function that delegates to db.SetupTestDatabaseSchema.
func SetupTestDatabaseSchema(dbConn *sql.DB) error {
	return db.SetupTestDatabaseSchema(dbConn)
}

// WithTx runs a test function with transaction-based isolation.
// Compatibility function that delegates to db.WithTx.
func WithTx(t *testing.T, dbConn *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()
	db.WithTx(t, dbConn, fn)
}

// CleanupDB properly closes a database connection and logs any errors.
// Compatibility function that delegates to db.CleanupDB.
func CleanupDB(t *testing.T, dbConn *sql.DB) {
	t.Helper()
	db.CleanupDB(t, dbConn)
}

// ResetTestData truncates all test tables to ensure test isolation.
// Compatibility function that delegates to db.ResetTestData.
func ResetTestData(dbConn *sql.DB) error {
	return db.ResetTestData(dbConn)
}

// AssertRollbackNoError attempts to roll back a transaction and logs an error if it fails.
// Compatibility function that delegates to db.AssertRollbackNoError.
func AssertRollbackNoError(t *testing.T, tx *sql.Tx) {
	t.Helper()
	db.AssertRollbackNoError(t, tx)
}

// CreateTestUser creates a test user in the database within the given transaction
// Compatibility function that delegates to api.CreateTestUser.
func CreateTestUser(t *testing.T, tx *sql.Tx) uuid.UUID {
	t.Helper()
	return api.CreateTestUser(t, tx)
}

// CreateTestCard creates a test card in the database within the given transaction
// Compatibility function that delegates to api.CreateTestCard.
func CreateTestCard(t *testing.T, tx *sql.Tx, userID uuid.UUID) *domain.Card {
	t.Helper()
	return api.CreateTestCard(t, tx, userID)
}

// GetCardByID retrieves a card by its ID from the database within the given transaction.
// Compatibility function that delegates to api.GetCardByID.
func GetCardByID(tx *sql.Tx, cardID uuid.UUID) (*domain.Card, error) {
	return api.GetCardByID(tx, cardID)
}

// GetAuthToken generates an authentication token for testing.
// Compatibility function that delegates to api.GetAuthToken.
func GetAuthToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	return api.GetAuthToken(t, userID)
}

// GetUserCardStats retrieves user card statistics for a given card and user.
// Compatibility function that delegates to api.GetUserCardStats.
func GetUserCardStats(t *testing.T, tx *sql.Tx, userID, cardID uuid.UUID) *domain.UserCardStats {
	t.Helper()
	return api.GetUserCardStats(t, tx, userID, cardID)
}

// RunInTransaction is an alias for WithTx to maintain compatibility with existing code.
// This is used in card_service_tx_test.go
func RunInTransaction(t *testing.T, db *sql.DB, ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	t.Helper()

	// Begin a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Make sure the transaction is rolled back when done
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	// Run the function with the transaction
	if err := fn(ctx, tx); err != nil {
		return err
	}

	// If we got here, no error occurred in the function, so commit
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
