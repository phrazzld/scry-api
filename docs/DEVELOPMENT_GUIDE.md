# Development Guide

This guide provides comprehensive documentation for common development tasks in the Scry API project. All commands use the standardized `Makefile` targets for consistency and ease of use.

## Table of Contents
- [Getting Started](#getting-started)
- [Common Development Commands](#common-development-commands)
- [Server Operations](#server-operations)
- [Database Operations](#database-operations)
- [Testing](#testing)
- [Code Quality](#code-quality)
- [Build and Deployment](#build-and-deployment)
- [Advanced Usage](#advanced-usage)
- [Troubleshooting](#troubleshooting)

## Getting Started

This project uses a `Makefile` to standardize common development commands. To see all available commands:

```bash
make help
```

## Common Development Commands

### Quick Reference

| Task | Command | Description |
|------|---------|-------------|
| Run server | `make run-server` | Start the API server |
| Run tests | `make test` | Run all tests |
| Apply migrations | `make migrate-up` | Apply database migrations |
| Lint code | `make lint` | Run golangci-lint |
| Format code | `make fmt` | Format with gofmt and goimports |
| Build binary | `make build` | Build the application |

## Server Operations

### Running the Server

```bash
# Start the server with default configuration
make run-server

# Start with custom flags
make run-server FLAGS="--debug --port=8081"

# Start with specific log level
make run-server FLAGS="--log-level=debug"
```

The underlying command is `go run ./cmd/server`, which uses package-based execution for proper multi-file compilation.

## Database Operations

### CGo Requirements

This project uses the PostgreSQL database driver, which requires CGo. Before running any database operations or tests, ensure:

1. CGo is enabled: `export CGO_ENABLED=1`
2. Required C libraries are installed: `gcc` and `libpq-dev`

For detailed requirements and troubleshooting, see [CGo Requirements](environment/CGO_REQUIREMENTS.md).

### Migration Management

```bash
# Apply all pending migrations
CGO_ENABLED=1 make migrate-up

# Rollback the last migration
CGO_ENABLED=1 make migrate-down

# Check migration status
CGO_ENABLED=1 make migrate-status

# View current migration version
CGO_ENABLED=1 make migrate-version

# Create a new migration
make migrate-create NAME=add_user_preferences
```

All migration commands use the main server binary with the `-migrate` flag.

## Testing

### Running Tests

```bash
# Run all tests (CGo must be enabled for database tests)
CGO_ENABLED=1 make test

# Run tests with verbose output
CGO_ENABLED=1 make test-verbose

# Run integration tests (requires database and CGo)
CGO_ENABLED=1 make test-integration

# Run tests without external dependencies
make test-no-deps

# Generate coverage report
CGO_ENABLED=1 make test-coverage

# Generate HTML coverage report
CGO_ENABLED=1 make test-coverage-html
```

### Test Tags

The project uses build tags to control test execution:
- `integration`: Tests that require a database
- `test_without_external_deps`: Tests that use mocked external services

### Coverage Thresholds

The project enforces code coverage thresholds in CI to maintain quality:

- **Overall Project**: Minimum 70% coverage required
- **Core Domain Logic**: Minimum 90-95% coverage required
- **Service Layer**: Minimum 85-90% coverage required
- **Data Access Layer**: Minimum 85% coverage required

These thresholds are defined in `coverage-thresholds.json` and enforced in the CI pipeline. Pull requests will fail if they reduce coverage below these thresholds.

## Code Quality

### Linting and Formatting

```bash
# Run linter
make lint

# Run linter with auto-fix
make lint-fix

# Format code
make fmt
```

### Pre-commit Checks

Before committing, run all quality checks:

```bash
make pre-commit
```

This runs formatting, dependency tidying, linting, and tests.

## Build and Deployment

### Building the Application

```bash
# Build the main binary
make build

# Build all packages
make build-all

# Clean build artifacts
make clean
```

The binary is output to `./bin/scry-api`.

### Docker Operations

```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

## Advanced Usage

### Dependency Management

```bash
# Download dependencies
make deps

# Tidy dependencies
make deps-tidy

# Verify dependencies
make deps-verify
```

### CI/CD Commands

These commands are used in continuous integration:

```bash
# Run full CI build
make ci-build

# Run CI tests with coverage
make ci-test
```

### Combined Operations

```bash
# Run lint and tests
make check

# Run all pre-commit checks
make pre-commit
```

## Why Package-Based Execution?

This project uses package-based Go commands (e.g., `go run ./cmd/server`) instead of file-based execution (e.g., `go run cmd/server/main.go`) for several reasons:

1. **Multi-file packages**: When a package contains multiple `.go` files, all must be compiled together
2. **Build consistency**: Package-based execution matches how `go build` works
3. **Future-proofing**: Adding new files to a package doesn't break existing commands
4. **CI/CD compatibility**: Works consistently across different environments

## Troubleshooting

### Common Issues

**Command not found: make**
- Install make on your system:
  - macOS: `brew install make`
  - Ubuntu/Debian: `sudo apt-get install make`
  - Other systems: Check your package manager

**Permission denied**
- Ensure scripts are executable: `chmod +x scripts/*.sh`
- Check file permissions in the project

**Database connection errors**
- Verify PostgreSQL is running
- Check DATABASE_URL environment variable
- Ensure database exists and migrations are applied
- Make sure CGo is enabled: `export CGO_ENABLED=1`
- Verify CGo dependencies are installed (gcc, libpq-dev)
- See [CGo Requirements](environment/CGO_REQUIREMENTS.md) for detailed troubleshooting

**Build failures**
- Run `make deps` to ensure dependencies are downloaded
- Check Go version: `go version` (requires 1.21+)
- Clear build cache: `go clean -cache`

### Getting Help

1. Check the error message carefully
2. Review relevant environment variables
3. Consult the [README.md](../README.md) for setup instructions
4. Check [CI configuration](.github/workflows/ci.yml) for working examples
5. Open an issue with detailed error information

## Contributing

When adding new commands:

1. Add the command to the `Makefile` with a descriptive target name
2. Include a help comment (## Description)
3. Update this guide with usage documentation
4. Test the command locally
5. Update CI workflows if needed

Remember to maintain consistency with existing patterns and follow the project's development philosophy.
