# CI Failure Audit: Missing testutils Package Functions

## Summary of Failure

The CI build is still failing with a more specific error. The issue is not related to our PostgreSQL configuration anymore, but to missing functions in the `testutils` package. The build is failing because certain test files can't find expected helper functions.

## Detailed Analysis

### Error Message

```
cmd/server/main_integration_test.go:102:12: undefined: testutils.WithTx
cmd/server/main_integration_test.go:292:12: undefined: testutils.WithTx
cmd/server/main_task_test.go:54:25: undefined: testutils.SetupTestDatabaseSchema
cmd/server/main_task_test.go:253:12: undefined: testutils.WithTx
FAIL	github.com/phrazzld/scry-api/cmd/server [build failed]
```

```
FAIL	github.com/phrazzld/scry-api/internal/testutils [build failed]
```

### Root Cause

Our test files in `cmd/server` are trying to use several functions from the `testutils` package:
1. `testutils.WithTx` - A function for transaction-based test isolation
2. `testutils.SetupTestDatabaseSchema` - A function for initializing test database schemas

However, these functions don't exist or can't be found. Looking at the T001-test-report.md we generated earlier, the issue appears to be that the functions have been moved or are only available in a different package.

The problem may be that the transaction isolation utilities have been moved to a subdirectory (`internal/testutils/db/`) while the tests are still trying to import them directly from `testutils`.

## Recommended Fix

### Option 1: Update Import Paths

Update all tests that use these functions to import them from the correct package path:

```go
// Change from
import "github.com/phrazzld/scry-api/internal/testutils"

// To
import "github.com/phrazzld/scry-api/internal/testutils/db"
```

And then update the function calls from `testutils.WithTx(...)` to `db.WithTx(...)`.

### Option 2: Add Forwarding Functions

Add forwarding functions to the `testutils` package that delegate to the new location:

```go
// In internal/testutils/helpers.go
package testutils

import "github.com/phrazzld/scry-api/internal/testutils/db"

// WithTx forwards to db.WithTx for backward compatibility
func WithTx(t *testing.T, dbConn *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
    db.WithTx(t, dbConn, fn)
}

// SetupTestDatabaseSchema forwards to db.SetupTestDatabaseSchema for backward compatibility
func SetupTestDatabaseSchema(t *testing.T, db *sql.DB) {
    db.SetupTestDatabaseSchema(t, db)
}
```

### Option 3: Move Functions Back

Move the functions back to the main `testutils` package or ensure they are accessible through proper exports.

## Implementation Plan

The most straightforward approach is Option 1 - updating import paths in the affected test files:

1. Locate all test files using `testutils.WithTx` and `testutils.SetupTestDatabaseSchema`
2. Update their imports to reference the correct package (`internal/testutils/db`)
3. Update the function calls to use the new namespace (`db.WithTx` instead of `testutils.WithTx`)

## Longer-term Recommendations

1. **Consistent Import Structure**: Establish clear guidelines for where test utilities should be located
2. **Deprecation Notices**: When moving functions between packages, add deprecation notices and forwarding functions
3. **Test Utility Documentation**: Create better documentation for test utilities to make it clear which package to import
4. **Package Structure Review**: Review the overall package structure to ensure logical organization of utilities

This issue highlights the importance of careful refactoring when moving functions between packages, especially in test code.
