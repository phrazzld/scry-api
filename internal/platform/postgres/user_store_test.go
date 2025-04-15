package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
func insertTestUser(ctx context.Context, t *testing.T, db *sql.DB, email string) uuid.UUID {
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
func getUserByID(ctx context.Context, t *testing.T, db *sql.DB, id uuid.UUID) *domain.User {
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
func countUsers(ctx context.Context, t *testing.T, db *sql.DB, whereClause string, args ...interface{}) int {
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
	// Set up the test database
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Create a new user store
	userStore := postgres.NewPostgresUserStore(db)

	// Test Case 1: Successful user creation
	t.Run("Successful user creation", func(t *testing.T) {
		// Create a test user
		user := createTestUser(t)

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Call the Create method
		err := userStore.Create(ctx, user)

		// Verify the result
		require.NoError(t, err, "User creation should succeed")

		// Verify the user was inserted into the database
		dbUser := getUserByID(ctx, t, db, user.ID)
		require.NotNil(t, dbUser, "User should exist in the database")
		assert.Equal(t, user.ID, dbUser.ID, "User ID should match")
		assert.Equal(t, user.Email, dbUser.Email, "User email should match")
		assert.NotEmpty(t, dbUser.HashedPassword, "Hashed password should not be empty")
		assert.Empty(t, user.Password, "Plaintext password should be cleared")

		// Verify timestamps
		assert.False(t, dbUser.CreatedAt.IsZero(), "CreatedAt should not be zero")
		assert.False(t, dbUser.UpdatedAt.IsZero(), "UpdatedAt should not be zero")
	})

	// Test Case 2: Attempt to create user with existing email
	t.Run("Duplicate email", func(t *testing.T) {
		// Create a test user
		email := fmt.Sprintf("duplicate-%s@example.com", uuid.New().String()[:8])

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Insert the first user directly into the database
		insertTestUser(ctx, t, db, email)

		// Create a second user with the same email
		user, err := domain.NewUser(email, "Password123!")
		require.NoError(t, err, "Creating user struct should succeed")

		// Call the Create method
		err = userStore.Create(ctx, user)

		// Verify the result
		assert.ErrorIs(
			t,
			err,
			store.ErrEmailExists,
			"Creating user with duplicate email should fail with ErrEmailExists",
		)

		// Verify there's still only one user with this email
		count := countUsers(ctx, t, db, "email = $1", email)
		assert.Equal(t, 1, count, "There should still be only one user with this email")
	})

	// Test Case 3: Attempt to create user with invalid data
	t.Run("Invalid user data", func(t *testing.T) {
		// Create a test user with invalid email
		user, err := domain.NewUser("not-an-email", "Password123!")
		require.Error(t, err, "Creating user with invalid email should fail validation")
		assert.Nil(t, user, "User should be nil after validation failure")

		// Create context
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Since we could not create a user with an invalid email through the constructor,
		// let's create a valid user first and then modify it to have invalid data
		user = createTestUser(t)
		user.Email = "not-an-email" // This will fail validation

		// Call the Create method
		err = userStore.Create(ctx, user)

		// Verify the result
		assert.Error(t, err, "Creating user with invalid email should fail")
		assert.Equal(t, domain.ErrInvalidEmail, err, "Error should be ErrInvalidEmail")

		// Verify no user was created
		count := countUsers(ctx, t, db, "email = $1", "not-an-email")
		assert.Equal(t, 0, count, "No user should be created with invalid email")
	})

	// Test Case 4: Attempt to create user with weak password
	t.Run("Weak password", func(t *testing.T) {
		// Create context
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Create a user with valid email but weak password
		user := &domain.User{
			ID:        uuid.New(),
			Email:     fmt.Sprintf("weak-password-%s@example.com", uuid.New().String()[:8]),
			Password:  "password", // Too short (less than 12 characters)
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		// Call the Create method
		err := userStore.Create(ctx, user)

		// Verify the result
		assert.Error(t, err, "Creating user with weak password should fail")
		assert.Equal(t, domain.ErrPasswordTooShort, err, "Error should be ErrPasswordTooShort")

		// Verify no user was created
		count := countUsers(ctx, t, db, "email = $1", user.Email)
		assert.Equal(t, 0, count, "No user should be created with weak password")
	})
}

// TestPostgresUserStore_GetByID tests the GetByID method
func TestPostgresUserStore_GetByID(t *testing.T) {
	// Set up the test database
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Create a new user store
	userStore := postgres.NewPostgresUserStore(db)

	// Test Case 1: Successfully retrieve existing user by ID
	t.Run("Successfully retrieve existing user", func(t *testing.T) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Insert a test user directly into the database
		email := fmt.Sprintf("getbyid-test-%s@example.com", uuid.New().String()[:8])
		userId := insertTestUser(ctx, t, db, email)

		// Call the GetByID method
		user, err := userStore.GetByID(ctx, userId)

		// Verify the result
		require.NoError(t, err, "GetByID should succeed for existing user")
		require.NotNil(t, user, "Retrieved user should not be nil")
		assert.Equal(t, userId, user.ID, "User ID should match")
		assert.Equal(t, email, user.Email, "User email should match")
		assert.NotEmpty(t, user.HashedPassword, "Hashed password should not be empty")
		assert.Empty(t, user.Password, "Plaintext password should be empty")
		assert.False(t, user.CreatedAt.IsZero(), "CreatedAt should not be zero")
		assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt should not be zero")
	})

	// Test Case 2: Attempt to retrieve non-existent user
	t.Run("Non-existent user", func(t *testing.T) {
		// Generate a random UUID that doesn't exist in the database
		nonExistentID := uuid.New()

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Call the GetByID method
		user, err := userStore.GetByID(ctx, nonExistentID)

		// Verify the result
		assert.Error(t, err, "GetByID should return error for non-existent user")
		assert.ErrorIs(t, err, store.ErrUserNotFound, "Error should be ErrUserNotFound")
		assert.Nil(t, user, "User should be nil for non-existent ID")
	})
}

// TestPostgresUserStore_GetByEmail tests the GetByEmail method
func TestPostgresUserStore_GetByEmail(t *testing.T) {
	// Set up the test database
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Create a new user store
	userStore := postgres.NewPostgresUserStore(db)

	// Test Case 1: Successfully retrieve existing user by email
	t.Run("Successfully retrieve existing user", func(t *testing.T) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Insert a test user directly into the database
		email := fmt.Sprintf("getbyemail-test-%s@example.com", uuid.New().String()[:8])
		userId := insertTestUser(ctx, t, db, email)

		// Call the GetByEmail method
		user, err := userStore.GetByEmail(ctx, email)

		// Verify the result
		require.NoError(t, err, "GetByEmail should succeed for existing user")
		require.NotNil(t, user, "Retrieved user should not be nil")
		assert.Equal(t, userId, user.ID, "User ID should match")
		assert.Equal(t, email, user.Email, "User email should match")
		assert.NotEmpty(t, user.HashedPassword, "Hashed password should not be empty")
		assert.Empty(t, user.Password, "Plaintext password should be empty")
		assert.False(t, user.CreatedAt.IsZero(), "CreatedAt should not be zero")
		assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt should not be zero")
	})

	// Test Case 2: Attempt to retrieve user with non-existent email
	t.Run("Non-existent email", func(t *testing.T) {
		// Use an email that doesn't exist in the database
		nonExistentEmail := fmt.Sprintf("nonexistent-%s@example.com", uuid.New().String())

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Call the GetByEmail method
		user, err := userStore.GetByEmail(ctx, nonExistentEmail)

		// Verify the result
		assert.Error(t, err, "GetByEmail should return error for non-existent email")
		assert.ErrorIs(t, err, store.ErrUserNotFound, "Error should be ErrUserNotFound")
		assert.Nil(t, user, "User should be nil for non-existent email")
	})

	// Test Case 3: Case insensitivity for email matching
	t.Run("Case insensitive email matching", func(t *testing.T) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Insert a test user with lowercase email
		email := fmt.Sprintf("casesensitive-%s@example.com", uuid.New().String()[:8])
		userId := insertTestUser(ctx, t, db, email)

		// Query with uppercase email
		upperEmail := strings.ToUpper(email)
		user, err := userStore.GetByEmail(ctx, upperEmail)

		// Verify the result (should find the user despite case difference)
		require.NoError(t, err, "GetByEmail should be case insensitive")
		require.NotNil(t, user, "Retrieved user should not be nil")
		assert.Equal(t, userId, user.ID, "User ID should match")
		assert.Equal(t, email, user.Email, "User email should match original case")
	})
}

