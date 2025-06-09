# Infrastructure Package

This package contains infrastructure-related tests and configuration for the Scry API project.

## Purpose

The infrastructure package contains **integration tests** that verify external system setup and connectivity, not production application code. These tests validate:

- Local development database setup (Docker-based PostgreSQL)
- Migration script functionality
- Terraform infrastructure provisioning
- Database connectivity and extensions

## Test Coverage

**Note**: This package intentionally shows "zero coverage" in standard coverage reports because it contains only test files that verify external systems. This is expected and correct behavior.

## Test Categories

### Local Development Tests (`local_dev/`)
- **Purpose**: Verify Docker-based PostgreSQL setup for local development
- **Trigger**: Set `DOCKER_TEST=1` environment variable
- **Requirements**: Docker and docker-compose installed

### Script Tests (`scripts/`)
- **Purpose**: Validate migration scripts and database operations
- **Trigger**: Set `TEST_DATABASE_URL` environment variable
- **Requirements**: Running PostgreSQL instance

### Terraform Tests (`terraform/test/`)
- **Purpose**: Validate infrastructure provisioning on cloud platforms
- **Trigger**: Set `TERRATEST_ENABLED=1` environment variable
- **Requirements**: Cloud provider credentials and terratest dependencies

## Running Tests

```bash
# Run all infrastructure tests
make test-infrastructure

# Run Docker-based tests only
make test-infrastructure-docker

# Run Terraform tests only (requires cloud credentials)
make test-infrastructure-terraform
```

## CI Integration

Infrastructure tests are designed to run in CI environments with appropriate credentials and can verify the complete infrastructure provisioning and connectivity pipeline.
