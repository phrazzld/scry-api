# T029: Build Tag Compatibility Resolution

## Work Completed

### 1. Updated Pre-commit Configuration
Updated `.pre-commit-config.yaml` to include the same build tags used in CI:
```yaml
- id: golangci-lint
  args: [--verbose, --build-tags=test_without_external_deps]  # Match CI configuration
```

### 2. Standardized Build Tags Across Test Helper Files
Added `test_without_external_deps` build tag to the following files to ensure compatibility with CI:

#### API Test Helpers
- `/internal/testutils/api/card_helpers.go`
- `/internal/testutils/api/request_helpers.go`
- `/internal/testutils/api/server_setup.go`
- `/internal/testutils/api/setup.go`
- `/internal/testutils/api/test_data_helpers.go`

#### Authentication Helpers
- `/internal/testutils/auth_helpers.go`
- `/internal/service/auth/jwt_service_mock.go`
- `/internal/service/auth/test_helpers.go`

#### Integration Tests
- `/cmd/server/auth_integration_test.go`
- `/cmd/server/card_api_integration_test.go`
- `/cmd/server/memo_api_integration_test.go`
- `/cmd/server/get_card_api_integration_test.go`
- `/cmd/server/card_review_api_test.go`
- `/cmd/server/refresh_token_integration_test.go`
- `/cmd/server/postpone_card_api_test.go`
- `/cmd/server/auth_middleware_test.go`
- `/cmd/server/main_integration_test.go`
- `/cmd/server/main_migration_test.go`
- `/cmd/server/main_task_test.go`
- `/cmd/server/compatibility.go`

#### HTTP and Database Helpers
- `/internal/testutils/helpers.go`
- `/internal/testutils/http_helpers.go`
- `/internal/testutils/api_helpers.go`
- `/internal/testutils/db.go`

### 3. Fixed Code Compatibility
- Updated references to `testutils.CreateTestJWTService()` to use `auth.NewTestJWTService()` in integration tests
- Addressed build errors due to function redeclarations by adjusting build tags

### 4. Disabled Incompatible Files
- Reverted changes to `testutils_for_tests.go` to keep it internal-only
- Disabled `integration_helpers.go` that contained duplicate function declarations
- Disabled problematic tests that were already being skipped

## Verification
- All packages are now building successfully with the `test_without_external_deps` build tag
- The pre-commit hooks now run with the same build tag as CI
- Integration tests are skipped when no database connection is available (expected behavior)

## Remaining Issues
- There are some linting errors that were present before our changes, which could be addressed in a separate task
- The SQL redaction issues in error logs still need to be addressed as noted in the T029-progress.md file

## Conclusion
The build tag compatibility issues have been resolved. All files now have appropriate build tags to ensure they are included in both local development with the `integration` tag and in CI with the `test_without_external_deps` tag. The pre-commit configuration has been updated to match CI's build tag settings, which should prevent future incompatibilities from being introduced.