// TestPostgresUserStore_Update tests the Update method
func TestPostgresUserStore_Update(t *testing.T) {
	// Set up the test database
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Create a new user store
	userStore := postgres.NewPostgresUserStore(db)

	// Test Case 1: Successfully update existing user with a new email but same password
	t.Run("Update email only", func(t *testing.T) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Insert a test user directly into the database
		oldEmail := fmt.Sprintf("update-test-email-%s@example.com", uuid.New().String()[:8])
		userId := insertTestUser(ctx, t, db, oldEmail)

		// Fetch the user to get current hashed password and timestamps
		originalUser := getUserByID(ctx, t, db, userId)
		require.NotNil(t, originalUser, "User should exist before update")
		oldHash := originalUser.HashedPassword

		// Create an updated user (change email but not password)
		newEmail := fmt.Sprintf("updated-email-%s@example.com", uuid.New().String()[:8])
		updatedUser := &domain.User{
			ID:        userId,
			Email:     newEmail,
			Password:  "", // No password update
			CreatedAt: originalUser.CreatedAt,
			UpdatedAt: originalUser.UpdatedAt, // Will be updated by the method
		}

		// Call the Update method
		err := userStore.Update(ctx, updatedUser)

		// Verify the result
		require.NoError(t, err, "Update should succeed for existing user")

		// Verify the user was updated in the database
		updatedDbUser := getUserByID(ctx, t, db, userId)
		require.NotNil(t, updatedDbUser, "User should still exist after update")
		assert.Equal(t, userId, updatedDbUser.ID, "User ID should not change")
		assert.Equal(t, newEmail, updatedDbUser.Email, "Email should be updated")
		assert.Equal(t, oldHash, updatedDbUser.HashedPassword, "Password hash should remain unchanged")
		assert.True(t, updatedDbUser.UpdatedAt.After(originalUser.UpdatedAt), "UpdatedAt should be updated")
		assert.Equal(t, originalUser.CreatedAt, updatedDbUser.CreatedAt, "CreatedAt should not change")
	})

	// Test Case 2: Successfully update existing user with a new password but same email
	t.Run("Update password only", func(t *testing.T) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Insert a test user directly into the database
		email := fmt.Sprintf("update-test-pwd-%s@example.com", uuid.New().String()[:8])
		userId := insertTestUser(ctx, t, db, email)

		// Fetch the user to get current hashed password and timestamps
		originalUser := getUserByID(ctx, t, db, userId)
		require.NotNil(t, originalUser, "User should exist before update")
		oldHash := originalUser.HashedPassword

		// Create an updated user (change password but not email)
		newPassword := "NewPassword123!"
		updatedUser := &domain.User{
			ID:        userId,
			Email:     email,       // Same email
			Password:  newPassword, // New password
			CreatedAt: originalUser.CreatedAt,
			UpdatedAt: originalUser.UpdatedAt, // Will be updated by the method
		}

		// Call the Update method
		err := userStore.Update(ctx, updatedUser)

		// Verify the result
		require.NoError(t, err, "Update should succeed for existing user")

		// Verify the user was updated in the database
		updatedDbUser := getUserByID(ctx, t, db, userId)
		require.NotNil(t, updatedDbUser, "User should still exist after update")
		assert.Equal(t, userId, updatedDbUser.ID, "User ID should not change")
		assert.Equal(t, email, updatedDbUser.Email, "Email should remain unchanged")
		assert.NotEqual(t, oldHash, updatedDbUser.HashedPassword, "Password hash should be updated")
		assert.True(t, updatedDbUser.UpdatedAt.After(originalUser.UpdatedAt), "UpdatedAt should be updated")
		assert.Equal(t, originalUser.CreatedAt, updatedDbUser.CreatedAt, "CreatedAt should not change")
		assert.Empty(t, updatedUser.Password, "Plaintext password should be cleared")
	})

	// Test Case 3: Attempt to update non-existent user
	t.Run("Non-existent user", func(t *testing.T) {
		// Generate a random UUID that doesn't exist in the database
		nonExistentID := uuid.New()

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Create a user update with the non-existent ID
		user := &domain.User{
			ID:        nonExistentID,
			Email:     "nonexistent@example.com",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		// Call the Update method
		err := userStore.Update(ctx, user)

		// Verify the result
		assert.Error(t, err, "Update should return error for non-existent user")
		assert.ErrorIs(t, err, store.ErrUserNotFound, "Error should be ErrUserNotFound")
	})

	// Test Case 4: Attempt to update email to one that already exists
	t.Run("Duplicate email", func(t *testing.T) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Insert two test users
		existingEmail := fmt.Sprintf("existing-email-%s@example.com", uuid.New().String()[:8])
		existingID := insertTestUser(ctx, t, db, existingEmail)

		updateEmail := fmt.Sprintf("update-email-%s@example.com", uuid.New().String()[:8])
		updateID := insertTestUser(ctx, t, db, updateEmail)

		// Get original user data
		originalUser := getUserByID(ctx, t, db, updateID)
		require.NotNil(t, originalUser, "User should exist before update")

		// Create an updated user (change email to one that already exists)
		updatedUser := &domain.User{
			ID:        updateID,
			Email:     existingEmail, // Email of the other user - should cause conflict
			CreatedAt: originalUser.CreatedAt,
			UpdatedAt: originalUser.UpdatedAt,
		}

		// Call the Update method
		err := userStore.Update(ctx, updatedUser)

		// Verify the result
		assert.Error(t, err, "Update should return error for duplicate email")
		assert.ErrorIs(t, err, store.ErrEmailExists, "Error should be ErrEmailExists")

		// Verify the user was not updated
		updatedDbUser := getUserByID(ctx, t, db, updateID)
		require.NotNil(t, updatedDbUser, "User should still exist")
		assert.Equal(t, updateEmail, updatedDbUser.Email, "Email should not be changed")

		// Verify the other user was not affected
		otherUser := getUserByID(ctx, t, db, existingID)
		require.NotNil(t, otherUser, "Other user should still exist")
		assert.Equal(t, existingEmail, otherUser.Email, "Other user's email should not change")
	})

	// Test Case 5: Update with invalid data
	t.Run("Invalid data", func(t *testing.T) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Insert a test user
		email := fmt.Sprintf("valid-email-%s@example.com", uuid.New().String()[:8])
		userId := insertTestUser(ctx, t, db, email)

		// Get original user data
		originalUser := getUserByID(ctx, t, db, userId)
		require.NotNil(t, originalUser, "User should exist before update")

		// Create an updated user with invalid email
		updatedUser := &domain.User{
			ID:        userId,
			Email:     "invalid-email", // Invalid email format
			CreatedAt: originalUser.CreatedAt,
			UpdatedAt: originalUser.UpdatedAt,
		}

		// Call the Update method
		err := userStore.Update(ctx, updatedUser)

		// Verify the result
		assert.Error(t, err, "Update should return error for invalid data")
		assert.Equal(t, domain.ErrInvalidEmail, err, "Error should be ErrInvalidEmail")

		// Verify the user was not updated
		updatedDbUser := getUserByID(ctx, t, db, userId)
		require.NotNil(t, updatedDbUser, "User should still exist")
		assert.Equal(t, email, updatedDbUser.Email, "Email should not be changed")
	})
}

