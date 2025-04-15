package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
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
	"golang.org/x/crypto/bcrypt"
)

const testTimeout = 5 * time.Second

// testDB is a package-level variable that holds a shared database connection
// for all tests in this package.
var testDB *sql.DB

// TestMain sets up the database and runs all tests once, rather than for each test.
// This improves performance by running migrations only once for all tests.
func TestMain(m *testing.M) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		os.Exit(0)
	}

	// Connect to database once for all tests
	dbURL := testutils.MustGetTestDatabaseURL()
	var err error
	testDB, err = sql.Open("pgx", dbURL)
	if err != nil {
		fmt.Printf("Failed to open database connection: %v\n", err)
		os.Exit(1)
	}

	// Set connection parameters
	testDB.SetMaxOpenConns(5)
	testDB.SetMaxIdleConns(5)
	testDB.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection with ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := testDB.PingContext(ctx); err != nil {
		fmt.Printf("Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	// Setup database schema using migrations
	if err := testutils.SetupTestDatabaseSchema(testDB); err != nil {
		fmt.Printf("Failed to setup test database schema: %v\n", err)
		os.Exit(1)
	}

	// Run all tests
	exitCode := m.Run()

	// Clean up
	if err := testDB.Close(); err != nil {
		fmt.Printf("Warning: Failed to close database connection: %v\n", err)
	}

	os.Exit(exitCode)
}

// createTestUser is a helper function to create a valid test user
func createTestUser(t *testing.T) *domain.User {
	email := fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
	user, err := domain.NewUser(email, "Password123!")
	require.NoError(t, err, "Failed to create test user")
	return user
}

// insertTestUser inserts a user into the database for testing using PostgresUserStore.Create
func insertTestUser(ctx context.Context, t *testing.T, db store.DBTX, email string) uuid.UUID {
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

	// Create a user store
	userStore := postgres.NewPostgresUserStore(db, bcrypt.DefaultCost)

	// Insert the user using Create method
	err := userStore.Create(ctx, user)
	require.NoError(t, err, "Failed to insert test user using UserStore.Create")

	// Password should be hashed and the plaintext cleared by the Create method
	assert.Empty(t, user.Password, "Plaintext password should be cleared")
	assert.NotEmpty(t, user.HashedPassword, "Hashed password should be set")

	return user.ID
}

// getUserByID retrieves a user from the database using PostgresUserStore.GetByID
func getUserByID(ctx context.Context, t *testing.T, db store.DBTX, id uuid.UUID) *domain.User {
	// Create a user store
	userStore := postgres.NewPostgresUserStore(db, bcrypt.DefaultCost)

	// Retrieve the user using GetByID method
	user, err := userStore.GetByID(ctx, id)

	// Handle the result
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return nil
		}
		require.NoError(t, err, "Failed to retrieve user by ID")
	}

	return user
}

// countUsers counts the number of users in the database matching certain criteria
func countUsers(ctx context.Context, t *testing.T, db store.DBTX, whereClause string, args ...interface{}) int {
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
	t.Parallel() // Enable parallel testing

	testutils.WithTx(t, testDB, func(tx store.DBTX) {
		// Initialize the store with the transaction
		userStore := postgres.NewPostgresUserStore(tx, bcrypt.DefaultCost)

		// Assertions
		assert.NotNil(t, userStore, "PostgresUserStore should be created successfully")
		assert.Same(t, tx, userStore.DB(), "Store should hold the provided database connection")

		// Verify the implementation satisfies the interface
		var _ store.UserStore = userStore
	})
}

// TestBasicDatabaseConnectivity verifies the test environment works correctly
func TestBasicDatabaseConnectivity(t *testing.T) {
	t.Parallel() // Enable parallel testing

	testutils.WithTx(t, testDB, func(tx store.DBTX) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Test basic database connectivity by inserting and querying a sample record
		testUUID := uuid.New()
		email := fmt.Sprintf("integration-test-%s@example.com", testUUID.String()[:8])
		hashedPassword := "hashed_password_placeholder"

		// Direct SQL insert to verify connection
		_, err := tx.ExecContext(ctx, `
			INSERT INTO users (id, email, hashed_password, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
		`, testUUID, email, hashedPassword)
		require.NoError(t, err, "Failed to insert test record directly")

		// Direct SQL query to verify insertion
		var count int
		err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&count)
		require.NoError(t, err, "Failed to query test record")
		assert.Equal(t, 1, count, "Should have inserted exactly one record")
	})
}

// TestPostgresUserStore_Create tests the Create method
func TestPostgresUserStore_Create(t *testing.T) {
	t.Parallel() // Enable parallel testing

	testutils.WithTx(t, testDB, func(tx store.DBTX) {
		// Create a new user store
		userStore := postgres.NewPostgresUserStore(tx, bcrypt.DefaultCost)

		// Test Case 1: Successful user creation
		t.Run("Successful user creation", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

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
			dbUser := getUserByID(ctx, t, tx, user.ID)
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
			t.Parallel() // Enable parallel subtests

			// Create a test user
			email := fmt.Sprintf("duplicate-%s@example.com", uuid.New().String()[:8])

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Insert the first user directly into the database
			insertTestUser(ctx, t, tx, email)

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
			count := countUsers(ctx, t, tx, "email = $1", email)
			assert.Equal(t, 1, count, "There should still be only one user with this email")
		})

		// Test Case 3: Attempt to create user with invalid data
		t.Run("Invalid user data", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

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
			count := countUsers(ctx, t, tx, "email = $1", "not-an-email")
			assert.Equal(t, 0, count, "No user should be created with invalid email")
		})

		// Test Case 4: Attempt to create user with weak password
		t.Run("Weak password", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

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
			count := countUsers(ctx, t, tx, "email = $1", user.Email)
			assert.Equal(t, 0, count, "No user should be created with weak password")
		})

		// Test Case 5: Attempt to create user with password that's too long
		t.Run("Password too long", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a user with valid email but password that's too long
			// Create a password longer than 72 characters (bcrypt's limit)
			tooLongPassword := strings.Repeat("p", 73)
			user := &domain.User{
				ID:        uuid.New(),
				Email:     fmt.Sprintf("long-password-%s@example.com", uuid.New().String()[:8]),
				Password:  tooLongPassword,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}

			// Call the Create method
			err := userStore.Create(ctx, user)

			// Verify the result
			assert.Error(t, err, "Creating user with too long password should fail")
			assert.Equal(t, domain.ErrPasswordTooLong, err, "Error should be ErrPasswordTooLong")

			// Verify no user was created
			count := countUsers(ctx, t, tx, "email = $1", user.Email)
			assert.Equal(t, 0, count, "No user should be created with too long password")
		})
	})
}
