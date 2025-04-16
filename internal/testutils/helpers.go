// Package testutils provides a set of standardized helper functions for testing
// across the codebase. These helpers ensure consistent test patterns, particularly
// for database operations (using transaction-based isolation with WithTx),
// test data creation, and environment setup.
//
// Helper functions follow these naming conventions:
// - Create*: Create entities in memory
// - MustInsert*: Insert entities into database with transaction isolation
// - Get*: Retrieve entities from database
// - Count*: Count entities matching criteria
// - SetupEnv: Configure environment variables for testing
// - CreateTempConfigFile: Create temporary configuration files
// - Assert*: Verify conditions and handle errors in tests
package testutils

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// CreateTestUser creates a new valid user with a random email for testing.
// It does not save the user to the database.
func CreateTestUser(t *testing.T) *domain.User {
	t.Helper()
	email := fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
	user, err := domain.NewUser(email, "Password123!")
	require.NoError(t, err, "Failed to create test user")
	return user
}

// MustInsertUser inserts a user into the database for testing.
// It requires a transaction obtained from WithTx to ensure test isolation.
// The function will fail the test if the insert operation fails.
func MustInsertUser(ctx context.Context, t *testing.T, db store.DBTX, email string) uuid.UUID {
	t.Helper()

	// Create a test password that meets validation requirements (12+ chars)
	password := "TestPassword123!"

	// Create a user with the provided email and test password
	user := &domain.User{
		ID:        uuid.New(),
		Email:     email,
		Password:  password,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// For simplicity in this implementation, directly hash the password
	// In a real application, this would be handled by a service
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err, "Failed to hash password")

	// Execute the SQL directly to avoid circular dependencies with postgres package
	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, email, hashed_password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, user.ID, user.Email, string(hashedPassword), user.CreatedAt, user.UpdatedAt)
	require.NoError(t, err, "Failed to insert test user")

	return user.ID
}

// GetUserByID retrieves a user from the database by ID.
// Returns nil if the user does not exist.
func GetUserByID(ctx context.Context, t *testing.T, db store.DBTX, id uuid.UUID) *domain.User {
	t.Helper()

	// Query for the user
	var user domain.User
	err := db.QueryRowContext(ctx, `
		SELECT id, email, hashed_password, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Email,
		&user.HashedPassword,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		require.NoError(t, err, "Failed to query user by ID")
	}

	return &user
}

// CountUsers counts the number of users in the database matching the given criteria.
func CountUsers(ctx context.Context, t *testing.T, db store.DBTX, whereClause string, args ...interface{}) int {
	t.Helper()

	query := "SELECT COUNT(*) FROM users"
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	var count int
	err := db.QueryRowContext(ctx, query, args...).Scan(&count)
	require.NoError(t, err, "Failed to count users")

	return count
}

// CreateTempConfigFile creates a temporary YAML config file with the given content.
// Returns the directory path and a cleanup function.
// The cleanup function is automatically called by t.TempDir() when the test completes.
func CreateTempConfigFile(t *testing.T, content string) (string, func()) {
	t.Helper()

	tempDir := t.TempDir()
	configPath := tempDir + "/config.yaml"

	err := os.WriteFile(configPath, []byte(content), 0600)
	require.NoError(t, err, "Failed to create temporary config file")

	// Return the directory path and a cleanup function
	return tempDir, func() {
		// t.TempDir() handles cleanup automatically
	}
}

// AssertCloseNoError ensures that the Close() method on the provided closer
// executes without error. It uses assert.NoError to allow subsequent defers
// to run even if this one fails (as opposed to using require.NoError which
// would abort the test immediately).
//
// Usage:
//
//	db, err := sql.Open("pgx", dbURL)
//	require.NoError(t, err)
//	defer testutils.AssertCloseNoError(t, db)
func AssertCloseNoError(t *testing.T, closer io.Closer) {
	t.Helper()
	if closer == nil {
		return
	}
	err := closer.Close()
	assert.NoError(t, err, "Deferred Close() failed for %T", closer)
}

// AssertRollbackNoError ensures that the Rollback() method on the provided tx
// executes without error, unless the error is sql.ErrTxDone which indicates
// the transaction was already committed or rolled back.
//
// This is specifically designed for use with SQL transactions, as it includes
// special handling for the common case where a transaction might already be
// committed or rolled back.
//
// Usage:
//
//	tx, err := db.BeginTx(ctx, nil)
//	require.NoError(t, err)
//	defer testutils.AssertRollbackNoError(t, tx)
func AssertRollbackNoError(t *testing.T, tx *sql.Tx) {
	t.Helper()
	if tx == nil {
		return
	}
	err := tx.Rollback()
	if err != nil && !errors.Is(err, sql.ErrTxDone) {
		assert.NoError(t, err, "Failed to rollback transaction")
	}
}

// CreateTestUserStore creates a new PostgresUserStore for testing.
// It uses the given transaction to ensure test isolation.
func CreateTestUserStore(tx store.DBTX) store.UserStore {
	return postgres.NewPostgresUserStore(tx, bcrypt.DefaultCost)
}
