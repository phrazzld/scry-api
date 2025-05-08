//go:build integration

package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// IsIntegrationTestEnvironment forwards to testdb.IsIntegrationTestEnvironment for compatibility
func IsIntegrationTestEnvironment() bool {
	return testdb.IsIntegrationTestEnvironment()
}

// WithTx executes a test function within a transaction, automatically rolling back
// after the test completes. This is a compatibility function that forwards to testdb.WithTx.
func WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()
	testdb.WithTx(t, db, fn)
}

// GetTestDBWithT returns a database connection for testing (new version with testing.T).
// This is a compatibility function that forwards to testdb.GetTestDBWithT.
func GetTestDBWithT(t *testing.T) *sql.DB {
	t.Helper()
	return testdb.GetTestDBWithT(t)
}

// GetTestDB returns a database connection for testing (original version returning error).
// This maintains the original function signature for backward compatibility.
func GetTestDB() (*sql.DB, error) {
	return testdb.GetTestDB()
}

// SetupTestDatabaseSchemaWithT runs database migrations to set up the test database.
// This is the new version that takes a testing.T parameter.
func SetupTestDatabaseSchemaWithT(t *testing.T, db *sql.DB) {
	t.Helper()
	testdb.SetupTestDatabaseSchema(t, db)
}

// SetupTestDatabaseSchema runs database migrations to set up the test database.
// This maintains the original function signature for backward compatibility.
func SetupTestDatabaseSchema(db *sql.DB) error {
	// We need an implementation that doesn't use testing.T features
	// Get project root - similar to the implementation in testdb
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find go.mod by traversing up directories
	projectRoot := ""
	for {
		if _, err := os.Stat(fmt.Sprintf("%s/go.mod", dir)); err == nil {
			projectRoot = dir
			break
		}

		parent := fmt.Sprintf("%s/..", dir)
		if parent == dir {
			return fmt.Errorf("could not find project root (go.mod file)")
		}
		dir = parent
	}

	// Set up migrations directory
	migrationsDir := fmt.Sprintf("%s/internal/platform/postgres/migrations", projectRoot)
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found at %s: %w", migrationsDir, err)
	}

	// Run migrations
	if err := testdb.ApplyMigrations(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// AssertCloseNoError safely closes a resource, logging any errors.
func AssertCloseNoError(t *testing.T, closer interface{}) {
	t.Helper()

	if closer == nil {
		return
	}

	if db, ok := closer.(*sql.DB); ok {
		testdb.CleanupDB(t, db)
		return
	}

	if c, ok := closer.(interface{ Close() error }); ok {
		if err := c.Close(); err != nil {
			t.Logf("Warning: failed to close resource: %v", err)
		}
	}
}

// MustInsertUser creates a test user in the database and returns the user ID.
// If the optional bcryptCost parameter is not provided, it defaults to 10.
func MustInsertUser(ctx context.Context, t *testing.T, tx *sql.Tx, email string, bcryptCost ...int) uuid.UUID {
	t.Helper()

	// Default bcrypt cost if not provided
	cost := 10
	if len(bcryptCost) > 0 {
		cost = bcryptCost[0]
	}

	// Generate a random UUID for the user
	userID := uuid.New()

	// Hash a default password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpassword123456"), cost)
	require.NoError(t, err, "Failed to hash password")

	// Insert the user directly using SQL
	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO users (id, email, hashed_password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)",
		userID,
		email,
		string(hashedPassword),
		time.Now().UTC(),
		time.Now().UTC(),
	)
	require.NoError(t, err, "Failed to insert test user")

	return userID
}
