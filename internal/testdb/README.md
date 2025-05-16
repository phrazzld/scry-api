# Database Testing Utilities

This package provides a comprehensive set of utilities for database testing in the Scry API. It implements a transaction-based isolation pattern that allows tests to run in parallel without interfering with each other.

## Core Features

- **Transaction Isolation**: Run tests in transactions that automatically roll back
- **Connection Management**: Simplified database connection setup with sane defaults
- **Migration Utilities**: Easily set up test database schema
- **Environment Detection**: Skip tests when database is not available
- **Consistent Error Handling**: Standard patterns for database error management

## Usage Patterns

### Basic Test with Transaction Isolation

```go
func TestFeature(t *testing.T) {
    // Tests can run in parallel safely with transaction isolation
    t.Parallel()

    // Skip if database is not available
    if testdb.ShouldSkipDatabaseTest() {
        t.Skip("DATABASE_URL not set - skipping integration test")
    }

    // Get a test database connection
    db := testdb.GetTestDBWithT(t)

    // Run test in a transaction that's automatically rolled back
    testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
        // Create stores with the transaction
        userStore := postgres.NewPostgresUserStore(tx, 10)
        memoStore := postgres.NewPostgresMemoStore(tx, nil)

        // Test your functionality using these stores
        ctx := context.Background()

        // Create a test user
        user, err := domain.NewUser("test@example.com", "password123456")
        require.NoError(t, err)
        require.NoError(t, userStore.Create(ctx, user))

        // Create a memo for the user
        memo, err := domain.NewMemo(user.ID, "Test memo")
        require.NoError(t, err)
        require.NoError(t, memoStore.Create(ctx, memo))

        // Run assertions on the expected behavior
        // ...

        // No cleanup needed - transaction will be rolled back automatically
    })
}
```

### Database Configuration

The package automatically detects database configuration from environment variables:

- `DATABASE_URL` - Primary connection string
- `SCRY_TEST_DB_URL` - Alternative connection string
- `SCRY_DATABASE_URL` - Fallback connection string

You can also use `SCRY_PROJECT_ROOT` to explicitly set the project root directory for migrations.

## Key Functions

### Connection Management

- `GetTestDBWithT(t *testing.T) *sql.DB` - Gets a database connection with cleanup registered
- `GetTestDB() (*sql.DB, error)` - Gets a database connection, returning any error
- `CleanupDB(t *testing.T, db *sql.DB)` - Safely closes a database connection

### Transaction Management

- `WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx))` - Executes a function in a transaction
- `AssertRollbackNoError(t *testing.T, tx *sql.Tx)` - Safely rolls back a transaction

### Environment Detection

- `IsIntegrationTestEnvironment() bool` - Checks if environment is set up for integration tests
- `ShouldSkipDatabaseTest() bool` - Determines if database tests should be skipped
- `GetTestDatabaseURL() string` - Gets the database URL for testing

### Schema Management

- `SetupTestDatabaseSchema(t *testing.T, db *sql.DB)` - Sets up the database schema using migrations
- `ApplyMigrations(db *sql.DB, migrationsDir string) error` - Applies migrations to a database

## Best Practices

1. **Always use transaction isolation** via `WithTx` for database tests
2. **Enable parallel testing** with `t.Parallel()` when using transactions
3. **Skip tests when needed** using `ShouldSkipDatabaseTest()`
4. **Use the testing.T versions** of functions when possible for better error reporting
5. **Keep test lifecycle management** with `t.Cleanup()` registered by `GetTestDBWithT`

## Migration from Old Patterns

If you're updating tests that use the old patterns from the `testutils` package:

1. Replace imports:
   ```go
   // Old
   import "github.com/phrazzld/scry-api/internal/testutils"

   // New
   import "github.com/phrazzld/scry-api/internal/testdb"
   ```

2. Update function calls:
   ```go
   // Old
   db := testutils.GetTestDBWithT(t)
   testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
       // ...
   })

   // New
   db := testdb.GetTestDBWithT(t)
   testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
       // ...
   })
   ```

## Implementation Notes

- All functions with `WithT` in the name take a `testing.T` parameter and register cleanups
- All functions use `t.Helper()` when appropriate to improve error reporting
- Database connections have sensible defaults for testing (max connections, timeout, etc.)
- Migrations are only run once per test run for efficiency
