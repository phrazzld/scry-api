# Scry API

[![CI Checks](https://github.com/phrazzld/scry-api/actions/workflows/ci.yml/badge.svg)](https://github.com/phrazzld/scry-api/actions/workflows/ci.yml)
[![Security Checks](https://github.com/phrazzld/scry-api/actions/workflows/security.yml/badge.svg)](https://github.com/phrazzld/scry-api/actions/workflows/security.yml)

Scry API is a Go backend service that manages spaced repetition flashcards. It generates flashcards from user-provided memos using LLM integration (Gemini), and employs a modified SM-2 spaced repetition algorithm to schedule reviews based on user performance.

## Getting Started / Setup

### Prerequisites
- Go 1.22+
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

#### Pre-commit Hooks

This project uses pre-commit hooks to ensure code quality. To set up pre-commit hooks:

1. Install pre-commit: `brew install pre-commit` or `pip install pre-commit`
2. In the repository root, run: `pre-commit install`
3. Pre-commit hooks will now run automatically on each commit

To manually run all hooks on all files:
```bash
pre-commit run --all-files
```

### Building the Project
```bash
go build ./cmd/server
```

## Running Tests
Run the full test suite:
```bash
go test ./...
```

To run tests for a specific package:
```bash
go test ./internal/domain
```

## Usage / Running the Application
1. Ensure your configuration is set up (either via `.env` file or `config.yaml` as described above)

2. Start the API server:
   ```bash
   go run ./cmd/server/main.go
   ```

3. The server will be available at `http://localhost:8080` (or the port specified in your configuration)

The server will automatically:
- Load configuration from environment variables and/or config file
- Validate all required settings are present
- Use default values for non-critical settings when not specified
- Log the configured port and other key settings at startup

## Key Scripts / Commands
- Format code: `go fmt ./...`
- Lint code: `golangci-lint run`
- Run tests with coverage: `go test -cover ./...`

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
