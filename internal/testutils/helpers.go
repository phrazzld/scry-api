//go:build !compatibility && ignore_redeclarations

// Package testutils provides a set of standardized helper functions for testing
// across the codebase. These helpers ensure consistent test patterns, particularly
// for database operations (using transaction-based isolation with WithTx),
// test data creation, and environment setup.
//
// Helper functions follow these naming conventions:
// - Create*: Create entities in memory
// - MustInsert*: Insert entities into database with transaction isolation
// - Get*: Retrieve entities from database

package testutils

// - Count*: Count entities matching criteria
// - SetupEnv: Configure environment variables for testing
// - CreateTempConfigFile: Create temporary configuration files
// - Assert*: Verify conditions and handle errors in tests

// This file provides general test utilities.
// It should be used in preference to the compatibility.go file where possible.

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
//
// If hashedPassword is provided, it will be used directly. Otherwise, a UserStore will be created
// to properly handle password hashing through the domain layer.
//
// The bcryptCost parameter controls the computational cost of password hashing.
// If it's <= 0, bcrypt.MinCost (4) will be used for faster test execution.
func MustInsertUser(
	ctx context.Context,
	t *testing.T,
	db store.DBTX,
	email string,
	bcryptCost int,
	hashedPassword ...string,
) uuid.UUID {
	t.Helper()

	// Generate a unique user ID
	userID := uuid.New()
	now := time.Now().UTC()

	// Set default test password
	password := "TestPassword123!"

	// If bcryptCost is not specified or invalid, use bcrypt.MinCost for faster tests
	if bcryptCost <= 0 {
		bcryptCost = bcrypt.MinCost
	}

	// If there are two approaches:
	// 1. If a hashed password is provided, insert directly with SQL
	// 2. If no hashed password is provided, use UserStore.Create to handle hashing properly
	if len(hashedPassword) > 0 && hashedPassword[0] != "" {
		// Direct SQL approach with pre-hashed password
		_, err := db.ExecContext(ctx, `
			INSERT INTO users (id, email, hashed_password, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, email, hashedPassword[0], now, now)
		require.NoError(t, err, "Failed to insert test user with pre-hashed password")
	} else {
		// UserStore approach - let the store handle domain validation and password hashing
		// Create a user with the provided email and test password
		user := &domain.User{
			ID:        userID,
			Email:     email,
			Password:  password, // Plain password - UserStore.Create will hash it
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Create a UserStore to handle the proper creation logic
		userStore := postgres.NewPostgresUserStore(db, bcryptCost)
		err := userStore.Create(ctx, user)
		require.NoError(t, err, "Failed to create test user using UserStore")
	}

	return userID
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
func CountUsers(
	ctx context.Context,
	t *testing.T,
	db store.DBTX,
	whereClause string,
	args ...interface{},
) int {
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
//
// The bcryptCost parameter controls the computational cost of password hashing.
// If it's <= 0, bcrypt.MinCost (4) will be used for faster test execution.
func CreateTestUserStore(tx store.DBTX, bcryptCost int) store.UserStore {
	// If bcryptCost is not specified or invalid, use bcrypt.MinCost for faster tests
	if bcryptCost <= 0 {
		bcryptCost = bcrypt.MinCost
	}
	return postgres.NewPostgresUserStore(tx, bcryptCost)
}

// Stores represents all database store implementations.
// This struct provides a convenient way to access all stores
// that share the same transaction for test isolation.
type Stores struct {
	UserStore          store.UserStore
	MemoStore          store.MemoStore
	CardStore          store.CardStore
	UserCardStatsStore store.UserCardStatsStore
}

// CreateTestStores creates all store implementations using a single transaction.
// This ensures that all changes made through these stores will be rolled back
// when the test completes, providing proper test isolation.
//
// The bcryptCost parameter controls the computational cost of password hashing.
// If it's <= 0, bcrypt.MinCost (4) will be used for faster test execution.
//
// Usage:
//
//	testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
//	    stores := testutils.CreateTestStores(tx, bcrypt.MinCost)
//
//	    // Use any of the stores
//	    user, err := stores.UserStore.Create(ctx, testUser)
//	    require.NoError(t, err)
//
//	    // Use another store with the same transaction
//	    memo, err := stores.MemoStore.Create(ctx, testMemo)
//	    require.NoError(t, err)
//
//	    // All changes will be rolled back automatically
//	})
func CreateTestStores(tx store.DBTX, bcryptCost int) Stores {
	// If bcryptCost is not specified or invalid, use bcrypt.MinCost for faster tests
	if bcryptCost <= 0 {
		bcryptCost = bcrypt.MinCost
	}

	return Stores{
		UserStore:          postgres.NewPostgresUserStore(tx, bcryptCost),
		MemoStore:          postgres.NewPostgresMemoStore(tx, nil),
		CardStore:          postgres.NewPostgresCardStore(tx, nil),
		UserCardStatsStore: postgres.NewPostgresUserCardStatsStore(tx, nil),
	}
}
