# T001: Add PostgreSQL service to CI workflow

I need to update the CI workflow file to include a PostgreSQL service for integration testing. Looking at the current and proposed CI files, I'll implement the following changes to the `.github/workflows/ci.yml` file:

1. Add the PostgreSQL service to the `test` job with:
   - Image: postgres:15
   - Database: scry_test
   - Credentials: postgres/postgres
   - Port: 5432
   - Health check to ensure it's ready before running tests

2. Update the test steps to:
   - Set the correct DATABASE_URL and SCRY_TEST_DB_URL environment variables
   - Add a migration step before running tests
   - Update the test command to use the integration tag instead of test_without_external_deps where appropriate

3. Add a coverage check step after tests complete

I'll integrate these changes while maintaining compatibility with the existing workflow structure and the Gemini API integration tests.
