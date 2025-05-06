# CI Failure Audit for PR #26 (After PostgreSQL Integration)

## Summary of Failure

The CI build for PR #26 "feat: implement card management API endpoints" is still failing, but with a new error. The test job fails at the database migration step with a configuration validation error.

## Detailed Analysis

### Error Message

```
2025/05/06 13:11:14 ERROR Failed to load configuration for migration error="failed to load configuration: configuration validation failed: Key: 'Config.Database.URL' Error:Field validation for 'URL' failed on the 'required' tag
Key: 'Config.Auth.JWTSecret' Error:Field validation for 'JWTSecret' failed on the 'required' tag
Key: 'Config.LLM.GeminiAPIKey' Error:Field validation for 'GeminiAPIKey' failed on the 'required' tag
Key: 'Config.LLM.PromptTemplatePath' Error:Field validation for 'PromptTemplatePath' failed on the 'required' tag"
```

### Root Cause

After examining the code, the problem is clear. The application's configuration system (in `internal/config/load.go`) requires several environment variables that we haven't included in our CI workflow:

1. While we correctly set `DATABASE_URL` in the environment, the code actually looks for `SCRY_DATABASE_URL` (with the "SCRY_" prefix) as set in the config binding on line 103 of `load.go`
2. All configuration variables are required due to the validation tags in `config.go`
3. The migration command attempts to load the full config, including LLM and auth settings, when it only needs the database settings

### Container Status

The PostgreSQL container itself appears to be running correctly:
- Container was started successfully
- Database was initialized properly
- PostgreSQL was listening on port 5432 as expected

## Recommended Fix

### Option 1 (Recommended): Set All Required Environment Variables

Add all the required environment variables with the correct SCRY_ prefix to the CI workflow:

```yaml
env:
  SCRY_DATABASE_URL: postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable
  SCRY_AUTH_JWT_SECRET: test-jwt-secret
  SCRY_AUTH_BCRYPT_COST: "10"
  SCRY_AUTH_TOKEN_LIFETIME_MINUTES: "60"
  SCRY_AUTH_REFRESH_TOKEN_LIFETIME_MINUTES: "10080"
  SCRY_LLM_GEMINI_API_KEY: test-gemini-key
  SCRY_LLM_MODEL_NAME: gemini-2.0-flash
  SCRY_LLM_PROMPT_TEMPLATE_PATH: prompts/flashcard_template.txt
  SCRY_LLM_MAX_RETRIES: "3"
  SCRY_LLM_RETRY_DELAY_SECONDS: "2"
  SCRY_SERVER_PORT: "8080"
  SCRY_SERVER_LOG_LEVEL: info
  SCRY_TASK_WORKER_COUNT: "2"
  SCRY_TASK_QUEUE_SIZE: "100"
  SCRY_TASK_STUCK_TASK_AGE_MINUTES: "30"
```

### Option 2 (Longer-term): Refactor Migration Command

For a more robust solution in the future, we should consider modifying the application code to make database migrations more flexible:

1. Create a separate lightweight configuration loader just for migrations that only validates database-related config
2. Update the migration command to use this lightweight config loader
3. This would make the migration process more robust and less coupled to other application components

## Implementation Steps

1. Modify `.github/workflows/ci.yml` to include all required environment variables with the SCRY_ prefix
2. Pay special attention to the SCRY_DATABASE_URL variable which is needed instead of DATABASE_URL
3. Keep the SCRY_TEST_DB_URL for tests that might use it directly
4. Ensure the SCRY_LLM_PROMPT_TEMPLATE_PATH points to a valid file that exists in the repository

## Longer-term Recommendations

1. Consider refactoring the configuration system to allow partial validation for specific subsystems
2. Add better error messages for missing configuration, especially for migration commands
3. Create a test-specific configuration profile with sensible defaults for all required values
4. Consider using a standalone migration tool that doesn't require the full application config
