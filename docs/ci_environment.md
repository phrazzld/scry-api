# CI Environment Configuration

This document provides comprehensive documentation of the CI environment configuration for the Scry API project, outlining environment variables, standardization approaches, common issues, and troubleshooting steps.

## Environment Detection

The system uses the following environment variables to detect CI environments:

| Variable | Purpose | Provider |
|----------|---------|----------|
| `CI` | Generic CI environment flag | Most CI providers |
| `GITHUB_ACTIONS` | GitHub Actions specific flag | GitHub Actions |
| `GITHUB_WORKSPACE` | Root directory of GitHub repository | GitHub Actions |
| `CI_PROJECT_DIR` | Root directory of GitLab repository | GitLab CI |
| `GITLAB_CI` | GitLab CI specific flag | GitLab CI |
| `JENKINS_URL` | Jenkins specific flag | Jenkins |
| `TRAVIS` | Travis CI specific flag | Travis CI |
| `CIRCLECI` | Circle CI specific flag | Circle CI |

Detection priority (highest to lowest):
1. Explicit override via `SCRY_PROJECT_ROOT`
2. GitHub Actions environment (`GITHUB_WORKSPACE`)
3. GitLab CI environment (`CI_PROJECT_DIR`)
4. Directory traversal looking for project markers (go.mod)

## Database Configuration

### Database Environment Variables

| Variable | Purpose | Priority | Format |
|----------|---------|----------|--------|
| `DATABASE_URL` | Primary database connection string | 1 (Highest) | `postgres://[username]:[password]@[host]:[port]/[database]?[options]` |
| `SCRY_TEST_DB_URL` | Test-specific database connection | 2 | Same as above |
| `SCRY_DATABASE_URL` | Fallback database connection | 3 | Same as above |

### CI-Specific Database Standards

In CI environments, the system enforces these standards:
- Username and password must be `postgres`
- Standard URL format: `postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable`
- Auto-correction of non-compliant credentials
- Synchronization of all database URL environment variables

See [DATABASE-CI-CONFIGURATION.md](../DATABASE-CI-CONFIGURATION.md) for detailed information on database configuration in CI.

## Project Root Detection

The system determines the project root using the following methods:

1. Environment variables (in order of priority):
   - `SCRY_PROJECT_ROOT` - Explicit override
   - `GITHUB_WORKSPACE` - GitHub Actions repository root
   - `CI_PROJECT_DIR` - GitLab CI repository root

2. Fallback auto-detection:
   - Traverses directories upward from current directory
   - Looks for project markers (go.mod file, .git directory)
   - Verifies the directory has a valid go.mod file

## Migration Configuration

Migration execution in CI uses standardized inputs:

1. Database URL from `GetTestDatabaseURL()`
2. Migration directory from `FindMigrationsDir()`
3. Migration table name from `testdb.MigrationTableName`

Migration verification includes explicit checks for:
- Migration table existence
- Applied migrations status
- Detailed error reporting

## Common CI Issues and Troubleshooting

### Database Connectivity Issues

| Issue | Symptom | Resolution |
|-------|---------|------------|
| Incorrect username | "role 'root' does not exist" | Auto-corrected to `postgres` in CI |
| Missing privileges | Permission denied errors | CI database user has superuser privileges |
| Connection timeout | "connection refused" | Check PostgreSQL service is running in CI |
| Network errors | "could not connect" | CI uses localhost connection by default |

### Project Root Detection Issues

| Issue | Symptom | Resolution |
|-------|---------|------------|
| Root not found | "unable to find project root" | Check CI environment variables (`GITHUB_WORKSPACE`) |
| Invalid path | "directory does not exist" | Verify CI checkout process |
| Wrong workspace | "go.mod not found" | Ensure correct repository checkout configuration |

### Migration Execution Issues

| Issue | Symptom | Resolution |
|-------|---------|------------|
| Migration files not found | "no migration files found" | Check migration directory resolved via `FindMigrationsDir()` |
| Permissions issues | "permission denied" | Ensure CI user has database permissions |
| Schema issues | "relation does not exist" | Verify migration execution order |

