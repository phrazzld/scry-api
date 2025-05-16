# Scry API

[![CI Checks](https://github.com/phrazzld/scry-api/actions/workflows/ci.yml/badge.svg)](https://github.com/phrazzld/scry-api/actions/workflows/ci.yml)
[![Security Checks](https://github.com/phrazzld/scry-api/actions/workflows/security.yml/badge.svg)](https://github.com/phrazzld/scry-api/actions/workflows/security.yml)

Scry API is a Go backend service that manages spaced repetition flashcards. It generates flashcards from user-provided memos using LLM integration (Gemini), and employs a modified SM-2 spaced repetition algorithm to schedule reviews based on user performance.

## Post-commit Hook Test

This line is added to test if the post-commit hook runs correctly.

## Getting Started / Setup

### Prerequisites
- Go 1.23+
- PostgreSQL with `pgvector` extension (for production, or via Docker for development)
- Gemini API key for LLM integration

### Environment Setup
1. Clone the repository:
   ```bash
   git clone https://github.com/phrazzld/scry-api.git
   cd scry-api
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Configure the application using one of these methods:

#### Method 1: Environment Variables (Recommended for Production)
Create a `.env` file in the project root with the following variables:
   ```
   # Server configuration
   SCRY_SERVER_PORT=8080
   SCRY_SERVER_LOG_LEVEL=info

   # Database configuration
   SCRY_DATABASE_URL=postgres://username:password@localhost:5432/scry

   # Authentication configuration (minimum 32 characters)
   SCRY_AUTH_JWT_SECRET=your-secure-jwt-secret-min-32-characters

   # LLM integration
   SCRY_LLM_GEMINI_API_KEY=your-gemini-api-key
   ```

   See [.env.example](.env.example) for a template with detailed comments.

#### Method 2: Configuration File (Alternative for Development)
Create a `config.yaml` file in the project root:
   ```yaml
   # Server settings
   server:
     port: 8080
     log_level: info

   # Database settings
   database:
     url: postgres://username:password@localhost:5432/scry

   # Authentication settings
   auth:
     jwt_secret: your-secure-jwt-secret-min-32-characters

   # LLM settings
   llm:
     gemini_api_key: your-gemini-api-key
   ```

   See [config.yaml.example](config.yaml.example) for a template with detailed comments.

> **Note:** Environment variables take precedence over values in config.yaml. Environment variables must have the `SCRY_` prefix and use underscores to represent nesting (e.g., `SCRY_SERVER_PORT` for `server.port`).

> **Security note:** Both `.env` and any custom config files containing secrets should never be committed to version control. They are already added to `.gitignore`.

### Development Setup

#### Build Tags for Testing

This project uses Go build tags to enable different implementation paths for testing with and without external dependencies. This is particularly important for the Gemini API integration.

##### Available Build Tags

- `test_without_external_deps`: Enables mock implementations instead of real external API clients

##### Using Build Tags

For running tests without external dependencies (e.g., in CI environments):

```bash
# Run all tests with mock implementations
go test -v -tags=test_without_external_deps ./...

# Run specific tests with mock implementations
go test -v -tags=test_without_external_deps ./internal/platform/gemini
```

For running tests with real implementations (requires API keys and external access):

```bash
# Run all tests with real implementations
go test -v ./...
```

For building the application with mock implementations:

```bash
# Build with mock implementations
go build -tags=test_without_external_deps ./...
```

For linting with mock implementations:

```bash
# Lint with mock implementations
golangci-lint run --build-tags=test_without_external_deps
```

#### Pre-commit Hooks

This project uses pre-commit hooks to ensure code quality and consistency across the codebase. Pre-commit hooks run automatically when you commit changes, catching issues early in the development cycle.

##### Installation

1. Install pre-commit:
   ```bash
   # macOS
   brew install pre-commit

   # Python/pip (any platform)
   pip install pre-commit
   ```

2. Set up the hooks in your local repository:
   ```bash
   pre-commit install
   ```

3. Pre-commit hooks will now run automatically on each commit

##### Available Hooks

The project uses the following pre-commit hooks:

**Code Quality & Build Checks**
- `golangci-lint`: Runs comprehensive Go linting with the same configuration as CI
- `go-build-check`: Verifies that the application builds without errors

**Formatting Hooks**
- `trailing-whitespace`: Removes trailing whitespace at the end of lines
- `end-of-file-fixer`: Ensures files end with a newline
- `golines`: Formats Go code, fixes imports, and wraps long lines (max 120 chars)

**Validation Hooks**
- `check-yaml`: Validates YAML syntax
- `check-json`: Validates JSON syntax
- `check-merge-conflict`: Prevents committing files with merge conflict markers
- `check-added-large-files`: Prevents committing large files (>500KB)

**Custom Hooks**
- `go-mod-tidy`: Ensures go.mod is always tidy
- `warn-long-files`: Warns (but doesn't block commits) when files exceed 500 lines, encouraging modular code design
- `fail-extremely-long-files`: Fails the commit if any file exceeds 1000 lines
- `check-for-panics`: Prevents committing code with direct panic() calls without exemption
- `check-sql-ordering`: Ensures all ORDER BY clauses include a secondary sort key for deterministic results

##### Usage

To manually run all hooks on all files (useful before pushing changes):
```bash
pre-commit run --all-files
```

To run a specific hook:
```bash
pre-commit run <hook-id> --all-files
```

For example:
```bash
pre-commit run golangci-lint --all-files
```

##### Configuration

The pre-commit configuration is in `.pre-commit-config.yaml` at the root of the repository. See this file for detailed hook documentation and configuration options.

### Building the Project
```bash
make build
```

For more build options, see the [Development Guide](docs/DEVELOPMENT_GUIDE.md#build-and-deployment).

## Running Tests
Run the full test suite:
```bash
make test
```

For more testing options (coverage, integration tests, etc.), see the [Development Guide](docs/DEVELOPMENT_GUIDE.md#testing).

## Usage / Running the Application
1. Ensure your configuration is set up (either via `.env` file or `config.yaml` as described above)

2. Start the API server:
   ```bash
   make run-server
   ```

3. The server will be available at `http://localhost:8080` (or the port specified in your configuration)

The server will automatically:
- Load configuration from environment variables and/or config file
- Validate all required settings are present
- Use default values for non-critical settings when not specified
- Log the configured port and other key settings at startup

### Database Migrations

The application uses [goose](https://github.com/pressly/goose) for database migrations.

To run migrations:

```bash
# Run all pending migrations
make migrate-up

# Rollback the last migration
make migrate-down

# Show migration status
make migrate-status

# Show current version
make migrate-version

# Create a new migration
make migrate-create NAME=create_users_table
```

For more database operations, see the [Development Guide](docs/DEVELOPMENT_GUIDE.md#database-operations).

Migration files are stored in `internal/platform/postgres/migrations/`. See the [migrations README](internal/platform/postgres/migrations/README.md) for more details.

## Key Scripts / Commands
- Format code: `make fmt`
- Lint code: `make lint`
- Run tests with coverage: `make test-coverage`
- View all available commands: `make help`

For a comprehensive list of development commands, see the [Development Guide](docs/DEVELOPMENT_GUIDE.md).

## Architecture Overview
The project follows a clean architecture approach with clear separation of concerns:

- `/cmd/server`: Application entry point and server setup
- `/internal/domain`: Core business entities and logic
- `/internal/service`: Application services and use cases
- `/internal/store`: Data storage interfaces
- `/internal/api`: HTTP handlers and routing
- `/internal/config`: Configuration management with support for environment variables and YAML files
- `/internal/generation`: LLM integration for card generation
- `/internal/task`: Background processing and job management
- `/internal/platform/postgres`: Database implementation
- `/internal/platform/logger`: Structured logging using Go's standard library `log/slog` with JSON output format

The configuration system (`/internal/config`) uses Viper for flexible configuration loading and validation:
- Strongly-typed configuration via Go structs
- Support for both environment variables and YAML files
- Validation using go-playground/validator
- Sensible defaults with clear precedence rules

For more details on the architectural principles, see the [Architecture Guidelines](docs/philosophy/ARCHITECTURE_GUIDELINES.md).

## How to Contribute
Contributions are welcome! Before contributing, please read the project's core principles and guidelines found in the `docs/philosophy/` directory, particularly:

- [Core Principles](docs/philosophy/CORE_PRINCIPLES.md)
- [Architecture Guidelines](docs/philosophy/ARCHITECTURE_GUIDELINES.md)
- [Coding Standards](docs/philosophy/CODING_STANDARDS.md)
- [Testing Strategy](docs/philosophy/TESTING_STRATEGY.md)

## License
This project is licensed under the MIT License - see the LICENSE file for details.
