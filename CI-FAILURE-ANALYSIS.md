# CI Failure Analysis

## Build Information
- **Repository**: phrazzld/scry-api
- **Branch**: feature/card-management-api
- **PR**: #26
- **Run ID**: 14953175861
- **Commit**: 6a787187d1a93370276df7be6b4d4450faf8204a

## Failure Summary
The CI pipeline is failing in the "Test" job. The database migration step fails with the error:

```
ERROR: type "memo_status" already exists (SQLSTATE 42710)
```

## Detailed Error Analysis
During the test phase, the CI workflow:
1. Resets the test database
2. Attempts to apply migrations using `go run cmd/server/main.go -migrate=up`
3. Fails when executing the `20250415000002_create_memos_table.sql` migration

The specific error occurs when trying to create the `memo_status` ENUM type, which apparently already exists in the database despite the database reset.

### Relevant Log Excerpt:
```
ERROR 20250415000002_create_memos_table.sql: failed to run SQL migration: failed to execute SQL query
"CREATE TYPE memo_status AS ENUM (
    'pending',
    'processing',
    'completed',
    'completed_with_errors',
    'failed'
);"
ERROR: type "memo_status" already exists (SQLSTATE 42710)
```

## Root Cause Analysis
The issue appears to be related to how the database is being reset in the CI environment. When the database is reset, the tables are dropped, but custom types like the `memo_status` ENUM are not being dropped properly.

The reset script appears to be dropping tables but not custom types (ENUMs, etc.) that exist in the database schema. When migrations run again, they attempt to create these types, resulting in the "already exists" error.

## Affected Components
- Database migration system
- CI pipeline configuration
- The `reset-test-db.sh` script

## Potential Solutions
1. Modify the database reset script to include dropping custom types (ENUMs) before running migrations
2. Update the migration to handle the case where the type already exists (IF NOT EXISTS)
3. Use a more comprehensive database reset approach in the CI pipeline that ensures all schema objects (including types) are properly dropped
