# Database URL Standardization

This document explains the standardization of database URLs in the Scry API project.

## Overview

The project uses database URLs for connecting to PostgreSQL databases in both development and test environments. To ensure consistent behavior, especially in CI environments, these URLs are standardized.

## Implementation

Database URL standardization is now consolidated in a single implementation:

- `internal/ciutil/database.go` - The canonical implementation
- `internal/testdb/url.go` - Now uses the ciutil implementation for consistency

### Standardization Logic

When running in a CI environment, the following standardization rules are applied:

1. **Usernames:** Always standardized to `postgres` in all CI environments
2. **Passwords:**
   - In GitHub Actions: Standardized to `postgres`
   - In other CI environments: Preserved as is
3. **SSL Mode:** Always adds `?sslmode=disable` if no query parameters are present
4. **Host and Port:** Preserved as is, with defaults applied if missing
5. **Database Name:** If missing, defaults to `scry_test`

### Environment Variables

The system looks for database URLs in the following environment variables (in order of priority):

1. `DATABASE_URL`
2. `SCRY_TEST_DB_URL` (preferred variable)
3. `SCRY_DATABASE_URL`

When a URL is standardized, all non-empty environment variables are updated with the standardized URL for consistency.

## Usage

In application code, use `ciutil.GetTestDatabaseURL()` to obtain a standardized database URL for testing. This function handles all the standardization logic and environment variable management.

```go
import (
    "log/slog"
    "github.com/phrazzld/scry-api/internal/ciutil"
)

func MyFunction() {
    logger := slog.Default()
    dbURL := ciutil.GetTestDatabaseURL(logger)
    // Use dbURL for database connections
}
```

## CI Environment Detection

The system automatically detects CI environments using the following environment variables:

- `CI` - Generic CI indicator
- `GITHUB_ACTIONS` - GitHub Actions specific
- `GITLAB_CI` - GitLab CI specific
- `JENKINS_URL` - Jenkins specific
- `TRAVIS` - Travis CI specific
- `CIRCLECI` - Circle CI specific

## Testing

Tests have been updated to verify that URL standardization works correctly across different environments. The key test cases include:

1. Non-CI environments preserve original URLs
2. Generic CI environments standardize usernames to `postgres`
3. GitHub Actions standardizes both username and password to `postgres`
4. All CI environments add `?sslmode=disable` if not present
5. URLs with missing components get appropriate defaults