// TestPostgresUserStore_Delete tests the Delete method
func TestPostgresUserStore_Delete(t *testing.T) {
	// Set up the test database
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Create a new user store
	userStore := postgres.NewPostgresUserStore(db)

	// Test Case 1: Successfully delete existing user
	t.Run("Successfully delete existing user", func(t *testing.T) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Insert a test user directly into the database
		email := fmt.Sprintf("delete-test-%s@example.com", uuid.New().String()[:8])
		userId := insertTestUser(ctx, t, db, email)

		// Verify user exists before deletion
		beforeCount := countUsers(ctx, t, db, "id = $1", userId)
		assert.Equal(t, 1, beforeCount, "User should exist before deletion")

		// Call the Delete method
		err := userStore.Delete(ctx, userId)

		// Verify the result
		require.NoError(t, err, "Delete should succeed for existing user")

		// Verify user no longer exists
		afterCount := countUsers(ctx, t, db, "id = $1", userId)
		assert.Equal(t, 0, afterCount, "User should not exist after deletion")
	})

	// Test Case 2: Attempt to delete non-existent user
	t.Run("Non-existent user", func(t *testing.T) {
		// Generate a random UUID that doesn't exist in the database
		nonExistentID := uuid.New()

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Call the Delete method
		err := userStore.Delete(ctx, nonExistentID)

		// Verify the result
		assert.Error(t, err, "Delete should return error for non-existent user")
		assert.ErrorIs(t, err, store.ErrUserNotFound, "Error should be ErrUserNotFound")
	})
}
