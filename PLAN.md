# Implementation Plan: Database Migration Framework Setup

## Goal
Set up a robust, maintainable database migration framework for the Scry API that integrates with the existing codebase, supports PostgreSQL migrations, and adheres to the project's architectural principles and coding standards.

## 1. Overview of Approach

After evaluating multiple potential migration tools (golang-migrate, goose, and liquibase), **this plan recommends using the `pressly/goose` library** integrated directly into the Go application. This approach provides the best balance of simplicity, integration with existing code, and adherence to project standards.

Key features of this approach:
- Migrations will be written as plain SQL files for maximum clarity and simplicity
- Migration commands will be executed through a dedicated subcommand in the main application binary
- The framework will leverage existing configuration, logging, and error handling systems
- Migration operations will be properly logged through the application's structured logging framework

## 2. Detailed Implementation Steps

### 2.1. Add Dependencies

1. Add `github.com/pressly/goose/v3` to `go.mod`:
   ```bash
   go get github.com/pressly/goose/v3
   ```

2. Ensure the appropriate PostgreSQL driver is present:
   ```bash
   go get github.com/jackc/pgx/v5/stdlib
   go mod tidy
   ```

### 2.2. Create Migration Directory Structure

1. Create a dedicated directory for migrations:
   ```bash
   mkdir -p internal/platform/postgres/migrations
   ```

2. Add a `.keep` file to ensure the directory is included in version control even when empty.

### 2.3. Implement Migration Command

Create a mechanism to run migrations through the main application by adding a `migrate` subcommand to `cmd/server/main.go`.

1. Modify `cmd/server/main.go` to detect and handle the `migrate` command:
   - Use standard `flag` package to detect migration commands
   - If a migration command is detected, execute it and exit
   - Otherwise, proceed with normal server startup

2. Implement a `runMigrations` function to handle the migration execution:
   - Takes the migration command (up, down, status, etc.) and arguments
   - Uses the existing configuration system to get database connection details
   - Connects to the database and executes the specified migration command
   - Returns appropriate errors or success messages

### 2.4. Implement Logging Integration

1. Create a custom logger adapter to integrate `goose` with the application's structured logging:
   ```go
   // slogGooseLogger adapts slog for goose's simple logger interface
   type slogGooseLogger struct{}

   func (l *slogGooseLogger) Printf(format string, v ...interface{}) {
       slog.Info(fmt.Sprintf(format, v...))
   }

   func (l *slogGooseLogger) Fatalf(format string, v ...interface{}) {
       slog.Error(fmt.Sprintf(format, v...))
       os.Exit(1)
   }
   ```

2. Configure `goose` to use this logger:
   ```go
   goose.SetLogger(&slogGooseLogger{})
   ```

### 2.5. Implement Migration Commands

Support the following migration commands:
- `up`: Apply all pending migrations
- `down`: Roll back the last applied migration
- `status`: Show current migration status
- `create`: Create a new migration file
- `version`: Show the current migration version

The implementation should handle all commands appropriately, validate inputs, and provide clear error messages.

### 2.6. Add Tests

Implement tests to verify the migration framework functions correctly:
- Unit tests for the migration command handling logic
- Basic integration tests to verify migration file creation and execution

### 2.7. Update Documentation

1. Update `README.md` with instructions on how to:
   - Create new migrations
   - Apply migrations
   - Check migration status
   - Roll back migrations

2. Document the migration file format and naming conventions

## 3. Example Implementation

### 3.1. Migration Command in `cmd/server/main.go`

