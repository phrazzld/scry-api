# CI Failure Audit - Missing DB Test Helper Functions

## Issue Description

After fixing the memo test helper functions, we have new CI failures related to missing database test utility functions:

```
undefined: testutils.WithTx
undefined: testutils.GetTestDB
undefined: testutils.SetupTestDatabaseSchema
```

These functions are used in multiple test files:
- `internal/service/card_service_operations_test.go`
- `internal/platform/postgres/user_store_update_delete_test.go`
- `internal/platform/postgres/user_store_test.go`
- `internal/platform/postgres/user_store_get_test.go`

## Root Cause Analysis

Similar to the previous issue, these test helper functions exist in the testutils package but with incompatible build tags. The error suggests that the database-related helper functions have been moved to a subpackage (`testutils/db`) as mentioned in a previous audit, but some test files are still referencing them directly from the `testutils` package.

Looking at the error patterns:

1. The tests are using `testutils.WithTx` and `testutils.GetTestDB` directly
2. These functions might have been moved to `testutils/db` package in a refactoring
3. Some tests were updated to use `testdb.WithTx` and `testdb.GetTestDBWithT` but others weren't

This is consistent with our findings in `memo_store_test.go` which correctly uses `testdb.WithTx` and `testdb.GetTestDBWithT`.

## Implementation Plan

We have two approaches to fix this issue:

### Option 1: Add Compatibility Layer for Database Functions

Create a compatibility layer in the testutils package with the `integration` build tag that forwards to the appropriate testdb functions:

```go
//go:build integration

package testutils

import (
    "database/sql"
    "testing"

    "github.com/phrazzld/scry-api/internal/testdb"
)

// WithTx is a compatibility function that forwards to testdb.WithTx
func WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
    testdb.WithTx(t, db, fn)
}

// GetTestDB is a compatibility function that forwards to testdb.GetTestDBWithT
func GetTestDB(t *testing.T) *sql.DB {
    return testdb.GetTestDBWithT(t)
}

// SetupTestDatabaseSchema is a compatibility function that forwards to testdb.SetupSchema
func SetupTestDatabaseSchema(t *testing.T, db *sql.DB) {
    testdb.SetupSchema(t, db)
}
```

### Option 2: Update All Test References

Update all test files to use the new `testdb` package directly instead of `testutils`:

1. Change `testutils.WithTx` to `testdb.WithTx`
2. Change `testutils.GetTestDB` to `testdb.GetTestDBWithT`
3. Change `testutils.SetupTestDatabaseSchema` to `testdb.SetupSchema`

## Recommended Approach

Option 1 is quicker and has less risk of introducing new issues. It provides backward compatibility while the codebase is transitioning to the new structure. This aligns with our incremental approach to fixing CI issues.

In the future, a more comprehensive refactoring could move all tests to use the new packages directly, but for now, a compatibility layer will ensure CI passes.

## Next Steps

1. Create a compatibility layer in `internal/testutils/db_compat.go` with the integration build tag
2. Implement the forwarding functions to the testdb package
3. Verify CI passes after the changes
