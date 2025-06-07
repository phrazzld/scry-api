# Build Tag Fixes for CI

The current issue is that our integration tests are failing in CI because there are multiple function definitions with the same name across different build tag files in the testutils package. This is causing compilation errors because in CI:

1. The build tag `!integration_test_internal` in db_forwarding.go is active
2. The build tag `test_without_external_deps` in db.go is also active
3. The build tag `!integration` in env.go is active

This means we have multiple definitions of:
- GetTestDB
- SetupTestDatabaseSchema
- WithTx
- GetTestDBWithT
- IsIntegrationTestEnvironment
- MustGetTestDatabaseURL
- GenerateAuthHeader

## Solution Strategy

Instead of modifying multiple existing files with complex build tag logic, we should:

1. Rename db_forwarding.go to a different name
2. Create a new file with a more specific build tag combination
3. Remove the functions from db_forwarding.go that conflict with other files
4. Ensure card_api_helpers.go is only built when needed

This approach minimizes changes to existing files while fixing the CI issues.

## Implementation Plan

1. The current `db_forwarding.go` file will be renamed and its build tag modified
2. Update build tag for `card_api_helpers.go` to avoid conflicts
3. Add documentation about the build tag strategy

## Testing

We should verify the changes by:
1. Running unit tests with `integration` tag
2. Compiling the postgres package with the `integration` tag
3. Checking for compilation errors or warnings
