# Transaction Isolation Pattern for Database Testing

This package implements a robust approach to database integration testing using transaction isolation. This pattern allows for:

- **Parallel test execution**: Tests can run simultaneously without interfering with each other
- **No manual cleanup**: All changes are automatically rolled back after each test
- **Test independence**: Each test starts with a known database state
- **Atomic changes**: All database operations in a test are either committed or rolled back together
- **Simplified testing**: No need to mock database operations

## Core Concept

Each test runs in its own database transaction, which is automatically rolled back when the test finishes. This creates complete isolation between tests, allowing them to run in parallel without interfering with each other.

## Usage

### Basic Pattern

```go
func TestSomething(t *testing.T) {
    // Enable parallel testing - safe with transaction isolation
    t.Parallel()

    // Get a database connection
    db, err := testutils.GetTestDB()
    require.NoError(t, err)
    defer testutils.AssertCloseNoError(t, db)

    // Run test in a transaction that will be rolled back automatically
    testutils.WithTx(t, db, func(tx store.DBTX) {
        // Use the transaction for all database operations
        // Any stores created with this transaction share the same transaction

        // Create test data
        userID := testutils.MustInsertUser(ctx, t, tx, "test@example.com")

        // Test your functionality
        // ...

        // No cleanup needed! Everything will be rolled back automatically
    })
}
```

### Using Multiple Stores Together

For tests that need to use multiple stores together (common for integration tests), use the `CreateTestStores` helper:

```go
testutils.WithTx(t, db, func(tx store.DBTX) {
    // Get all stores sharing the same transaction
    stores := testutils.CreateTestStores(tx)

    // Now you can use any of the stores
    user, err := stores.UserStore.Create(ctx, testUser)
    require.NoError(t, err)

    // Use another store with the same transaction
    memo, err := stores.MemoStore.Create(ctx, userMemo)
    require.NoError(t, err)

    // Test business logic that involves multiple stores
    // ...
})
```

## Key Helpers

- `WithTx(t, db, func(tx store.DBTX))`: Runs a function in a transaction that's automatically rolled back
- `GetTestDB()`: Creates a database connection ready for testing
- `CreateTestStores(tx)`: Creates all store implementations with the same transaction
- `MustInsertUser(ctx, t, tx, email)`: Inserts a test user and returns the ID
- `AssertCloseNoError(t, closer)`: Safely closes a resource in defer statements

## Best Practices

1. **Always use `t.Parallel()`** with this pattern - tests are fully isolated
2. **Create all stores from the same transaction** when testing functionality that spans multiple stores
3. **Don't commit the transaction** - let `WithTx` handle rollback automatically
4. **Use the store interfaces** rather than direct SQL when possible
5. **Create helper functions** for common test setup patterns

## Benefits Over Other Testing Approaches

- **Better than table truncation**: No need to truncate tables between tests, much faster
- **Better than mocking**: Tests against real database behavior, not simulated behavior
- **Better than separate databases**: Shared schema, no need to manage multiple databases
- **Better than fixtures**: Dynamic test data specific to each test, no shared state
- **Better than manual isolation**: Automatic rollback prevents test pollution
