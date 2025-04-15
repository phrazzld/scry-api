package testutils_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelpers_CreateTestUser verifies that the CreateTestUser function creates a valid user
func TestHelpers_CreateTestUser(t *testing.T) {
	t.Parallel()

	// Call the helper
	user := testutils.CreateTestUser(t)

	// Verify the user has expected properties
	assert.NotNil(t, user, "User should not be nil")
	assert.NotEqual(t, uuid.Nil, user.ID, "User ID should not be nil")
	assert.Contains(t, user.Email, "test-", "Email should contain 'test-' prefix")
	assert.Contains(t, user.Email, "@example.com", "Email should contain '@example.com' domain")
	assert.NotZero(t, user.CreatedAt, "CreatedAt should not be zero")
	assert.NotZero(t, user.UpdatedAt, "UpdatedAt should not be zero")
}

// TestHelpers_DatabaseOperations tests the database-related helper functions
func TestHelpers_DatabaseOperations(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Connect to database
	dbURL := testutils.MustGetTestDatabaseURL()
	db, err := sql.Open("pgx", dbURL)
	require.NoError(t, err, "Failed to open database connection")
	defer testutils.AssertCloseNoError(t, db)

	// Set up the database schema
	err = testutils.SetupTestDatabaseSchema(db)
	require.NoError(t, err, "Failed to set up test database schema")

	t.Run("MustInsertUser and GetUserByID", func(t *testing.T) {
		t.Parallel() // Parallel testing is safe with WithTx

		testutils.WithTx(t, db, func(tx store.DBTX) {
			ctx := context.Background()

			// Test inserting a user
			email := "test-insert-helper@example.com"
			userID := testutils.MustInsertUser(ctx, t, tx, email)

			// Verify the user was inserted
			user := testutils.GetUserByID(ctx, t, tx, userID)
			require.NotNil(t, user, "User should exist in the database")
			assert.Equal(t, userID, user.ID, "User ID should match")
			assert.Equal(t, email, user.Email, "User email should match")
			assert.NotEmpty(t, user.HashedPassword, "Hashed password should not be empty")
		})
	})

	t.Run("CountUsers", func(t *testing.T) {
		t.Parallel() // Parallel testing is safe with WithTx

		testutils.WithTx(t, db, func(tx store.DBTX) {
			ctx := context.Background()

			// Insert some test users
			email1 := "test-count-1@example.com"
			email2 := "test-count-2@example.com"
			testutils.MustInsertUser(ctx, t, tx, email1)
			testutils.MustInsertUser(ctx, t, tx, email2)

			// Test count all users
			allCount := testutils.CountUsers(ctx, t, tx, "")
			assert.Equal(t, 2, allCount, "Should count all inserted users")

			// Test count with WHERE clause
			filteredCount := testutils.CountUsers(ctx, t, tx, "email = $1", email1)
			assert.Equal(t, 1, filteredCount, "Should count users matching filter")
		})
	})
}

// TestHelpers_CreateTempConfigFile verifies the CreateTempConfigFile function
func TestHelpers_CreateTempConfigFile(t *testing.T) {
	t.Parallel()

	// Test creating a temp config file
	configContent := `
server:
  port: 8080
  log_level: debug
`
	tempDir, cleanup := testutils.CreateTempConfigFile(t, configContent)
	defer cleanup()

	// Verify the file was created
	configPath := tempDir + "/config.yaml"
	fileInfo, err := os.Stat(configPath)
	require.NoError(t, err, "Config file should exist")
	assert.False(t, fileInfo.IsDir(), "Config file should not be a directory")
	assert.Greater(t, fileInfo.Size(), int64(0), "Config file should not be empty")

	// Verify the content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should be able to read config file")
	assert.Equal(t, configContent, string(content), "File should contain expected content")
}
