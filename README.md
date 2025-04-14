# Scry API

Scry API is a Go backend service that manages spaced repetition flashcards. It generates flashcards from user-provided memos using LLM integration (Gemini), and employs a modified SM-2 spaced repetition algorithm to schedule reviews based on user performance.

## Getting Started / Setup

### Prerequisites
- Go 1.21+ 
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

3. Configure environment variables (create a `.env` file in the project root):
   ```
   # Server
   PORT=8080
   LOG_LEVEL=info
   
   # Database
   DATABASE_URL=postgres://username:password@localhost:5432/scry
   
   # Authentication
   JWT_SECRET=your-secret-key
   
   # LLM Integration
   GEMINI_API_KEY=your-gemini-api-key
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
Start the API server:
```bash
go run ./cmd/server/main.go
```

The server will be available at `http://localhost:8080` (or the port specified in your environment variables).

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
- `/internal/config`: Configuration management
- `/internal/generation`: LLM integration for card generation
- `/internal/task`: Background processing and job management
- `/internal/platform/postgres`: Database implementation

For more details on the architectural principles, see the [Architecture Guidelines](docs/philosophy/ARCHITECTURE_GUIDELINES.md).

## How to Contribute
Contributions are welcome! Before contributing, please read the project's core principles and guidelines found in the `docs/philosophy/` directory, particularly:

- [Core Principles](docs/philosophy/CORE_PRINCIPLES.md)
- [Architecture Guidelines](docs/philosophy/ARCHITECTURE_GUIDELINES.md)
- [Coding Standards](docs/philosophy/CODING_STANDARDS.md)
- [Testing Strategy](docs/philosophy/TESTING_STRATEGY.md)

## License
This project is licensed under the MIT License - see the LICENSE file for details.