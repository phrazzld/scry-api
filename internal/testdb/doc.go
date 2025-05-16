//go:build integration || test_without_external_deps

// Package testdb provides a comprehensive set of utilities for database testing.
//
// This package implements a transaction-based isolation pattern for database tests,
// which allows tests to run in parallel without interfering with each other. Each
// test runs in its own transaction, which is automatically rolled back when the test
// completes, ensuring that tests do not affect each other's data.
//
// # Core Features
//
// - Transaction Isolation: Run tests in transactions that roll back automatically
// - Connection Management: Simplified database connection setup
// - Migration Utilities: Easily set up test database schema
// - Environment Detection: Skip tests when database is not available
// - Consistent Error Handling: Standard patterns for database error management
//
// # Transaction Isolation Pattern
//
// The primary pattern implemented in this package is transaction-based isolation.
// Each test runs in its own transaction, which is automatically rolled back
// when the test completes. This provides several benefits:
//
//  1. Tests can run in parallel without interfering with each other (t.Parallel())
//  2. No manual cleanup is needed - changes are rolled back automatically
//  3. Tests see a consistent database state (the transaction's snapshot)
//  4. Tests can operate on the same tables/data without conflicts
//  5. Tests run faster since there's no need to truncate tables between tests
//
// # Basic Usage
//
// Here's a simple example of using transaction isolation:
//
//	func TestMyFeature(t *testing.T) {
//	    // Enable parallel testing safely
//	    t.Parallel()
//
//	    // Skip if database is not available
//	    if testdb.ShouldSkipDatabaseTest() {
//	        t.Skip("DATABASE_URL not set - skipping integration test")
//	    }
//
//	    // Get a DB connection with automatic cleanup
//	    db := testdb.GetTestDBWithT(t)
//
//	    // Run your test in a transaction
//	    testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
//	        // Create test store instances with the transaction
//	        userStore := postgres.NewPostgresUserStore(tx, 10)
//
//	        // Use the store to test your functionality
//	        ctx := context.Background()
//	        user, err := userStore.Create(ctx, testUser)
//	        require.NoError(t, err)
//
//	        // No cleanup needed - transaction will be rolled back automatically
//	    })
//	}
//
// # Environment Variables
//
// The package uses the following environment variables:
//
// - DATABASE_URL: Primary connection string
// - SCRY_TEST_DB_URL: Alternative connection string
// - SCRY_DATABASE_URL: Fallback connection string
// - SCRY_PROJECT_ROOT: Explicit project root directory for migrations
//
// # Key Functions
//
// Connection Management:
//
// - GetTestDBWithT(t *testing.T) *sql.DB: Gets a database connection with cleanup registered
// - GetTestDB() (*sql.DB, error): Gets a database connection, returning any error
// - CleanupDB(t *testing.T, db *sql.DB): Safely closes a database connection
//
// Transaction Management:
//
// - WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)): Executes a function in a transaction
// - AssertRollbackNoError(t *testing.T, tx *sql.Tx): Safely rolls back a transaction
//
// Environment Detection:
//
// - IsIntegrationTestEnvironment() bool: Checks if environment is set up for integration tests
// - ShouldSkipDatabaseTest() bool: Determines if database tests should be skipped
// - GetTestDatabaseURL() string: Gets the database URL for testing
//
// Schema Management:
//
// - SetupTestDatabaseSchema(t *testing.T, db *sql.DB): Sets up the database schema using migrations
// - ApplyMigrations(db *sql.DB, migrationsDir string) error: Applies migrations to a database
//
// # Best Practices
//
// 1. Always use transaction isolation via WithTx for database tests
// 2. Enable parallel testing with t.Parallel() when using transactions
// 3. Skip tests when needed using ShouldSkipDatabaseTest()
// 4. Use the testing.T versions of functions when possible for better error reporting
// 5. Keep test lifecycle management with t.Cleanup() registered by GetTestDBWithT
package testdb
