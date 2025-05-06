# CI Configuration Fix

## Problem

The CI build for PR #26 was failing at the database migration step with configuration validation errors. After analyzing the code, we found that:

1. The application requires all configuration settings to be valid, even for running migrations.
2. While we correctly set up the PostgreSQL container, we were not providing all required environment variables.
3. The application expects environment variables to have the `SCRY_` prefix (e.g., `SCRY_DATABASE_URL` instead of `DATABASE_URL`).

## Solution Implemented

We've updated the GitHub Actions workflow (`.github/workflows/ci.yml`) to include all required environment variables with the correct `SCRY_` prefix for both the migration and test steps:

1. Added `SCRY_DATABASE_URL` (in addition to the existing `DATABASE_URL` and `SCRY_TEST_DB_URL`)
2. Added all required configuration values for:
   - Authentication (`SCRY_AUTH_*`)
   - LLM integration (`SCRY_LLM_*`)
   - Server configuration (`SCRY_SERVER_*`)
   - Task processing (`SCRY_TASK_*`)

These changes allow the migration step to successfully validate the configuration and apply database migrations before running the tests.

## Longer-term Recommendations

For a more robust solution in the future, consider:

1. **Refactor the Migration Command**: Create a lightweight configuration loader specifically for migrations that only validates database-related config. This would make migrations less coupled to other application components.

2. **Improve Config Validation**: Modify the config validation to make certain fields optional or only validate them in specific contexts.

3. **Add Better Error Messages**: Enhance error messages for missing configuration, especially for migration commands, to make troubleshooting easier.

4. **Use CI-specific Config File**: Consider generating a CI-specific config file in the workflow instead of setting many environment variables.

5. **Document Required Variables**: Add a section to the README or a dedicated CI documentation file that lists all required environment variables for CI.

## Verification

The fix can be verified by:

1. Checking that the CI build passes with our changes.
2. Confirming that the migration step successfully applies migrations to the PostgreSQL container.
3. Verifying that all integration tests execute rather than being skipped.

This fix balances the immediate need to get the CI working with minimal changes to the codebase.