```go
package main

import (
    "database/sql"
    "flag"
    "fmt"
    "log/slog"
    "os"

    "github.com/jackc/pgx/v5/stdlib"
    "github.com/phrazzld/scry-api/internal/config"
    "github.com/phrazzld/scry-api/internal/platform/logger"
    "github.com/pressly/goose/v3"
)

const migrationsDir = "internal/platform/postgres/migrations"

func main() {
    // Migration command-line flags
    migrateCmd := flag.String("migrate", "", "Run database migrations (up|down|create|status|version)")
    migrationName := flag.String("name", "", "Name for new migration file (used with -migrate=create)")
    flag.Parse()

    // Initialize logger early for startup messages
    slog.Info("Scry API Server starting...")

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        slog.Error("Failed to load configuration", "error", err)
        os.Exit(1)
    }

    // Setup structured logging
    _, err = logger.Setup(cfg.Server)
    if err != nil {
        slog.Error("Failed to set up logger", "error", err)
        os.Exit(1)
    }

    // Handle migration command if specified
    if *migrateCmd != "" {
        err := runMigrations(cfg, *migrateCmd, *migrationName)
        if err != nil {
            slog.Error("Migration command failed", "command", *migrateCmd, "error", err)
            os.Exit(1)
        }
        slog.Info("Migration command finished successfully", "command", *migrateCmd)
        os.Exit(0)
    }

    // Normal server startup logic
    slog.Info("Scry API Server initialized successfully", "port", cfg.Server.Port)
    // ... rest of the server logic ...
}

// runMigrations handles the execution of database migrations
func runMigrations(cfg *config.Config, command string, args ...string) error {
    slog.Info("Running migration command", "command", command, "dir", migrationsDir)

    // Configure goose logger
    goose.SetLogger(&slogGooseLogger{})

    // Open database connection
    db, err := sql.Open("pgx", cfg.Database.URL)
    if err != nil {
        return fmt.Errorf("failed to open database connection: %w", err)
    }
    defer db.Close()

    if err := db.Ping(); err != nil {
        return fmt.Errorf("failed to ping database: %w", err)
    }

    // Execute the goose command
    switch command {
    case "up":
        return goose.Up(db, migrationsDir)
    case "down":
        return goose.Down(db, migrationsDir)
    case "status":
        return goose.Status(db, migrationsDir)
    case "create":
        if len(args) == 0 || args[0] == "" {
            return fmt.Errorf("migration name must be provided for 'create' command")
        }
        return goose.Create(db, migrationsDir, args[0], "sql")
    case "version":
        return goose.Version(db, migrationsDir)
    default:
        return fmt.Errorf("unknown migration command: %s", command)
    }
}

// slogGooseLogger adapts slog for goose's logger interface
type slogGooseLogger struct{}

func (l *slogGooseLogger) Printf(format string, v ...interface{}) {
    slog.Info(fmt.Sprintf(format, v...))
}

func (l *slogGooseLogger) Fatalf(format string, v ...interface{}) {
    slog.Error(fmt.Sprintf(format, v...))
    os.Exit(1)
}
```

### 3.2. Example Initial Migration

Creating an example initial migration to verify the setup:

```bash
go run ./cmd/server/main.go -migrate=create -name=init_schema
```

This will create `internal/platform/postgres/migrations/yyyymmddhhmmss_init_schema.sql` with placeholders for `-- +goose Up` and `-- +goose Down` sections.

## 4. Alignment with Project Standards

### 4.1. Simplicity and Clarity (CORE_PRINCIPLES.md)
This approach embraces simplicity by:
- Using plain SQL files for migrations (clear, easy to understand format)
- Minimizing setup complexity with a straightforward integration pattern
- Using a battle-tested library with a clean API
- Keeping the migration command interface simple and intuitive

### 4.2. Separation of Concerns (ARCHITECTURE_GUIDELINES.md)
This approach maintains proper separation by:
- Keeping migration-related code in a dedicated function
- Integrating with existing configuration and logging systems
- Focusing each migration file on a single schema change
- Allowing separation between the concerns of running the application and managing migrations

### 4.3. Testability (TESTING_STRATEGY.md)
The approach supports testability by:
- Making migration commands callable as pure functions for unit testing
- Allowing integration tests with test containers to verify actual migration behavior
- Supporting isolation of migration functionality from the rest of the application

### 4.4. Coding Conventions (CODING_STANDARDS.md)
The implementation follows project coding standards by:
- Using strong typing throughout
- Employing standard Go practices for error handling
- Using the established logging framework
- Adding descriptive comments to explain the migration process

### 4.5. Documentation (DOCUMENTATION_APPROACH.md)
The plan ensures proper documentation by:
- Adding clear usage instructions to README.md
- Using descriptive migration file names
- Adding comments to explain migration file formats
- Documenting commands and options

## 5. Potential Challenges and Mitigation

1. **Database Connection Errors:**
   - Implement proper error handling and clear error messages
   - Provide guidance in docs for common connection issues

2. **Migration Conflicts:**
   - Use a consistent naming convention with timestamps
   - Document best practices for creating migrations
   - Ensure CI/CD pipeline catches conflicts before deployment

3. **Migration Performance:**
   - For large migrations that might lock tables, consider breaking them into smaller operations
   - Document migration performance considerations

## 6. Implementation Summary

This plan provides a clean, maintainable, and integrated approach to database migrations for the Scry API. By using the `pressly/goose` library with SQL migration files, we achieve a good balance of simplicity, power, and integration with the existing codebase.

The implementation avoids external dependencies outside the Go ecosystem and leverages the application's existing configuration and logging systems. Migration commands will be accessible through the main application binary, providing a cohesive experience for developers and in CI/CD environments.
