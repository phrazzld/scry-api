# Test Integration Analysis Report

## Build Tag Consistency

After analyzing the codebase, I found that most database test files correctly use the `integration` build tag, but there are a few inconsistencies:

### Files Missing Integration Build Tag

1. `/Users/phaedrus/Development/scry/scry-api/internal/testutils/db_test.go`
   - Contains database tests with calls to `sql.Open`, `db.QueryRowContext`
   - Uses transaction isolation testing but lacks integration tag

2. `/Users/phaedrus/Development/scry/scry-api/infrastructure/terraform/test/terraform_test.go`
   - Tests database connectivity with real infrastructure
   - Contains `sql.Open` and database queries

3. `/Users/phaedrus/Development/scry/scry-api/internal/service/card_review/service_test.go`
   - Contains database transaction interfaces and mocks
   - Mocks database access but should still be marked as integration

4. `/Users/phaedrus/Development/scry/scry-api/infrastructure/local_dev/local_postgres_test.go`
   - Tests Docker-based PostgreSQL setup
   - Contains direct database connections via `sql.Open`

## Transaction Isolation Usage

The project has a robust transaction isolation pattern for database tests, but there are some inconsistencies:

### Patterns Used

1. **Preferred Approach**: Using `testutils.WithTx` with individual database connections per test
2. **Legacy Approach**: Using a shared `testDB` with `BeginTx` managed directly in the test

### Implementation Groups

#### Files Using Proper Transaction Isolation

Most store test files in `internal/platform/postgres` and service layer transaction tests properly use the `WithTx` pattern:
- `card_store_test.go`
- `memo_store_test.go`
- `card_store_crud_test.go`
- `card_store_getnext_test.go`
- `stats_store_test.go`
- `user_store_test.go`
- `card_service_tx_test.go`
- `memo_service_tx_test.go`
- `user_service_tx_test.go`

#### Files Using Alternative Isolation Methods

Error leakage test files share a common `testDB` connection established in `TestMain` and manually create transactions:
- `user_store_error_leakage_test.go`
- `card_store_error_leakage_test.go`
- `memo_store_error_leakage_test.go`
- `stats_store_error_leakage_test.go`
- `task_store_error_leakage_test.go`
- `error_leakage_test.go`

## Recommendations for Future Work

1. **Build Tags**:
   - Add the `//go:build integration` tag to the files listed above
   - Standardize on the `//go:build integration` format across all files

2. **Transaction Isolation**:
   - Migrate all test files to use `testutils.WithTx` for consistency
   - Update error leakage tests to use the standard pattern instead of the shared `testDB` approach
   - Phase out direct use of `BeginTx` and manual rollback

3. **Documentation**:
   - Update the comments in `testutils/db.go` to emphasize the preferred pattern
   - Add warnings to deprecated functions like `ResetTestData`

These changes will ensure more consistent test behavior and better isolation between tests, particularly in CI environments.