### Build and Test Issues

| Issue | Symptom | Resolution |
|-------|---------|------------|
| Build tag conflicts | Build errors | Ensure appropriate build tags for CI |
| Test failures | Test timeout or file not found | Check test skip logic in CI environment |
| Resource limits | Out of memory errors | Adjust CI resource limits |

## Simulation and Testing

### Local CI Environment Simulation

Test your changes in a CI-like environment:

```bash
export CI=true
export GITHUB_ACTIONS=true
export GITHUB_WORKSPACE=$(pwd)
./test-ci-database.sh
```

### Pre-flight Verification

Before submitting changes that might affect CI:

1. Run database configuration tests:
   ```bash
   ./test-ci-database.sh
   ```

2. Verify project root detection:
   ```bash
   go test -v ./internal/testdb -run TestFindProjectRootInCI
   ```

3. Test migration execution:
   ```bash
   go test -v ./cmd/server -run TestMigrations
   ```

## CI Scripts and Tools

### Core CI Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| `scripts/run-integration-tests.sh` | Run integration tests in CI | Automatically executed in CI pipeline |
| `scripts/wait-for-db.sh` | Wait for database to be ready | Used in CI before running tests |
| `scripts/reset-test-db.sh` | Reset test database | Prepare clean database for tests |
| `test-ci-database.sh` | Test database config in CI | Verify database configuration works in CI |

### CI-Specific Commands

Common CI-specific commands:

```bash
# Run migrations in CI
go run ./cmd/server/main.go -migrate=up

# Run tests with CI-compatible tags
go test -v -tags=test_without_external_deps ./...

# Verify database connectivity
go test -v ./internal/testdb -run TestDatabaseConnectivity
```

## Environment Variables Reference

For a complete reference of all environment variables used in the project, including application configuration variables, see [ENVIRONMENT_VARIABLES.md](./ENVIRONMENT_VARIABLES.md).

Below are the CI-specific environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CI` | No | - | Flag indicating CI environment |
| `GITHUB_ACTIONS` | No | - | Flag for GitHub Actions environment |
| `GITHUB_WORKSPACE` | Yes (in GitHub Actions) | - | Root directory of checked out repository |
| `CI_PROJECT_DIR` | Yes (in GitLab CI) | - | Root directory of the GitLab repository |
| `GITLAB_CI` | No | - | Flag for GitLab CI environment |
| `JENKINS_URL` | No | - | Flag for Jenkins CI environment |
| `TRAVIS` | No | - | Flag for Travis CI environment |
| `CIRCLECI` | No | - | Flag for Circle CI environment |
| `SCRY_PROJECT_ROOT` | No | - | Explicit override for project root |

Database-related environment variables with special behavior in CI:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | - | Primary database connection string (will be standardized in CI) |
| `SCRY_DATABASE_URL` | No | - | Standard database connection string |
| `SCRY_TEST_DB_URL` | No | - | Test-specific database connection string |

Note: In CI environments, the database URL is automatically standardized to use `postgres:postgres` credentials, regardless of what was provided in the environment variables.

## Best Practices

1. **Consistent Environment Variable Usage**:
   - Always check for existence before using
   - Provide meaningful defaults where appropriate
   - Follow established naming conventions

2. **Error Handling**:
   - Provide detailed error messages that identify the specific failure
   - Include context information in logs (masked credentials)
   - Handle common failure cases with graceful fallbacks

3. **CI-Friendly Design**:
   - Design components to be aware of CI environment
   - Consider environment differences (filesystem, permissions)
   - Incorporate auto-correction mechanisms for common CI issues
   - Add comprehensive logging for CI debugging

4. **Testing CI Changes**:
   - Test changes locally with simulated CI environment
   - Include tests specific to CI environment scenarios
   - Document CI-specific considerations for your changes

## Future Improvements

1. Add automated pre-flight checks in CI pipeline
2. Standardize all environment variable naming and usage
3. Create dashboard for CI environment health monitoring
4. Develop interactive troubleshooting guide
