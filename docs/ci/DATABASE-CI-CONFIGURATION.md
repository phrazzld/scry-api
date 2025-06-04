# Database Configuration in CI

This document describes the standardized approach for database configuration in the Scry API CI pipeline.

## Key Principles

1. **Consistent Credentials**: Always use `postgres` user in CI
2. **Standardized Connection URL**: Use the same format in all places
3. **Enhanced Diagnostics**: Provide detailed logging in CI
4. **Unified Environment Variables**: Keep all database URL environment variables in sync
5. **Explicit Verification**: Verify schema migrations in CI

## Standard Configuration

### Database URL Format

In CI, all database connections should use this standard URL format:
```
postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable
```

### Environment Variables

The following environment variables are used (in order of priority):
1. `DATABASE_URL`
2. `SCRY_TEST_DB_URL`
3. `SCRY_DATABASE_URL`

## Auto-Correction Mechanisms

The system includes several auto-correction mechanisms to ensure consistent database configuration:

1. **Username Standardization**:
   - In `testdb.GetTestDatabaseURL()`, any username that isn't `postgres` is automatically corrected
   - The correction is applied to all database URL environment variables
   - Detailed diagnostic logs are printed when correction occurs

2. **Database Scripts**:
   - `wait-for-db.sh` and `reset-test-db.sh` enforce the standard configuration in CI
   - Scripts handle errors gracefully and provide detailed diagnostics

3. **Migration Verification**:
   - Migration execution in CI includes verification steps
   - Explicit checks for migration table existence and applied migrations
   - Enhanced error reporting to identify common issues

## Testing CI Configuration

A test script is provided to verify database configuration in a CI-like environment:
```bash
./test-ci-database.sh
```

This script simulates the CI environment and tests:
1. Database URL handling with different usernames
2. Standardization of inconsistent environment variables
3. Migration verification

## Implementation Details

### 1. URL Standardization in `testdb/db.go`

The core auto-correction logic is in `GetTestDatabaseURL()`:
- Detects CI environment
- Logs all database URL environment variables
- Parses the URL to extract username
- If username is not `postgres`, standardizes to `postgres`
- Updates all database URL environment variables
- Returns the standardized URL in CI

### 2. Enhanced Migration Diagnostics in `cmd/server/main.go`

The `runMigrations()` function includes:
- Verbose logging for database connection details
- Verification of migration directory and files
- Enhanced error messages for common issues
- Verification of applied migrations after execution

### 3. Utility Script Enhancements

Both database utility scripts include enhanced diagnostics and standardization:

- **wait-for-db.sh**:
  - Shows PostgreSQL version and connection info in CI
  - Forces standard database URL in CI
  - Reports network diagnostics on failure

- **reset-test-db.sh**:
  - Enforces standard database URL in CI
  - Handles custom types (enums) properly
  - Reports detailed diagnostic information on failure

## Common Issues and Resolutions

1. **"role 'root' does not exist"**:
   - Cause: Using incorrect username in database URL
   - Resolution: Username is automatically corrected to `postgres`

2. **"relation schema_migrations does not exist"**:
   - Cause: Migrations not run or failed
   - Resolution: Enhanced migration verification ensures migrations are applied

3. **Project root detection in CI**:
   - Cause: Different environment in CI vs local development
   - Resolution: Improved project root detection with explicit checks for GitHub Actions

4. **Inconsistent environment variables**:
   - Cause: Multiple database URL variables with different values
   - Resolution: Auto-standardization ensures all variables have the same value
