package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTimeout = 5 * time.Second

// setupTestDB opens a database connection and ensures a clean test environment
// by dropping and recreating the users table.
func setupTestDB(t *testing.T) *sql.DB {
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get database URL from environment
	dbURL := testutils.GetTestDatabaseURL(t)

	// Connect to the database
	db, err := sql.Open("pgx", dbURL)
	require.NoError(t, err, "Failed to open database connection")

	// Set connection pool parameters
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Create a context with timeout for DB operations
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Ping the database to ensure connection is alive
	err = db.PingContext(ctx)
	require.NoError(t, err, "Failed to ping database")

	// Recreate the test table to ensure a clean state
	// Drop the table if it exists
	_, err = db.ExecContext(ctx, "DROP TABLE IF EXISTS users")
	require.NoError(t, err, "Failed to drop users table")

	// Create the table with the same schema as in migrations
	createTableSQL := `
	CREATE TABLE users (
		id UUID PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		hashed_password TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);

	CREATE INDEX idx_users_email ON users(email);
	`
	_, err = db.ExecContext(ctx, createTableSQL)
	require.NoError(t, err, "Failed to create users table")

	return db
}

// teardownTestDB closes the database connection and performs any needed cleanup
func teardownTestDB(t *testing.T, db *sql.DB) {
	if db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Clean up the test data by dropping the table
		_, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS users")
		if err != nil {
			t.Logf("Warning: Failed to drop users table during cleanup: %v", err)
		}

		err = db.Close()
		if err != nil {
			t.Logf("Warning: Failed to close database connection: %v", err)
		}
	}
}

// createTestUser is a helper function to create a valid test user
//
//nolint:unused
func createTestUser(t *testing.T) *domain.User {
	email := fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
	user, err := domain.NewUser(email, "Password123!")
	require.NoError(t, err, "Failed to create test user")
	return user
}

// insertTestUser inserts a user directly into the database for testing
//
//nolint:unused
func insertTestUser(t *testing.T, db *sql.DB, email string) uuid.UUID {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Generate a unique ID
	id := uuid.New()
	hashedPassword := "$2a$10$abcdefghijklmnopqrstuvwxyz0123456789"

	// Insert the user directly
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, email, hashed_password, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, id, email, hashedPassword)
	require.NoError(t, err, "Failed to insert test user directly")

	return id
}

// getUserByID retrieves a user from the database directly for verification
//
//nolint:unused
func getUserByID(t *testing.T, db *sql.DB, id uuid.UUID) *domain.User {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Query the user
	var user domain.User
	err := db.QueryRowContext(ctx, `
		SELECT id, email, hashed_password, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.HashedPassword, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		require.NoError(t, err, "Failed to query user by ID")
	}

	return &user
}

// countUsers counts the number of users in the database matching certain criteria
//
//nolint:unused
func countUsers(t *testing.T, db *sql.DB, whereClause string, args ...interface{}) int {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	query := "SELECT COUNT(*) FROM users"
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	var count int
	err := db.QueryRowContext(ctx, query, args...).Scan(&count)
	require.NoError(t, err, "Failed to count users")

	return count
}

// TestNewPostgresUserStore verifies the constructor works correctly
func TestNewPostgresUserStore(t *testing.T) {
	// Set up the test database
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Initialize the store
	userStore := postgres.NewPostgresUserStore(db)

	// Assertions
	assert.NotNil(t, userStore, "PostgresUserStore should be created successfully")
	assert.Same(t, db, userStore.DB(), "Store should hold the provided database connection")

	// Verify the implementation satisfies the interface
	var _ store.UserStore = userStore
}

// TestBasicDatabaseConnectivity verifies the test environment works correctly
func TestBasicDatabaseConnectivity(t *testing.T) {
	// Set up the test database
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test basic database connectivity by inserting and querying a sample record
	testUUID := uuid.New()
	email := fmt.Sprintf("integration-test-%s@example.com", testUUID.String()[:8])
	hashedPassword := "hashed_password_placeholder"

	// Direct SQL insert to verify connection
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, email, hashed_password, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, testUUID, email, hashedPassword)
	require.NoError(t, err, "Failed to insert test record directly")

	// Direct SQL query to verify insertion
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&count)
	require.NoError(t, err, "Failed to query test record")
	assert.Equal(t, 1, count, "Should have inserted exactly one record")
}

// TestPostgresUserStore_Create tests the Create method
func TestPostgresUserStore_Create(t *testing.T) {
	t.Skip("Implementing method is a future task")

	// Test Cases (to be implemented):
	// 1. Successful user creation
	// 2. Attempt to create user with existing email (should return ErrEmailExists)
	// 3. Attempt to create user with invalid data (should return validation error)
}

// TestPostgresUserStore_GetByID tests the GetByID method
func TestPostgresUserStore_GetByID(t *testing.T) {
	t.Skip("Implementing method is a future task")

	// Test Cases (to be implemented):
	// 1. Successfully retrieve existing user by ID
	// 2. Attempt to retrieve non-existent user (should return ErrUserNotFound)
}

// TestPostgresUserStore_GetByEmail tests the GetByEmail method
func TestPostgresUserStore_GetByEmail(t *testing.T) {
	t.Skip("Implementing method is a future task")

	// Test Cases (to be implemented):
	// 1. Successfully retrieve existing user by email
	// 2. Attempt to retrieve user with non-existent email (should return ErrUserNotFound)
}

// TestPostgresUserStore_Update tests the Update method
func TestPostgresUserStore_Update(t *testing.T) {
	t.Skip("Implementing method is a future task")

	// Test Cases (to be implemented):
	// 1. Successfully update existing user
	// 2. Attempt to update non-existent user (should return ErrUserNotFound)
	// 3. Attempt to update email to one that already exists (should return ErrEmailExists)
	// 4. Update with password change (should rehash password)
	// 5. Update without password change (should keep existing hash)
}

// TestPostgresUserStore_Delete tests the Delete method
func TestPostgresUserStore_Delete(t *testing.T) {
	t.Skip("Implementing method is a future task")

	// Test Cases (to be implemented):
	// 1. Successfully delete existing user
	// 2. Attempt to delete non-existent user (should return ErrUserNotFound)
}
