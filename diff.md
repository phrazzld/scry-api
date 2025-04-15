# Code Review Instructions

You are a meticulous AI Code Reviewer and guardian of project standards. Your task is to thoroughly review the provided code changes (diff) against the project's established standards and provide constructive, actionable feedback.

## Instructions

1. **Analyze Diff:** Carefully examine the code changes provided in the diff.

2. **Evaluate Against Standards:** For every change, critically assess its adherence to **all** provided standards documents (`CORE_PRINCIPLES.md`, `ARCHITECTURE_GUIDELINES.md`, `CODING_STANDARDS.md`, `TESTING_STRATEGY.md`, `DOCUMENTATION_APPROACH.md`). Look for:
   * Potential bugs or logical errors.
   * Violations of simplicity, modularity, or explicitness (`CORE_PRINCIPLES.md`).
   * Conflicts with architectural patterns or separation of concerns (`ARCHITECTURE_GUIDELINES.md`).
   * Deviations from coding conventions (`CODING_STANDARDS.md`).
   * Poor test design, unnecessary complexity, or excessive mocking (`TESTING_STRATEGY.md`).
   * Inadequate or unclear documentation (`DOCUMENTATION_APPROACH.md`).
   * Opportunities for improvement in clarity, efficiency, or maintainability.

3. **Provide Feedback:** Structure your feedback clearly. For each issue found:
   * Describe the issue precisely.
   * Reference the specific standard(s) it violates (if applicable).
   * Suggest a concrete solution or improvement.
   * Note the file and line number(s).

4. **Summarize:** Conclude with a Markdown table summarizing the key findings:

   | Issue Description | Location (File:Line) | Suggested Solution / Improvement | Risk Assessment (Low/Medium/High) | Standard Violated |
   |---|---|---|---|---|
   | ... | ... | ... | ... | ... |

## Output

Provide the detailed code review feedback, followed by the summary table, formatted as Markdown suitable for saving as `CODE_REVIEW.MD`. Ensure feedback is constructive and directly tied to the provided standards or general best practices.

## Diffdiff --git a/BACKLOG.md b/BACKLOG.md
index 119e42d..211ccf0 100644
--- a/BACKLOG.md
+++ b/BACKLOG.md
@@ -7,12 +7,30 @@
 This backlog outlines the major work items required to build the Minimum Viable Product (MVP) for the Scry Go backend API, hosted entirely on DigitalOcean. Items should generally be tackled in order, aligning with our core principles and engineering guidelines. Each item represents a meaningful chunk of work, intended to be broken down further into detailed plans and tasks.
 
 
-* **Authentication Implementation:**
-    * Implement `store.UserStore` interface and PostgreSQL implementation (`internal/platform/postgres`) for user CRUD, including secure password hashing (`bcrypt`).
-    * Implement JWT generation logic within an `auth.Service`.
-    * Implement User Registration endpoint (`POST /auth/register`) in `internal/api`, utilizing `auth.Service` and `store.UserStore`.
+* **User Store Implementation:**
+    * Define `store.UserStore` interface with methods for CRUD operations.
+    * Implement PostgreSQL implementation (`internal/platform/postgres`) for user CRUD operations.
+    * Implement secure password hashing using `bcrypt`.
+    * Ensure validation of all user data before storage.
+    * Add comprehensive tests for store implementation.
+
+* **JWT Authentication Service (Dependency: User Store):**
+    * Implement JWT generation and validation logic in `auth.Service`.
+    * Implement token refresh mechanisms if needed.
+    * Add necessary configuration for JWT secrets and token lifetimes.
+    * Add comprehensive tests for authentication service.
+
+* **Authentication API Endpoints (Dependencies: User Store, JWT Auth Service):**
+    * Implement User Registration endpoint (`POST /auth/register`) in `internal/api`.
     * Implement User Login endpoint (`POST /auth/login`).
-    * Implement Authentication Middleware (JWT validation) for protecting relevant API routes.
+    * Ensure proper error handling and validation for all endpoints.
+    * Add integration tests for authentication endpoints.
+
+* **Authentication Middleware (Dependency: JWT Auth Service):**
+    * Implement JWT validation middleware for protecting API routes.
+    * Integrate middleware with the router.
+    * Add role-based access control if needed.
+    * Add tests for middleware functionality.
 
 * **Asynchronous Task Runner Setup:**
     * Implement basic in-memory background task queue & worker pool (`internal/task`) using goroutines/channels.
diff --git a/PLAN.md b/PLAN.md
deleted file mode 100644
index a33d343..0000000
--- a/PLAN.md
+++ /dev/null
@@ -1,281 +0,0 @@
-# Implementation Plan: Database Migration Framework Setup
-
-## Goal
-Set up a robust, maintainable database migration framework for the Scry API that integrates with the existing codebase, supports PostgreSQL migrations, and adheres to the project's architectural principles and coding standards.
-
-## 1. Overview of Approach
-
-After evaluating multiple potential migration tools (golang-migrate, goose, and liquibase), **this plan recommends using the `pressly/goose` library** integrated directly into the Go application. This approach provides the best balance of simplicity, integration with existing code, and adherence to project standards.
-
-Key features of this approach:
-- Migrations will be written as plain SQL files for maximum clarity and simplicity
-- Migration commands will be executed through a dedicated subcommand in the main application binary
-- The framework will leverage existing configuration, logging, and error handling systems
-- Migration operations will be properly logged through the application's structured logging framework
-
-## 2. Detailed Implementation Steps
-
-### 2.1. Add Dependencies
-
-1. Add `github.com/pressly/goose/v3` to `go.mod`:
-   ```bash
-   go get github.com/pressly/goose/v3
-   ```
-
-2. Ensure the appropriate PostgreSQL driver is present:
-   ```bash
-   go get github.com/jackc/pgx/v5/stdlib
-   go mod tidy
-   ```
-
-### 2.2. Create Migration Directory Structure
-
-1. Create a dedicated directory for migrations:
-   ```bash
-   mkdir -p internal/platform/postgres/migrations
-   ```
-
-2. Add a `.keep` file to ensure the directory is included in version control even when empty.
-
-### 2.3. Implement Migration Command
-
-Create a mechanism to run migrations through the main application by adding a `migrate` subcommand to `cmd/server/main.go`.
-
-1. Modify `cmd/server/main.go` to detect and handle the `migrate` command:
-   - Use standard `flag` package to detect migration commands
-   - If a migration command is detected, execute it and exit
-   - Otherwise, proceed with normal server startup
-
-2. Implement a `runMigrations` function to handle the migration execution:
-   - Takes the migration command (up, down, status, etc.) and arguments
-   - Uses the existing configuration system to get database connection details
-   - Connects to the database and executes the specified migration command
-   - Returns appropriate errors or success messages
-
-### 2.4. Implement Logging Integration
-
-1. Create a custom logger adapter to integrate `goose` with the application's structured logging:
-   ```go
-   // slogGooseLogger adapts slog for goose's simple logger interface
-   type slogGooseLogger struct{}
-
-   func (l *slogGooseLogger) Printf(format string, v ...interface{}) {
-       slog.Info(fmt.Sprintf(format, v...))
-   }
-
-   func (l *slogGooseLogger) Fatalf(format string, v ...interface{}) {
-       slog.Error(fmt.Sprintf(format, v...))
-       os.Exit(1)
-   }
-   ```
-
-2. Configure `goose` to use this logger:
-   ```go
-   goose.SetLogger(&slogGooseLogger{})
-   ```
-
-### 2.5. Implement Migration Commands
-
-Support the following migration commands:
-- `up`: Apply all pending migrations
-- `down`: Roll back the last applied migration
-- `status`: Show current migration status
-- `create`: Create a new migration file
-- `version`: Show the current migration version
-
-The implementation should handle all commands appropriately, validate inputs, and provide clear error messages.
-
-### 2.6. Add Tests
-
-Implement tests to verify the migration framework functions correctly:
-- Unit tests for the migration command handling logic
-- Basic integration tests to verify migration file creation and execution
-
-### 2.7. Update Documentation
-
-1. Update `README.md` with instructions on how to:
-   - Create new migrations
-   - Apply migrations
-   - Check migration status
-   - Roll back migrations
-
-2. Document the migration file format and naming conventions
-
-## 3. Example Implementation
-
-### 3.1. Migration Command in `cmd/server/main.go`
-
-```go
-package main
-
-import (
-    "database/sql"
-    "flag"
-    "fmt"
-    "log/slog"
-    "os"
-
-    "github.com/jackc/pgx/v5/stdlib"
-    "github.com/phrazzld/scry-api/internal/config"
-    "github.com/phrazzld/scry-api/internal/platform/logger"
-    "github.com/pressly/goose/v3"
-)
-
-const migrationsDir = "internal/platform/postgres/migrations"
-
-func main() {
-    // Migration command-line flags
-    migrateCmd := flag.String("migrate", "", "Run database migrations (up|down|create|status|version)")
-    migrationName := flag.String("name", "", "Name for new migration file (used with -migrate=create)")
-    flag.Parse()
-
-    // Initialize logger early for startup messages
-    slog.Info("Scry API Server starting...")
-
-    // Load configuration
-    cfg, err := config.Load()
-    if err != nil {
-        slog.Error("Failed to load configuration", "error", err)
-        os.Exit(1)
-    }
-
-    // Setup structured logging
-    _, err = logger.Setup(cfg.Server)
-    if err != nil {
-        slog.Error("Failed to set up logger", "error", err)
-        os.Exit(1)
-    }
-
-    // Handle migration command if specified
-    if *migrateCmd != "" {
-        err := runMigrations(cfg, *migrateCmd, *migrationName)
-        if err != nil {
-            slog.Error("Migration command failed", "command", *migrateCmd, "error", err)
-            os.Exit(1)
-        }
-        slog.Info("Migration command finished successfully", "command", *migrateCmd)
-        os.Exit(0)
-    }
-
-    // Normal server startup logic
-    slog.Info("Scry API Server initialized successfully", "port", cfg.Server.Port)
-    // ... rest of the server logic ...
-}
-
-// runMigrations handles the execution of database migrations
-func runMigrations(cfg *config.Config, command string, args ...string) error {
-    slog.Info("Running migration command", "command", command, "dir", migrationsDir)
-
-    // Configure goose logger
-    goose.SetLogger(&slogGooseLogger{})
-
-    // Open database connection
-    db, err := sql.Open("pgx", cfg.Database.URL)
-    if err != nil {
-        return fmt.Errorf("failed to open database connection: %w", err)
-    }
-    defer db.Close()
-
-    if err := db.Ping(); err != nil {
-        return fmt.Errorf("failed to ping database: %w", err)
-    }
-
-    // Execute the goose command
-    switch command {
-    case "up":
-        return goose.Up(db, migrationsDir)
-    case "down":
-        return goose.Down(db, migrationsDir)
-    case "status":
-        return goose.Status(db, migrationsDir)
-    case "create":
-        if len(args) == 0 || args[0] == "" {
-            return fmt.Errorf("migration name must be provided for 'create' command")
-        }
-        return goose.Create(db, migrationsDir, args[0], "sql")
-    case "version":
-        return goose.Version(db, migrationsDir)
-    default:
-        return fmt.Errorf("unknown migration command: %s", command)
-    }
-}
-
-// slogGooseLogger adapts slog for goose's logger interface
-type slogGooseLogger struct{}
-
-func (l *slogGooseLogger) Printf(format string, v ...interface{}) {
-    slog.Info(fmt.Sprintf(format, v...))
-}
-
-func (l *slogGooseLogger) Fatalf(format string, v ...interface{}) {
-    slog.Error(fmt.Sprintf(format, v...))
-    os.Exit(1)
-}
-```
-
-### 3.2. Example Initial Migration
-
-Creating an example initial migration to verify the setup:
-
-```bash
-go run ./cmd/server/main.go -migrate=create -name=init_schema
-```
-
-This will create `internal/platform/postgres/migrations/yyyymmddhhmmss_init_schema.sql` with placeholders for `-- +goose Up` and `-- +goose Down` sections.
-
-## 4. Alignment with Project Standards
-
-### 4.1. Simplicity and Clarity (CORE_PRINCIPLES.md)
-This approach embraces simplicity by:
-- Using plain SQL files for migrations (clear, easy to understand format)
-- Minimizing setup complexity with a straightforward integration pattern
-- Using a battle-tested library with a clean API
-- Keeping the migration command interface simple and intuitive
-
-### 4.2. Separation of Concerns (ARCHITECTURE_GUIDELINES.md)
-This approach maintains proper separation by:
-- Keeping migration-related code in a dedicated function
-- Integrating with existing configuration and logging systems
-- Focusing each migration file on a single schema change
-- Allowing separation between the concerns of running the application and managing migrations
-
-### 4.3. Testability (TESTING_STRATEGY.md)
-The approach supports testability by:
-- Making migration commands callable as pure functions for unit testing
-- Allowing integration tests with test containers to verify actual migration behavior
-- Supporting isolation of migration functionality from the rest of the application
-
-### 4.4. Coding Conventions (CODING_STANDARDS.md)
-The implementation follows project coding standards by:
-- Using strong typing throughout
-- Employing standard Go practices for error handling
-- Using the established logging framework
-- Adding descriptive comments to explain the migration process
-
-### 4.5. Documentation (DOCUMENTATION_APPROACH.md)
-The plan ensures proper documentation by:
-- Adding clear usage instructions to README.md
-- Using descriptive migration file names
-- Adding comments to explain migration file formats
-- Documenting commands and options
-
-## 5. Potential Challenges and Mitigation
-
-1. **Database Connection Errors:**
-   - Implement proper error handling and clear error messages
-   - Provide guidance in docs for common connection issues
-
-2. **Migration Conflicts:**
-   - Use a consistent naming convention with timestamps
-   - Document best practices for creating migrations
-   - Ensure CI/CD pipeline catches conflicts before deployment
-
-3. **Migration Performance:**
-   - For large migrations that might lock tables, consider breaking them into smaller operations
-   - Document migration performance considerations
-
-## 6. Implementation Summary
-
-This plan provides a clean, maintainable, and integrated approach to database migrations for the Scry API. By using the `pressly/goose` library with SQL migration files, we achieve a good balance of simplicity, power, and integration with the existing codebase.
-
-The implementation avoids external dependencies outside the Go ecosystem and leverages the application's existing configuration and logging systems. Migration commands will be accessible through the main application binary, providing a cohesive experience for developers and in CI/CD environments.
diff --git a/TODO.md b/TODO.md
deleted file mode 100644
index 2f6bda7..0000000
--- a/TODO.md
+++ /dev/null
@@ -1,75 +0,0 @@
-# TODO
-
-## Design Principles (CORE_PRINCIPLES.md)
-- [x] **Refactor UserCardStats Mutability:** Remove mutable methods from `UserCardStats` and rely solely on `srs.Service`.
-  - **Action:** Delete the `UpdateReview` and `PostponeReview` methods from `internal/domain/user_card_stats.go`. Refactor any code that currently calls these methods to use the corresponding methods in `internal/domain/srs/service.go` instead, ensuring immutability is maintained. Update relevant tests.
-  - **Depends On:** None
-  - **AC Ref:** Design Principles Issue 1
-
-- [x] **Correct Ease Factor DB Constraint:** Align the database check constraint for `ease_factor` with the defined algorithm minimum.
-  - **Action:** Modify the SQL `CHECK` constraint in `internal/platform/postgres/migrations/20250415000004_create_user_card_stats_table.sql` from `CHECK (ease_factor > 1.0 AND ease_factor <= 2.5)` to `CHECK (ease_factor >= 1.3 AND ease_factor <= 2.5)`. Ensure the corresponding down migration (if applicable) is correct or add a new migration if necessary.
-  - **Depends On:** None
-  - **AC Ref:** Design Principles Issue 2
-
-- [x] **Remove Redundant Local Dev Test Helpers:** Eliminate helper functions in `local_postgres_test.go` that duplicate existing configuration files.
-  - **Action:** Delete the `generateDockerComposeYml` and `generateInitScript` functions from `infrastructure/local_dev/local_postgres_test.go`. Update the tests (e.g., `TestLocalPostgresSetup`) to assume the `docker-compose.yml` and `init-scripts/01-init.sql` files exist in their expected locations relative to the test file.
-  - **Depends On:** None
-  - **AC Ref:** Design Principles Issue 3
-
-## Architectural Patterns (ARCHITECTURE_GUIDELINES.md)
-- [x] **Refactor slogGooseLogger Fatalf:** Prevent `slogGooseLogger.Fatalf` from exiting the application directly.
-  - **Action:** Remove the `os.Exit(1)` call from the `Fatalf` method in `cmd/server/main.go`'s `slogGooseLogger`. Modify the `runMigrations` function to return the error encountered during `goose` operations. Update the `main` function's migration handling block to check for errors returned by `runMigrations` and call `os.Exit(1)` there if an error occurred.
-  - **Depends On:** None
-  - **AC Ref:** Architectural Patterns Issue 1
-
-- [x] **Add Explicit DB Password Management in Terraform:** Introduce a Terraform variable for the database user password.
-  - **Action:** Define a new `variable "database_password"` in `infrastructure/terraform/variables.tf` (mark as sensitive). Update the `digitalocean_database_user` resource in `infrastructure/terraform/main.tf` to use this variable for the password instead of relying on auto-generation. Update `terraform.tfvars.example` and any relevant documentation.
-  - **Depends On:** None
-  - **AC Ref:** Architectural Patterns Issue 2
-
-## Code Quality (CODING_STANDARDS.md)
-- [x] **Enhance DB Connection Error Handling:** Add specific error type checks for database connection attempts.
-  - **Action:** In `cmd/server/main.go` within the `runMigrations` function's `db.PingContext` error handling block (lines ~220-248), add specific checks using `errors.Is` or type assertions for common connection errors (e.g., `context.DeadlineExceeded`, `pgconn.PgError` for authentication failures, network errors) to provide more informative error messages.
-  - **Depends On:** None
-  - **AC Ref:** Code Quality Issue 1
-
-- [x] **Add TODO for Robust Email Validation:** Mark the basic email validation for future improvement.
-  - **Action:** Add a `// TODO:` comment above the `validateEmailFormat` function in `internal/domain/user.go` indicating that the current implementation is basic and should be replaced with a more robust validation library in a future task.
-  - **Depends On:** None
-  - **AC Ref:** Code Quality Issue 2
-
-## Test Quality (TESTING_STRATEGY.md)
-- [x] **Use Relative Paths in Migration Syntax Test:** Refactor `TestMigrationsValidSyntax` to avoid absolute paths.
-  - **Action:** Modify the path construction logic in `cmd/server/migrations_test.go` (lines ~79-83) for `TestMigrationsValidSyntax`. Instead of constructing an absolute path based on `os.Getwd()`, use a relative path from the test file's location or determine the project root reliably. Consider using `filepath.Abs` on the relative path if an absolute path is still required by `goose.CollectMigrations`.
-  - **Depends On:** None
-  - **AC Ref:** Test Quality Issue 1
-
-- [x] **Use filepath.Join in Local Postgres Test:** Refactor `TestLocalPostgresSetup` to use `filepath.Join`.
-  - **Action:** Modify the path construction logic in `infrastructure/local_dev/local_postgres_test.go` (line ~23). Replace the hardcoded relative path concatenation for finding `docker-compose.yml` with `filepath.Join(".", "docker-compose.yml")` or similar to correctly refer to the file relative to the working directory.
-  - **Depends On:** None
-  - **AC Ref:** Test Quality Issue 2
-
-- [x] **Enhance Terraform Test Validation:** Improve Terraform tests to verify database connectivity.
-  - **Action:** Modify the `TestTerraformDatabaseInfrastructure` test in `infrastructure/terraform/test/terraform_test.go`. After `terraform.InitAndApply`, use the `connection_string` output to establish a database connection, perform a `Ping()` to verify connectivity, and optionally attempt to run a simple query or apply migrations.
-  - **Depends On:** Add Explicit DB Password Management in Terraform
-  - **AC Ref:** Test Quality Issue 3
-
-## Documentation Practices (DOCUMENTATION_APPROACH.md)
-- [x] **Add Godoc Comments to SRS Algorithm Functions:** Document core SRS calculation functions.
-  - **Action:** Add comprehensive Godoc comments to the functions `calculateNewEaseFactor`, `calculateNewInterval`, `calculateNextReviewDate`, and `calculateNextStats` in `internal/domain/srs/algorithm.go`. Explain the purpose, parameters, return values, and any relevant algorithmic details for each function.
-  - **Depends On:** None
-  - **AC Ref:** Documentation Practices Issue 1
-
-- [x] **Document SRS Lapse Handling Multiplier:** Clarify the 'Good' outcome multiplier after a lapse in SRS design docs.
-  - **Action:** Update the `docs/design/srs_algorithm.md` document. Add a specific point under "Lapse Handling" or within the interval calculation description explaining the use of the `1.5` multiplier for the "Good" outcome immediately following an "Again" outcome (lapse). Include the rationale for this specific value.
-  - **Depends On:** Refactor UserCardStats Mutability
-  - **AC Ref:** Documentation Practices Issue 2
-
-## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS
-- [ ] **Issue/Assumption:** Acceptance Criteria References
-  - **Context:** The `PLAN.md` (Code Review) does not have explicit AC IDs.
-  - **Assumption:** The `AC Ref` fields in this `TODO.md` refer to the specific numbered issues within each section of the `PLAN.md` (Code Review) document (e.g., "Design Principles Issue 1", "Test Quality Issue 3").
-
-- [ ] **Issue/Assumption:** Exit handling in slogGooseLogger.Fatalf
-  - **Context:** PLAN.md Section Architectural Patterns 1 shows `slogGooseLogger.Fatalf` calling `os.Exit(1)`, but `main.go` already handles exits.
-  - **Assumption:** The `slogGooseLogger.Fatalf` implementation should only log the error using `slog.Error` and not call `os.Exit(1)`. The `runMigrations` function will return errors to `main` which handles program exit consistently.
diff --git a/create-database-schema-migrations-TASK.md b/create-database-schema-migrations-TASK.md
deleted file mode 100644
index 11753a5..0000000
--- a/create-database-schema-migrations-TASK.md
+++ /dev/null
@@ -1,166 +0,0 @@
-# Create Initial Database Schema Migrations
-
-## Task Description
-Implement initial database schema migrations for all domain models (User, Memo, Card, UserCardStats). This includes creating SQL migration files with proper "up" and "down" migrations, ensuring appropriate constraints and indexes, and following PostgreSQL best practices.
-
-## Acceptance Criteria
-1. Create migration files for all required tables using the goose migration format
-2. Define `users` table with appropriate fields matching the User domain model
-3. Define `memos` table with appropriate fields matching the Memo domain model, including status field
-4. Define `cards` table with appropriate fields including JSONB content structure
-5. Define `user_card_stats` table with appropriate fields for SRS algorithm
-6. Add essential indexes for performance, especially on `next_review_at` field
-7. All tables must have proper foreign key constraints
-8. Implement complete "down" migrations for rollback scenarios
-9. Ensure all migrations follow PostgreSQL best practices
-10. Migrations must be reversible (down migration should completely undo up migration)
-
-## Depends On
-- Database migration framework implementation ✓
-- Core domain models implementation ✓
-
-# EXECUTE
-
-## 1. SELECT AND ASSESS TASK
-
-- **Goal:** Choose and assess the next appropriate task from `TODO.MD`.
-- **Actions:**
-    - Scan `TODO.MD` for tasks marked `[ ]` (incomplete). Select the first task whose prerequisites (`Depends On:`) are already marked `[x]` (complete) or are 'None'.
-    - Record the exact Task Title.
-    - Mark the task as in-progress by changing `[ ]` to `[~]` in `TODO.MD`.
-    - **Assess Complexity:** Analyze the task requirements, determining if it's:
-        - **Simple:** Small change, single file, clear requirements, no architecture changes
-        - **Complex:** Multiple files, complex logic, architectural considerations, or any uncertainty
-    - **Route Accordingly:**
-        - For **Simple** tasks, follow Section 2 (Fast Track)
-        - For **Complex** tasks, follow Section 3 (Comprehensive Track)
-
-## 2. FAST TRACK (SIMPLE TASKS)
-
-### 2.1. CREATE MINIMAL PLAN
-
-- **Goal:** Document a straightforward implementation approach.
-- **Actions:**
-    - **Analyze:** Review the task details from `TODO.MD`.
-    - **Document:** Create `<sanitized-task-title>-PLAN.md` with:
-        - Task title
-        - Brief implementation approach (1-2 sentences)
-
-### 2.2. WRITE MINIMAL TESTS (IF APPLICABLE)
-
-- **Goal:** Define happy path tests only.
-- **Actions:**
-    - Write minimal tests for the core happy path
-    - Skip if task isn't directly testable
-
-### 2.3. IMPLEMENT FUNCTIONALITY
-
-- **Goal:** Write clean, simple code to satisfy requirements.
-- **Actions:**
-    - Consult project standards documents as needed
-    - Implement the functionality directly
-
-### 2.4. FINALIZE & COMMIT
-
-- **Goal:** Ensure work passes checks and is recorded.
-- **Actions:**
-    - Run checks (linting, tests) and fix any issues
-    - Update task status in `TODO.MD` to `[x]` (complete)
-    - Commit with conventional commit format
-
-## 3. COMPREHENSIVE TRACK (COMPLEX TASKS)
-
-### 3.1. PREPARE TASK PROMPT
-
-- **Goal:** Create a detailed prompt for implementation planning.
-- **Actions:**
-    - **Filename:** Sanitize Task Title -> `<sanitized-task-title>-TASK.md`.
-    - **Analyze:** Re-read task details (Action, AC Ref, Depends On) from `TODO.MD` and the relevant section in `PLAN.MD`.
-    - **Retrieve Base Prompt:** Copy the content from `prompts/execute.md` to use as the base for your task prompt.
-    - **Customize Prompt:** Create `<sanitized-task-title>-TASK.md` by adding task-specific details to the base prompt:
-        - Add task title, description, and acceptance criteria at the top.
-        - Keep all the original instructions from the base prompt.
-        - Ensure the prompt maintains the focus on standards alignment.
-
-### 3.2. GENERATE IMPLEMENTATION PLAN WITH ARCHITECT
-
-- **Goal:** Use `architect` to generate an implementation plan based on the task prompt and project context.
-- **Actions:**
-    - **Find Task Context:**
-        1. Find the top ten most relevant files for task-specific context
-    - **Run Architect:**
-        1. Run `architect --instructions <sanitized-task-title>-TASK.md --output-dir architect_output --model gemini-2.5-pro-exp-03-25 --model gemini-2.0-flash docs/DEVELOPMENT_PHILOSOPHY.md [top-ten-relevant-files]`
-        2. After architect finishes, review all files in the architect_output directory (typically gemini-2.5-pro-exp-03-25.md and gemini-2.0-flash.md).
-        3. ***Think hard*** about the different model outputs and create a single synthesized file that combines the best elements and insights from all outputs: `<sanitized-task-title>-PLAN.md`
-    - If you encounter an error, write it to a persistent logfile and try again.
-    - Report success/failure. Stop on unresolvable errors.
-    - **Review Plan:** Verify the implementation plan aligns with our standards hierarchy:
-        1. Simplicity and clarity over cleverness (`CORE_PRINCIPLES.md`)
-        2. Clean separation of concerns (`ARCHITECTURE_GUIDELINES.md`)
-        3. Straightforward testability with minimal mocking (`TESTING_STRATEGY.md`)
-        4. Adherence to coding conventions (`CODING_STANDARDS.md`)
-        5. Support for clear documentation (`DOCUMENTATION_APPROACH.md`)
-    - Remove `<sanitized-task-title>-TASK.md`.
-
-### 3.3. WRITE FAILING TESTS
-
-- **Goal:** Define expected behavior via tests, adhering strictly to the testing philosophy.
-- **Actions:**
-    - **Consult All Standards:** Review task requirements (`AC Ref:`, `<sanitized-task-title>-PLAN.md`) and adhere to all standards, with particular focus on testing:
-        - Ensure tests reflect the simplicity principle (`CORE_PRINCIPLES.md`)
-        - Test through public interfaces as defined in the architecture (`ARCHITECTURE_GUIDELINES.md`)
-        - Follow coding standards in test code too (`CODING_STANDARDS.md`)
-        - **Strictly adhere to testing principles, avoiding mocks of internal components** (`TESTING_STRATEGY.md`)
-        - Document test rationale where needed (`DOCUMENTATION_APPROACH.md`)
-    - **Write Happy Path Tests:** Write the minimum tests needed to verify the core *behavior* for the happy path, focusing on the public interface. **Prioritize tests that avoid mocking internal components.**
-    - **Write Critical Edge Case Tests:** Add tests for important error conditions or edge cases identified.
-    - **Verify Test Simplicity:** ***Think hard*** - "Are these tests simple? Do they avoid complex setup? Do they rely on mocking internal code? If yes, reconsider the test approach itself."
-    - Ensure tests currently fail (as appropriate for TDD/BDD style).
-- **Guidance:** Test *behavior*, not implementation. **Aggressively avoid unnecessary mocks.** If mocking seems unavoidable for internal logic, it's a signal to improve the design.
-
-### 3.4. IMPLEMENT FUNCTIONALITY
-
-- **Goal:** Write the minimal code needed to make tests pass (green).
-- **Actions:**
-    - **Consult Standards:** Review `CONTRIBUTING.MD`, `CODING_STANDARDS.md`, `ARCHITECTURE_GUIDELINES.md`, etc.
-    - **Write Code:** Implement the functionality based on `<sanitized-task-title>-PLAN.md` that satisfies the failing tests.
-    - **Focus on Passing Tests:** Initially implement just enough code to make tests pass, deferring optimization.
-    - **Adhere Strictly:** Follow project standards and the chosen plan.
-- **Guidance:** Focus on making tests pass first, then improve the implementation in the refactoring phase.
-
-### 3.5. REFACTOR FOR STANDARDS COMPLIANCE
-
-- **Goal:** Improve code quality while maintaining passing tests.
-- **Actions:**
-    - **Review Code:** Analyze the code files just implemented to ensure they pass tests.
-    - **Assess Standards Compliance:** ***Think hard*** and evaluate against all standards:
-        - **Core Principles:** "Does this implementation embrace simplicity? Does it have clear responsibilities? Is it explicit rather than implicit?" (`CORE_PRINCIPLES.md`)
-        - **Architecture:** "Is there clean separation between core logic and infrastructure? Are dependencies pointing inward?" (`ARCHITECTURE_GUIDELINES.md`)
-        - **Code Quality:** "Does it follow our coding conventions? Does it leverage types effectively? Does it prefer immutability?" (`CODING_STANDARDS.md`)
-        - **Testability:** "Can this code be tested simply? Does it require complex setup or extensive mocking of internal components?" (`TESTING_STRATEGY.md`)
-        - **Documentation:** "Are design decisions clear? Would comments explain the 'why' not just the 'what'?" (`DOCUMENTATION_APPROACH.md`)
-    - **Identify Refactors:** If any standard is not met, identify the **minimal necessary refactoring** to address the issues:
-        - For simplicity issues: Extract responsibilities, reduce complexity
-        - For architectural issues: Improve separation of concerns, realign dependencies
-        - For code quality issues: Apply coding conventions, use types more effectively
-        - For testability issues: Reduce coupling, extract pure functions, improve interfaces
-        - For documentation issues: Clarify design decisions with appropriate comments
-    - **Perform Refactor:** Apply the identified refactoring changes while ensuring tests continue to pass.
-
-### 3.6. VERIFY ALL TESTS PASS
-
-- **Goal:** Ensure all tests pass with the refactored implementation.
-- **Actions:**
-    - Run the code and all tests.
-    - Verify that all tests pass, including the original failing tests and any additional tests added.
-    - If any tests fail after refactoring, fix the implementation while maintaining standards compliance.
-    - **Do NOT modify tests to make them pass unless the test itself was fundamentally flawed.**
-
-### 3.7. FINALIZE & COMMIT
-
-- **Goal:** Ensure work is complete, passes all checks, and is recorded.
-- **Actions:**
-    - **Run Checks & Fix:** Execute linting, building, and the **full test suite**. Fix *any* code issues causing failures.
-    - **Update Task Status:** Change the task status in `TODO.MD` from `[~]` (in progress) to `[x]` (complete).
-    - **Remove Task-Specific Reference Files:** Delete <sanitized-task-title>-PLAN.md
-    - **Add, Commit, and Push Changes**
diff --git a/go.mod b/go.mod
index e0ef6bd..4178e61 100644
--- a/go.mod
+++ b/go.mod
@@ -12,6 +12,7 @@ require (
 	github.com/pressly/goose/v3 v3.24.2
 	github.com/spf13/viper v1.20.1
 	github.com/stretchr/testify v1.10.0
+	golang.org/x/crypto v0.37.0
 )
 
 require (
@@ -56,7 +57,6 @@ require (
 	github.com/ulikunitz/xz v0.5.10 // indirect
 	github.com/zclconf/go-cty v1.15.0 // indirect
 	go.uber.org/multierr v1.11.0 // indirect
-	golang.org/x/crypto v0.37.0 // indirect
 	golang.org/x/mod v0.18.0 // indirect
 	golang.org/x/net v0.39.0 // indirect
 	golang.org/x/sync v0.13.0 // indirect
diff --git a/internal/domain/user.go b/internal/domain/user.go
index 18a1768..794caf7 100644
--- a/internal/domain/user.go
+++ b/internal/domain/user.go
@@ -9,10 +9,13 @@ import (
 
 // Common validation errors
 var (
-	ErrEmptyUserID         = errors.New("user ID cannot be empty")
-	ErrInvalidEmail        = errors.New("invalid email format")
-	ErrEmptyEmail          = errors.New("email cannot be empty")
-	ErrPasswordTooShort    = errors.New("password must be at least 8 characters long")
+	ErrEmptyUserID        = errors.New("user ID cannot be empty")
+	ErrInvalidEmail       = errors.New("invalid email format")
+	ErrEmptyEmail         = errors.New("email cannot be empty")
+	ErrPasswordTooShort   = errors.New("password must be at least 8 characters long")
+	ErrPasswordNotComplex = errors.New(
+		"password must contain at least one uppercase letter, one lowercase letter, one number, and one special character",
+	)
 	ErrEmptyPassword       = errors.New("password cannot be empty")
 	ErrEmptyHashedPassword = errors.New("hashed password cannot be empty")
 )
@@ -22,21 +25,25 @@ var (
 type User struct {
 	ID             uuid.UUID `json:"id"`
 	Email          string    `json:"email"`
+	Password       string    `json:"-"` // Plaintext password, used temporarily during registration/updates
 	HashedPassword string    `json:"-"` // Never expose password hash in JSON
 	CreatedAt      time.Time `json:"created_at"`
 	UpdatedAt      time.Time `json:"updated_at"`
 }
 
-// NewUser creates a new User with the given email and hashed password.
+// NewUser creates a new User with the given email and password.
 // It generates a new UUID for the user ID and sets the creation/update timestamps.
 // Returns an error if validation fails.
-func NewUser(email, hashedPassword string) (*User, error) {
+//
+// NOTE: This function only sets up the user structure with the plaintext password.
+// The caller is responsible for hashing the password before storing the user.
+func NewUser(email, password string) (*User, error) {
 	user := &User{
-		ID:             uuid.New(),
-		Email:          email,
-		HashedPassword: hashedPassword,
-		CreatedAt:      time.Now().UTC(),
-		UpdatedAt:      time.Now().UTC(),
+		ID:        uuid.New(),
+		Email:     email,
+		Password:  password, // Plaintext password - must be hashed before storage
+		CreatedAt: time.Now().UTC(),
+		UpdatedAt: time.Now().UTC(),
 	}
 
 	if err := user.Validate(); err != nil {
@@ -63,8 +70,24 @@ func (u *User) Validate() error {
 		return ErrInvalidEmail
 	}
 
-	if u.HashedPassword == "" {
-		return ErrEmptyHashedPassword
+	// Password validation
+	// During user creation/update we need to validate the provided password
+	if u.Password != "" {
+		// When plaintext password is provided, validate its complexity
+		if len(u.Password) < 8 {
+			return ErrPasswordTooShort
+		}
+
+		// Additional password complexity checks
+		if !validatePasswordComplexity(u.Password) {
+			return ErrPasswordNotComplex
+		}
+	} else {
+		// When no plaintext password is provided, the user must have a hashed password
+		// (this would be the case for existing users in the database)
+		if u.HashedPassword == "" {
+			return ErrEmptyPassword
+		}
 	}
 
 	return nil
@@ -113,3 +136,40 @@ func validateEmailFormat(email string) bool {
 
 	return true
 }
+
+// validatePasswordComplexity checks if a password meets complexity requirements:
+// - At least one uppercase letter
+// - At least one lowercase letter
+// - At least one number
+// - At least one special character
+func validatePasswordComplexity(password string) bool {
+	var (
+		hasUpper   bool
+		hasLower   bool
+		hasNumber  bool
+		hasSpecial bool
+	)
+
+	specialChars := "!@#$%^&*()-_+={}[]|:;\"'<>,.?/~`"
+
+	for _, char := range password {
+		switch {
+		case 'A' <= char && char <= 'Z':
+			hasUpper = true
+		case 'a' <= char && char <= 'z':
+			hasLower = true
+		case '0' <= char && char <= '9':
+			hasNumber = true
+		default:
+			// Check if char is in specialChars
+			for _, special := range specialChars {
+				if char == special {
+					hasSpecial = true
+					break
+				}
+			}
+		}
+	}
+
+	return hasUpper && hasLower && hasNumber && hasSpecial
+}
diff --git a/internal/domain/user_test.go b/internal/domain/user_test.go
index e3b8511..6a41df2 100644
--- a/internal/domain/user_test.go
+++ b/internal/domain/user_test.go
@@ -9,7 +9,7 @@ import (
 func TestNewUser(t *testing.T) {
 	// Test valid user creation
 	validEmail := "test@example.com"
-	validPassword := "hashedpassword123"
+	validPassword := "Password123!"
 
 	user, err := NewUser(validEmail, validPassword)
 
@@ -25,8 +25,8 @@ func TestNewUser(t *testing.T) {
 		t.Errorf("Expected email %s, got %s", validEmail, user.Email)
 	}
 
-	if user.HashedPassword != validPassword {
-		t.Errorf("Expected hashed password %s, got %s", validPassword, user.HashedPassword)
+	if user.Password != validPassword {
+		t.Errorf("Expected password %s, got %s", validPassword, user.Password)
 	}
 
 	if user.CreatedAt.IsZero() {
@@ -50,8 +50,20 @@ func TestNewUser(t *testing.T) {
 
 	// Test invalid password
 	_, err = NewUser(validEmail, "")
-	if err != ErrEmptyHashedPassword {
-		t.Errorf("Expected error %v, got %v", ErrEmptyHashedPassword, err)
+	if err != ErrEmptyPassword {
+		t.Errorf("Expected error %v, got %v", ErrEmptyPassword, err)
+	}
+
+	// Test password too short
+	_, err = NewUser(validEmail, "Pass1!")
+	if err != ErrPasswordTooShort {
+		t.Errorf("Expected error %v, got %v", ErrPasswordTooShort, err)
+	}
+
+	// Test password complexity
+	_, err = NewUser(validEmail, "password123")
+	if err != ErrPasswordNotComplex {
+		t.Errorf("Expected error %v, got %v", ErrPasswordNotComplex, err)
 	}
 }
 
@@ -87,11 +99,20 @@ func TestUserValidate(t *testing.T) {
 		t.Errorf("Expected error %v, got %v", ErrInvalidEmail, err)
 	}
 
-	// Test invalid password
+	// Test both password fields empty
 	invalidUser = validUser
 	invalidUser.HashedPassword = ""
-	if err := invalidUser.Validate(); err != ErrEmptyHashedPassword {
-		t.Errorf("Expected error %v, got %v", ErrEmptyHashedPassword, err)
+	if err := invalidUser.Validate(); err != ErrEmptyPassword {
+		t.Errorf("Expected error %v, got %v", ErrEmptyPassword, err)
+	}
+
+	// When Password is provided, check that password validation is done
+	// and HashedPassword validation is skipped
+	invalidUser = validUser
+	invalidUser.Password = "abc"    // Too short
+	invalidUser.HashedPassword = "" // Would normally cause ErrEmptyHashedPassword
+	if err := invalidUser.Validate(); err != ErrPasswordTooShort {
+		t.Errorf("Expected error %v, got %v", ErrPasswordTooShort, err)
 	}
 }
 
@@ -124,3 +145,113 @@ func TestValidateEmailFormat(t *testing.T) {
 		}
 	}
 }
+
+func TestUserValidate_PasswordComplexity(t *testing.T) {
+	tests := []struct {
+		name     string
+		password string
+		wantErr  error
+	}{
+		{
+			name:     "valid password with all requirements",
+			password: "Password123!",
+			wantErr:  nil,
+		},
+		{
+			name:     "password too short",
+			password: "Pass1!",
+			wantErr:  ErrPasswordTooShort,
+		},
+		{
+			name:     "password missing uppercase",
+			password: "password123!",
+			wantErr:  ErrPasswordNotComplex,
+		},
+		{
+			name:     "password missing lowercase",
+			password: "PASSWORD123!",
+			wantErr:  ErrPasswordNotComplex,
+		},
+		{
+			name:     "password missing number",
+			password: "Password!",
+			wantErr:  ErrPasswordNotComplex,
+		},
+		{
+			name:     "password missing special character",
+			password: "Password123",
+			wantErr:  ErrPasswordNotComplex,
+		},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			user := &User{
+				ID:             uuid.New(),
+				Email:          "test@example.com",
+				Password:       tt.password,
+				HashedPassword: "some-hashed-password", // Not validated when Password is present
+			}
+
+			err := user.Validate()
+
+			if tt.wantErr != nil {
+				if err != tt.wantErr {
+					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
+				}
+			} else {
+				if err != nil {
+					t.Errorf("Expected no error, got %v", err)
+				}
+			}
+		})
+	}
+}
+
+func TestValidatePasswordComplexity(t *testing.T) {
+	tests := []struct {
+		name     string
+		password string
+		want     bool
+	}{
+		{
+			name:     "valid password with all requirements",
+			password: "Password123!",
+			want:     true,
+		},
+		{
+			name:     "password missing uppercase",
+			password: "password123!",
+			want:     false,
+		},
+		{
+			name:     "password missing lowercase",
+			password: "PASSWORD123!",
+			want:     false,
+		},
+		{
+			name:     "password missing number",
+			password: "Password!",
+			want:     false,
+		},
+		{
+			name:     "password missing special character",
+			password: "Password123",
+			want:     false,
+		},
+		{
+			name:     "password with different special characters",
+			password: "Password123@#$%^&*()-_=+[]{}|;:,.<>?/~`",
+			want:     true,
+		},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			got := validatePasswordComplexity(tt.password)
+			if got != tt.want {
+				t.Errorf("validatePasswordComplexity() = %v, want %v", got, tt.want)
+			}
+		})
+	}
+}
diff --git a/internal/platform/postgres/user_store.go b/internal/platform/postgres/user_store.go
new file mode 100644
index 0000000..b5bd424
--- /dev/null
+++ b/internal/platform/postgres/user_store.go
@@ -0,0 +1,376 @@
+package postgres
+
+import (
+	"context"
+	"database/sql"
+	"errors"
+	"log/slog"
+	"time"
+
+	"github.com/google/uuid"
+	"github.com/jackc/pgx/v5/pgconn"
+	"github.com/phrazzld/scry-api/internal/domain"
+	"github.com/phrazzld/scry-api/internal/platform/logger"
+	"github.com/phrazzld/scry-api/internal/store"
+	"golang.org/x/crypto/bcrypt"
+)
+
+// PostgreSQL error codes
+const uniqueViolationCode = "23505" // PostgreSQL unique violation error code
+
+// PostgresUserStore implements the store.UserStore interface
+// using a PostgreSQL database as the storage backend.
+type PostgresUserStore struct {
+	db *sql.DB
+}
+
+// DB returns the underlying database connection for testing purposes.
+// This method is not part of the store.UserStore interface.
+func (s *PostgresUserStore) DB() *sql.DB {
+	return s.db
+}
+
+// NewPostgresUserStore creates a new PostgreSQL implementation of the UserStore interface.
+// It accepts a database connection that should be initialized and managed by the caller.
+func NewPostgresUserStore(db *sql.DB) *PostgresUserStore {
+	return &PostgresUserStore{
+		db: db,
+	}
+}
+
+// isUniqueViolation checks if the given error is a PostgreSQL unique constraint violation.
+// This is used to detect when an operation fails due to a unique constraint,
+// such as duplicate email addresses.
+func isUniqueViolation(err error) bool {
+	var pgErr *pgconn.PgError
+	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
+		return true
+	}
+	return false
+}
+
+// Ensure PostgresUserStore implements store.UserStore interface
+var _ store.UserStore = (*PostgresUserStore)(nil)
+
+// Create implements store.UserStore.Create
+// It creates a new user in the database, handling domain validation and password hashing.
+// Returns store.ErrEmailExists if a user with the same email already exists.
+func (s *PostgresUserStore) Create(ctx context.Context, user *domain.User) error {
+	// Get the logger from context or use default
+	log := logger.FromContext(ctx)
+
+	// First, validate the user data
+	if err := user.Validate(); err != nil {
+		log.Warn("user validation failed during create",
+			slog.String("error", err.Error()),
+			slog.String("email", user.Email))
+		return err
+	}
+
+	// Hash the password using bcrypt
+	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
+	if err != nil {
+		log.Error("failed to hash password",
+			slog.String("error", err.Error()))
+		return err
+	}
+
+	// Store the hashed password and clear the plaintext password from memory
+	user.HashedPassword = string(hashedPassword)
+	user.Password = "" // Clear plaintext password from memory for security
+
+	// Start a transaction to ensure data consistency
+	tx, err := s.db.BeginTx(ctx, nil)
+	if err != nil {
+		log.Error("failed to begin transaction",
+			slog.String("error", err.Error()))
+		return err
+	}
+	// Defer a rollback in case anything fails
+	defer func() {
+		// If error occurs, attempt to rollback
+		if err != nil {
+			if rbErr := tx.Rollback(); rbErr != nil {
+				log.Error("failed to rollback transaction",
+					slog.String("rollback_error", rbErr.Error()),
+					slog.String("original_error", err.Error()))
+			}
+		}
+	}()
+
+	// Insert the user into the database
+	_, err = tx.ExecContext(ctx, `
+		INSERT INTO users (id, email, hashed_password, created_at, updated_at)
+		VALUES ($1, $2, $3, $4, $5)
+	`, user.ID, user.Email, user.HashedPassword, user.CreatedAt, user.UpdatedAt)
+
+	if err != nil {
+		// Check for unique constraint violation (duplicate email)
+		if isUniqueViolation(err) {
+			log.Warn("attempt to create user with existing email",
+				slog.String("email", user.Email))
+			return store.ErrEmailExists
+		}
+		// Log other errors
+		log.Error("failed to insert user",
+			slog.String("error", err.Error()),
+			slog.String("email", user.Email))
+		return err
+	}
+
+	// Commit the transaction
+	if err = tx.Commit(); err != nil {
+		log.Error("failed to commit transaction",
+			slog.String("error", err.Error()),
+			slog.String("email", user.Email))
+		return err
+	}
+
+	log.Info("user created successfully",
+		slog.String("user_id", user.ID.String()),
+		slog.String("email", user.Email))
+	return nil
+}
+
+// GetByID implements store.UserStore.GetByID
+// It retrieves a user by their unique ID from the database.
+// Returns store.ErrUserNotFound if the user does not exist.
+func (s *PostgresUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
+	// Get the logger from context or use default
+	log := logger.FromContext(ctx)
+
+	log.Debug("retrieving user by ID", slog.String("user_id", id.String()))
+
+	// Query the user from database
+	var user domain.User
+	err := s.db.QueryRowContext(ctx, `
+		SELECT id, email, hashed_password, created_at, updated_at
+		FROM users
+		WHERE id = $1
+	`, id).Scan(&user.ID, &user.Email, &user.HashedPassword, &user.CreatedAt, &user.UpdatedAt)
+
+	// Handle the result
+	if err != nil {
+		if errors.Is(err, sql.ErrNoRows) {
+			log.Debug("user not found", slog.String("user_id", id.String()))
+			return nil, store.ErrUserNotFound
+		}
+		log.Error("failed to query user by ID",
+			slog.String("user_id", id.String()),
+			slog.String("error", err.Error()))
+		return nil, err
+	}
+
+	// Ensure the Password field is empty as it should never be populated from the database
+	user.Password = ""
+
+	log.Debug("user retrieved successfully", slog.String("user_id", id.String()))
+	return &user, nil
+}
+
+// GetByEmail implements store.UserStore.GetByEmail
+// It retrieves a user by their email address from the database.
+// Returns store.ErrUserNotFound if the user does not exist.
+// The email matching is case-insensitive.
+func (s *PostgresUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
+	// Get the logger from context or use default
+	log := logger.FromContext(ctx)
+
+	log.Debug("retrieving user by email",
+		slog.String("email", email))
+
+	// Query the user from database with case-insensitive email matching
+	var user domain.User
+	err := s.db.QueryRowContext(ctx, `
+		SELECT id, email, hashed_password, created_at, updated_at
+		FROM users
+		WHERE LOWER(email) = LOWER($1)
+	`, email).Scan(&user.ID, &user.Email, &user.HashedPassword, &user.CreatedAt, &user.UpdatedAt)
+
+	// Handle the result
+	if err != nil {
+		if errors.Is(err, sql.ErrNoRows) {
+			log.Debug("user not found", slog.String("email", email))
+			return nil, store.ErrUserNotFound
+		}
+		log.Error("failed to query user by email",
+			slog.String("email", email),
+			slog.String("error", err.Error()))
+		return nil, err
+	}
+
+	// Ensure the Password field is empty as it should never be populated from the database
+	user.Password = ""
+
+	log.Debug("user retrieved successfully",
+		slog.String("user_id", user.ID.String()),
+		slog.String("email", user.Email))
+	return &user, nil
+}
+
+// Update implements store.UserStore.Update
+// It modifies an existing user's details in the database.
+// Returns store.ErrUserNotFound if the user does not exist.
+// Returns store.ErrEmailExists if updating to an email that already exists.
+// Returns validation errors from the domain User if data is invalid.
+func (s *PostgresUserStore) Update(ctx context.Context, user *domain.User) error {
+	// Get the logger from context or use default
+	log := logger.FromContext(ctx)
+
+	log.Debug("updating user", slog.String("user_id", user.ID.String()))
+
+	// First, validate the user data
+	if err := user.Validate(); err != nil {
+		log.Warn("user validation failed during update",
+			slog.String("error", err.Error()),
+			slog.String("user_id", user.ID.String()))
+		return err
+	}
+
+	// Update the timestamp
+	user.UpdatedAt = time.Now().UTC()
+
+	// Determine password hash to store
+	var hashedPasswordToStore string
+
+	if user.Password != "" {
+		// Hash the new password if provided
+		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
+		if err != nil {
+			log.Error("failed to hash password",
+				slog.String("error", err.Error()),
+				slog.String("user_id", user.ID.String()))
+			return err
+		}
+		hashedPasswordToStore = string(hashedPassword)
+		user.Password = "" // Clear plaintext password for security
+	} else {
+		// Fetch the existing password hash if not updating the password
+		err := s.db.QueryRowContext(ctx, `
+			SELECT hashed_password FROM users WHERE id = $1
+		`, user.ID).Scan(&hashedPasswordToStore)
+
+		if err != nil {
+			if errors.Is(err, sql.ErrNoRows) {
+				log.Debug("user not found", slog.String("user_id", user.ID.String()))
+				return store.ErrUserNotFound
+			}
+			log.Error("failed to fetch existing user",
+				slog.String("error", err.Error()),
+				slog.String("user_id", user.ID.String()))
+			return err
+		}
+	}
+
+	// Start a transaction for the update
+	tx, err := s.db.BeginTx(ctx, nil)
+	if err != nil {
+		log.Error("failed to begin transaction",
+			slog.String("error", err.Error()),
+			slog.String("user_id", user.ID.String()))
+		return err
+	}
+	// Defer a rollback in case anything fails
+	defer func() {
+		if err != nil {
+			if rbErr := tx.Rollback(); rbErr != nil {
+				log.Error("failed to rollback transaction",
+					slog.String("rollback_error", rbErr.Error()),
+					slog.String("original_error", err.Error()),
+					slog.String("user_id", user.ID.String()))
+			}
+		}
+	}()
+
+	// Execute the update statement
+	result, err := tx.ExecContext(ctx, `
+		UPDATE users
+		SET email = $1, hashed_password = $2, updated_at = $3
+		WHERE id = $4
+	`, user.Email, hashedPasswordToStore, user.UpdatedAt, user.ID)
+
+	if err != nil {
+		// Check for unique constraint violation (duplicate email)
+		if isUniqueViolation(err) {
+			log.Warn("email already exists",
+				slog.String("email", user.Email),
+				slog.String("user_id", user.ID.String()))
+			return store.ErrEmailExists
+		}
+		// Log other errors
+		log.Error("failed to update user",
+			slog.String("error", err.Error()),
+			slog.String("user_id", user.ID.String()))
+		return err
+	}
+
+	// Check if a row was actually updated
+	rowsAffected, err := result.RowsAffected()
+	if err != nil {
+		log.Error("failed to get rows affected",
+			slog.String("error", err.Error()),
+			slog.String("user_id", user.ID.String()))
+		return err
+	}
+
+	// If no rows were affected, the user didn't exist
+	if rowsAffected == 0 {
+		log.Debug("user not found for update", slog.String("user_id", user.ID.String()))
+		return store.ErrUserNotFound
+	}
+
+	// Commit the transaction
+	if err = tx.Commit(); err != nil {
+		log.Error("failed to commit transaction",
+			slog.String("error", err.Error()),
+			slog.String("user_id", user.ID.String()))
+		return err
+	}
+
+	log.Info("user updated successfully",
+		slog.String("user_id", user.ID.String()),
+		slog.String("email", user.Email))
+	return nil
+}
+
+// Delete implements store.UserStore.Delete
+// It removes a user from the database by their ID.
+// Returns store.ErrUserNotFound if the user does not exist.
+func (s *PostgresUserStore) Delete(ctx context.Context, id uuid.UUID) error {
+	// Get the logger from context or use default
+	log := logger.FromContext(ctx)
+
+	log.Debug("deleting user by ID", slog.String("user_id", id.String()))
+
+	// Execute the DELETE statement
+	result, err := s.db.ExecContext(ctx, `
+		DELETE FROM users
+		WHERE id = $1
+	`, id)
+
+	// Handle execution errors
+	if err != nil {
+		log.Error("failed to execute delete statement",
+			slog.String("user_id", id.String()),
+			slog.String("error", err.Error()))
+		return err
+	}
+
+	// Check if a row was actually deleted
+	rowsAffected, err := result.RowsAffected()
+	if err != nil {
+		log.Error("failed to get rows affected",
+			slog.String("user_id", id.String()),
+			slog.String("error", err.Error()))
+		return err
+	}
+
+	// If no rows were affected, the user didn't exist
+	if rowsAffected == 0 {
+		log.Debug("user not found for deletion", slog.String("user_id", id.String()))
+		return store.ErrUserNotFound
+	}
+
+	log.Info("user deleted successfully", slog.String("user_id", id.String()))
+	return nil
+}
diff --git a/internal/platform/postgres/user_store_test.go b/internal/platform/postgres/user_store_test.go
new file mode 100644
index 0000000..eccc6a5
--- /dev/null
+++ b/internal/platform/postgres/user_store_test.go
@@ -0,0 +1,709 @@
+package postgres_test
+
+import (
+	"context"
+	"database/sql"
+	"fmt"
+	"strings"
+	"testing"
+	"time"
+
+	"github.com/google/uuid"
+	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
+	"github.com/phrazzld/scry-api/internal/domain"
+	"github.com/phrazzld/scry-api/internal/platform/postgres"
+	"github.com/phrazzld/scry-api/internal/store"
+	"github.com/phrazzld/scry-api/internal/testutils"
+	"github.com/stretchr/testify/assert"
+	"github.com/stretchr/testify/require"
+)
+
+const testTimeout = 5 * time.Second
+
+// setupTestDB opens a database connection and ensures a clean test environment
+// by dropping and recreating the users table.
+func setupTestDB(t *testing.T) *sql.DB {
+	if !testutils.IsIntegrationTestEnvironment() {
+		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
+	}
+
+	// Get database URL from environment
+	dbURL := testutils.GetTestDatabaseURL(t)
+
+	// Connect to the database
+	db, err := sql.Open("pgx", dbURL)
+	require.NoError(t, err, "Failed to open database connection")
+
+	// Set connection pool parameters
+	db.SetMaxOpenConns(5)
+	db.SetMaxIdleConns(5)
+	db.SetConnMaxLifetime(5 * time.Minute)
+
+	// Create a context with timeout for DB operations
+	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+	defer cancel()
+
+	// Ping the database to ensure connection is alive
+	err = db.PingContext(ctx)
+	require.NoError(t, err, "Failed to ping database")
+
+	// Recreate the test table to ensure a clean state
+	// Drop the table if it exists
+	_, err = db.ExecContext(ctx, "DROP TABLE IF EXISTS users")
+	require.NoError(t, err, "Failed to drop users table")
+
+	// Create the table with the same schema as in migrations
+	createTableSQL := `
+	CREATE TABLE users (
+		id UUID PRIMARY KEY,
+		email VARCHAR(255) UNIQUE NOT NULL,
+		hashed_password TEXT NOT NULL,
+		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
+		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
+	);
+
+	CREATE INDEX idx_users_email ON users(email);
+	`
+	_, err = db.ExecContext(ctx, createTableSQL)
+	require.NoError(t, err, "Failed to create users table")
+
+	return db
+}
+
+// teardownTestDB closes the database connection and performs any needed cleanup
+func teardownTestDB(t *testing.T, db *sql.DB) {
+	if db != nil {
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Clean up the test data by dropping the table
+		_, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS users")
+		if err != nil {
+			t.Logf("Warning: Failed to drop users table during cleanup: %v", err)
+		}
+
+		err = db.Close()
+		if err != nil {
+			t.Logf("Warning: Failed to close database connection: %v", err)
+		}
+	}
+}
+
+// createTestUser is a helper function to create a valid test user
+//
+//nolint:unused
+func createTestUser(t *testing.T) *domain.User {
+	email := fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
+	user, err := domain.NewUser(email, "Password123!")
+	require.NoError(t, err, "Failed to create test user")
+	return user
+}
+
+// insertTestUser inserts a user directly into the database for testing
+//
+//nolint:unused
+func insertTestUser(t *testing.T, db *sql.DB, email string) uuid.UUID {
+	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+	defer cancel()
+
+	// Generate a unique ID
+	id := uuid.New()
+	hashedPassword := "$2a$10$abcdefghijklmnopqrstuvwxyz0123456789"
+
+	// Insert the user directly
+	_, err := db.ExecContext(ctx, `
+		INSERT INTO users (id, email, hashed_password, created_at, updated_at)
+		VALUES ($1, $2, $3, NOW(), NOW())
+	`, id, email, hashedPassword)
+	require.NoError(t, err, "Failed to insert test user directly")
+
+	return id
+}
+
+// getUserByID retrieves a user from the database directly for verification
+//
+//nolint:unused
+func getUserByID(t *testing.T, db *sql.DB, id uuid.UUID) *domain.User {
+	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+	defer cancel()
+
+	// Query the user
+	var user domain.User
+	err := db.QueryRowContext(ctx, `
+		SELECT id, email, hashed_password, created_at, updated_at
+		FROM users
+		WHERE id = $1
+	`, id).Scan(&user.ID, &user.Email, &user.HashedPassword, &user.CreatedAt, &user.UpdatedAt)
+
+	if err != nil {
+		if err == sql.ErrNoRows {
+			return nil
+		}
+		require.NoError(t, err, "Failed to query user by ID")
+	}
+
+	return &user
+}
+
+// countUsers counts the number of users in the database matching certain criteria
+//
+//nolint:unused
+func countUsers(t *testing.T, db *sql.DB, whereClause string, args ...interface{}) int {
+	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+	defer cancel()
+
+	query := "SELECT COUNT(*) FROM users"
+	if whereClause != "" {
+		query += " WHERE " + whereClause
+	}
+
+	var count int
+	err := db.QueryRowContext(ctx, query, args...).Scan(&count)
+	require.NoError(t, err, "Failed to count users")
+
+	return count
+}
+
+// TestNewPostgresUserStore verifies the constructor works correctly
+func TestNewPostgresUserStore(t *testing.T) {
+	// Set up the test database
+	db := setupTestDB(t)
+	defer teardownTestDB(t, db)
+
+	// Initialize the store
+	userStore := postgres.NewPostgresUserStore(db)
+
+	// Assertions
+	assert.NotNil(t, userStore, "PostgresUserStore should be created successfully")
+	assert.Same(t, db, userStore.DB(), "Store should hold the provided database connection")
+
+	// Verify the implementation satisfies the interface
+	var _ store.UserStore = userStore
+}
+
+// TestBasicDatabaseConnectivity verifies the test environment works correctly
+func TestBasicDatabaseConnectivity(t *testing.T) {
+	// Set up the test database
+	db := setupTestDB(t)
+	defer teardownTestDB(t, db)
+
+	// Create a context with timeout
+	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+	defer cancel()
+
+	// Test basic database connectivity by inserting and querying a sample record
+	testUUID := uuid.New()
+	email := fmt.Sprintf("integration-test-%s@example.com", testUUID.String()[:8])
+	hashedPassword := "hashed_password_placeholder"
+
+	// Direct SQL insert to verify connection
+	_, err := db.ExecContext(ctx, `
+		INSERT INTO users (id, email, hashed_password, created_at, updated_at)
+		VALUES ($1, $2, $3, NOW(), NOW())
+	`, testUUID, email, hashedPassword)
+	require.NoError(t, err, "Failed to insert test record directly")
+
+	// Direct SQL query to verify insertion
+	var count int
+	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&count)
+	require.NoError(t, err, "Failed to query test record")
+	assert.Equal(t, 1, count, "Should have inserted exactly one record")
+}
+
+// TestPostgresUserStore_Create tests the Create method
+func TestPostgresUserStore_Create(t *testing.T) {
+	// Set up the test database
+	db := setupTestDB(t)
+	defer teardownTestDB(t, db)
+
+	// Create a new user store
+	userStore := postgres.NewPostgresUserStore(db)
+
+	// Test Case 1: Successful user creation
+	t.Run("Successful user creation", func(t *testing.T) {
+		// Create a test user
+		user := createTestUser(t)
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Call the Create method
+		err := userStore.Create(ctx, user)
+
+		// Verify the result
+		require.NoError(t, err, "User creation should succeed")
+
+		// Verify the user was inserted into the database
+		dbUser := getUserByID(t, db, user.ID)
+		require.NotNil(t, dbUser, "User should exist in the database")
+		assert.Equal(t, user.ID, dbUser.ID, "User ID should match")
+		assert.Equal(t, user.Email, dbUser.Email, "User email should match")
+		assert.NotEmpty(t, dbUser.HashedPassword, "Hashed password should not be empty")
+		assert.Empty(t, user.Password, "Plaintext password should be cleared")
+
+		// Verify timestamps
+		assert.False(t, dbUser.CreatedAt.IsZero(), "CreatedAt should not be zero")
+		assert.False(t, dbUser.UpdatedAt.IsZero(), "UpdatedAt should not be zero")
+	})
+
+	// Test Case 2: Attempt to create user with existing email
+	t.Run("Duplicate email", func(t *testing.T) {
+		// Create a test user
+		email := fmt.Sprintf("duplicate-%s@example.com", uuid.New().String()[:8])
+
+		// Insert the first user directly into the database
+		insertTestUser(t, db, email)
+
+		// Create a second user with the same email
+		user, err := domain.NewUser(email, "Password123!")
+		require.NoError(t, err, "Creating user struct should succeed")
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Call the Create method
+		err = userStore.Create(ctx, user)
+
+		// Verify the result
+		assert.ErrorIs(
+			t,
+			err,
+			store.ErrEmailExists,
+			"Creating user with duplicate email should fail with ErrEmailExists",
+		)
+
+		// Verify there's still only one user with this email
+		count := countUsers(t, db, "email = $1", email)
+		assert.Equal(t, 1, count, "There should still be only one user with this email")
+	})
+
+	// Test Case 3: Attempt to create user with invalid data
+	t.Run("Invalid user data", func(t *testing.T) {
+		// Create a test user with invalid email
+		user, err := domain.NewUser("not-an-email", "Password123!")
+		require.Error(t, err, "Creating user with invalid email should fail validation")
+		assert.Nil(t, user, "User should be nil after validation failure")
+
+		// Create context
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Since we could not create a user with an invalid email through the constructor,
+		// let's create a valid user first and then modify it to have invalid data
+		user = createTestUser(t)
+		user.Email = "not-an-email" // This will fail validation
+
+		// Call the Create method
+		err = userStore.Create(ctx, user)
+
+		// Verify the result
+		assert.Error(t, err, "Creating user with invalid email should fail")
+		assert.Equal(t, domain.ErrInvalidEmail, err, "Error should be ErrInvalidEmail")
+
+		// Verify no user was created
+		count := countUsers(t, db, "email = $1", "not-an-email")
+		assert.Equal(t, 0, count, "No user should be created with invalid email")
+	})
+
+	// Test Case 4: Attempt to create user with weak password
+	t.Run("Weak password", func(t *testing.T) {
+		// Create context
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Create a user with valid email but weak password
+		user := &domain.User{
+			ID:        uuid.New(),
+			Email:     fmt.Sprintf("weak-password-%s@example.com", uuid.New().String()[:8]),
+			Password:  "password", // Missing complexity requirements
+			CreatedAt: time.Now().UTC(),
+			UpdatedAt: time.Now().UTC(),
+		}
+
+		// Call the Create method
+		err := userStore.Create(ctx, user)
+
+		// Verify the result
+		assert.Error(t, err, "Creating user with weak password should fail")
+		assert.Equal(t, domain.ErrPasswordNotComplex, err, "Error should be ErrPasswordNotComplex")
+
+		// Verify no user was created
+		count := countUsers(t, db, "email = $1", user.Email)
+		assert.Equal(t, 0, count, "No user should be created with weak password")
+	})
+}
+
+// TestPostgresUserStore_GetByID tests the GetByID method
+func TestPostgresUserStore_GetByID(t *testing.T) {
+	// Set up the test database
+	db := setupTestDB(t)
+	defer teardownTestDB(t, db)
+
+	// Create a new user store
+	userStore := postgres.NewPostgresUserStore(db)
+
+	// Test Case 1: Successfully retrieve existing user by ID
+	t.Run("Successfully retrieve existing user", func(t *testing.T) {
+		// Insert a test user directly into the database
+		email := fmt.Sprintf("getbyid-test-%s@example.com", uuid.New().String()[:8])
+		userId := insertTestUser(t, db, email)
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Call the GetByID method
+		user, err := userStore.GetByID(ctx, userId)
+
+		// Verify the result
+		require.NoError(t, err, "GetByID should succeed for existing user")
+		require.NotNil(t, user, "Retrieved user should not be nil")
+		assert.Equal(t, userId, user.ID, "User ID should match")
+		assert.Equal(t, email, user.Email, "User email should match")
+		assert.NotEmpty(t, user.HashedPassword, "Hashed password should not be empty")
+		assert.Empty(t, user.Password, "Plaintext password should be empty")
+		assert.False(t, user.CreatedAt.IsZero(), "CreatedAt should not be zero")
+		assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt should not be zero")
+	})
+
+	// Test Case 2: Attempt to retrieve non-existent user
+	t.Run("Non-existent user", func(t *testing.T) {
+		// Generate a random UUID that doesn't exist in the database
+		nonExistentID := uuid.New()
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Call the GetByID method
+		user, err := userStore.GetByID(ctx, nonExistentID)
+
+		// Verify the result
+		assert.Error(t, err, "GetByID should return error for non-existent user")
+		assert.ErrorIs(t, err, store.ErrUserNotFound, "Error should be ErrUserNotFound")
+		assert.Nil(t, user, "User should be nil for non-existent ID")
+	})
+}
+
+// TestPostgresUserStore_GetByEmail tests the GetByEmail method
+func TestPostgresUserStore_GetByEmail(t *testing.T) {
+	// Set up the test database
+	db := setupTestDB(t)
+	defer teardownTestDB(t, db)
+
+	// Create a new user store
+	userStore := postgres.NewPostgresUserStore(db)
+
+	// Test Case 1: Successfully retrieve existing user by email
+	t.Run("Successfully retrieve existing user", func(t *testing.T) {
+		// Insert a test user directly into the database
+		email := fmt.Sprintf("getbyemail-test-%s@example.com", uuid.New().String()[:8])
+		userId := insertTestUser(t, db, email)
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Call the GetByEmail method
+		user, err := userStore.GetByEmail(ctx, email)
+
+		// Verify the result
+		require.NoError(t, err, "GetByEmail should succeed for existing user")
+		require.NotNil(t, user, "Retrieved user should not be nil")
+		assert.Equal(t, userId, user.ID, "User ID should match")
+		assert.Equal(t, email, user.Email, "User email should match")
+		assert.NotEmpty(t, user.HashedPassword, "Hashed password should not be empty")
+		assert.Empty(t, user.Password, "Plaintext password should be empty")
+		assert.False(t, user.CreatedAt.IsZero(), "CreatedAt should not be zero")
+		assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt should not be zero")
+	})
+
+	// Test Case 2: Attempt to retrieve user with non-existent email
+	t.Run("Non-existent email", func(t *testing.T) {
+		// Use an email that doesn't exist in the database
+		nonExistentEmail := fmt.Sprintf("nonexistent-%s@example.com", uuid.New().String())
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Call the GetByEmail method
+		user, err := userStore.GetByEmail(ctx, nonExistentEmail)
+
+		// Verify the result
+		assert.Error(t, err, "GetByEmail should return error for non-existent email")
+		assert.ErrorIs(t, err, store.ErrUserNotFound, "Error should be ErrUserNotFound")
+		assert.Nil(t, user, "User should be nil for non-existent email")
+	})
+
+	// Test Case 3: Case insensitivity for email matching
+	t.Run("Case insensitive email matching", func(t *testing.T) {
+		// Insert a test user with lowercase email
+		email := fmt.Sprintf("casesensitive-%s@example.com", uuid.New().String()[:8])
+		userId := insertTestUser(t, db, email)
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Query with uppercase email
+		upperEmail := strings.ToUpper(email)
+		user, err := userStore.GetByEmail(ctx, upperEmail)
+
+		// Verify the result (should find the user despite case difference)
+		require.NoError(t, err, "GetByEmail should be case insensitive")
+		require.NotNil(t, user, "Retrieved user should not be nil")
+		assert.Equal(t, userId, user.ID, "User ID should match")
+		assert.Equal(t, email, user.Email, "User email should match original case")
+	})
+}
+
+// TestPostgresUserStore_Update tests the Update method
+func TestPostgresUserStore_Update(t *testing.T) {
+	// Set up the test database
+	db := setupTestDB(t)
+	defer teardownTestDB(t, db)
+
+	// Create a new user store
+	userStore := postgres.NewPostgresUserStore(db)
+
+	// Test Case 1: Successfully update existing user with a new email but same password
+	t.Run("Update email only", func(t *testing.T) {
+		// Insert a test user directly into the database
+		oldEmail := fmt.Sprintf("update-test-email-%s@example.com", uuid.New().String()[:8])
+		userId := insertTestUser(t, db, oldEmail)
+
+		// Fetch the user to get current hashed password and timestamps
+		originalUser := getUserByID(t, db, userId)
+		require.NotNil(t, originalUser, "User should exist before update")
+		oldHash := originalUser.HashedPassword
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Create an updated user (change email but not password)
+		newEmail := fmt.Sprintf("updated-email-%s@example.com", uuid.New().String()[:8])
+		updatedUser := &domain.User{
+			ID:        userId,
+			Email:     newEmail,
+			Password:  "", // No password update
+			CreatedAt: originalUser.CreatedAt,
+			UpdatedAt: originalUser.UpdatedAt, // Will be updated by the method
+		}
+
+		// Call the Update method
+		err := userStore.Update(ctx, updatedUser)
+
+		// Verify the result
+		require.NoError(t, err, "Update should succeed for existing user")
+
+		// Verify the user was updated in the database
+		updatedDbUser := getUserByID(t, db, userId)
+		require.NotNil(t, updatedDbUser, "User should still exist after update")
+		assert.Equal(t, userId, updatedDbUser.ID, "User ID should not change")
+		assert.Equal(t, newEmail, updatedDbUser.Email, "Email should be updated")
+		assert.Equal(t, oldHash, updatedDbUser.HashedPassword, "Password hash should remain unchanged")
+		assert.True(t, updatedDbUser.UpdatedAt.After(originalUser.UpdatedAt), "UpdatedAt should be updated")
+		assert.Equal(t, originalUser.CreatedAt, updatedDbUser.CreatedAt, "CreatedAt should not change")
+	})
+
+	// Test Case 2: Successfully update existing user with a new password but same email
+	t.Run("Update password only", func(t *testing.T) {
+		// Insert a test user directly into the database
+		email := fmt.Sprintf("update-test-pwd-%s@example.com", uuid.New().String()[:8])
+		userId := insertTestUser(t, db, email)
+
+		// Fetch the user to get current hashed password and timestamps
+		originalUser := getUserByID(t, db, userId)
+		require.NotNil(t, originalUser, "User should exist before update")
+		oldHash := originalUser.HashedPassword
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Create an updated user (change password but not email)
+		newPassword := "NewPassword123!"
+		updatedUser := &domain.User{
+			ID:        userId,
+			Email:     email,       // Same email
+			Password:  newPassword, // New password
+			CreatedAt: originalUser.CreatedAt,
+			UpdatedAt: originalUser.UpdatedAt, // Will be updated by the method
+		}
+
+		// Call the Update method
+		err := userStore.Update(ctx, updatedUser)
+
+		// Verify the result
+		require.NoError(t, err, "Update should succeed for existing user")
+
+		// Verify the user was updated in the database
+		updatedDbUser := getUserByID(t, db, userId)
+		require.NotNil(t, updatedDbUser, "User should still exist after update")
+		assert.Equal(t, userId, updatedDbUser.ID, "User ID should not change")
+		assert.Equal(t, email, updatedDbUser.Email, "Email should remain unchanged")
+		assert.NotEqual(t, oldHash, updatedDbUser.HashedPassword, "Password hash should be updated")
+		assert.True(t, updatedDbUser.UpdatedAt.After(originalUser.UpdatedAt), "UpdatedAt should be updated")
+		assert.Equal(t, originalUser.CreatedAt, updatedDbUser.CreatedAt, "CreatedAt should not change")
+		assert.Empty(t, updatedUser.Password, "Plaintext password should be cleared")
+	})
+
+	// Test Case 3: Attempt to update non-existent user
+	t.Run("Non-existent user", func(t *testing.T) {
+		// Generate a random UUID that doesn't exist in the database
+		nonExistentID := uuid.New()
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Create a user update with the non-existent ID
+		user := &domain.User{
+			ID:        nonExistentID,
+			Email:     "nonexistent@example.com",
+			CreatedAt: time.Now().UTC(),
+			UpdatedAt: time.Now().UTC(),
+		}
+
+		// Call the Update method
+		err := userStore.Update(ctx, user)
+
+		// Verify the result
+		assert.Error(t, err, "Update should return error for non-existent user")
+		assert.ErrorIs(t, err, store.ErrUserNotFound, "Error should be ErrUserNotFound")
+	})
+
+	// Test Case 4: Attempt to update email to one that already exists
+	t.Run("Duplicate email", func(t *testing.T) {
+		// Insert two test users
+		existingEmail := fmt.Sprintf("existing-email-%s@example.com", uuid.New().String()[:8])
+		existingID := insertTestUser(t, db, existingEmail)
+
+		updateEmail := fmt.Sprintf("update-email-%s@example.com", uuid.New().String()[:8])
+		updateID := insertTestUser(t, db, updateEmail)
+
+		// Get original user data
+		originalUser := getUserByID(t, db, updateID)
+		require.NotNil(t, originalUser, "User should exist before update")
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Create an updated user (change email to one that already exists)
+		updatedUser := &domain.User{
+			ID:        updateID,
+			Email:     existingEmail, // Email of the other user - should cause conflict
+			CreatedAt: originalUser.CreatedAt,
+			UpdatedAt: originalUser.UpdatedAt,
+		}
+
+		// Call the Update method
+		err := userStore.Update(ctx, updatedUser)
+
+		// Verify the result
+		assert.Error(t, err, "Update should return error for duplicate email")
+		assert.ErrorIs(t, err, store.ErrEmailExists, "Error should be ErrEmailExists")
+
+		// Verify the user was not updated
+		updatedDbUser := getUserByID(t, db, updateID)
+		require.NotNil(t, updatedDbUser, "User should still exist")
+		assert.Equal(t, updateEmail, updatedDbUser.Email, "Email should not be changed")
+
+		// Verify the other user was not affected
+		otherUser := getUserByID(t, db, existingID)
+		require.NotNil(t, otherUser, "Other user should still exist")
+		assert.Equal(t, existingEmail, otherUser.Email, "Other user's email should not change")
+	})
+
+	// Test Case 5: Update with invalid data
+	t.Run("Invalid data", func(t *testing.T) {
+		// Insert a test user
+		email := fmt.Sprintf("valid-email-%s@example.com", uuid.New().String()[:8])
+		userId := insertTestUser(t, db, email)
+
+		// Get original user data
+		originalUser := getUserByID(t, db, userId)
+		require.NotNil(t, originalUser, "User should exist before update")
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Create an updated user with invalid email
+		updatedUser := &domain.User{
+			ID:        userId,
+			Email:     "invalid-email", // Invalid email format
+			CreatedAt: originalUser.CreatedAt,
+			UpdatedAt: originalUser.UpdatedAt,
+		}
+
+		// Call the Update method
+		err := userStore.Update(ctx, updatedUser)
+
+		// Verify the result
+		assert.Error(t, err, "Update should return error for invalid data")
+		assert.Equal(t, domain.ErrInvalidEmail, err, "Error should be ErrInvalidEmail")
+
+		// Verify the user was not updated
+		updatedDbUser := getUserByID(t, db, userId)
+		require.NotNil(t, updatedDbUser, "User should still exist")
+		assert.Equal(t, email, updatedDbUser.Email, "Email should not be changed")
+	})
+}
+
+// TestPostgresUserStore_Delete tests the Delete method
+func TestPostgresUserStore_Delete(t *testing.T) {
+	// Set up the test database
+	db := setupTestDB(t)
+	defer teardownTestDB(t, db)
+
+	// Create a new user store
+	userStore := postgres.NewPostgresUserStore(db)
+
+	// Test Case 1: Successfully delete existing user
+	t.Run("Successfully delete existing user", func(t *testing.T) {
+		// Insert a test user directly into the database
+		email := fmt.Sprintf("delete-test-%s@example.com", uuid.New().String()[:8])
+		userId := insertTestUser(t, db, email)
+
+		// Verify user exists before deletion
+		beforeCount := countUsers(t, db, "id = $1", userId)
+		assert.Equal(t, 1, beforeCount, "User should exist before deletion")
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Call the Delete method
+		err := userStore.Delete(ctx, userId)
+
+		// Verify the result
+		require.NoError(t, err, "Delete should succeed for existing user")
+
+		// Verify user no longer exists
+		afterCount := countUsers(t, db, "id = $1", userId)
+		assert.Equal(t, 0, afterCount, "User should not exist after deletion")
+	})
+
+	// Test Case 2: Attempt to delete non-existent user
+	t.Run("Non-existent user", func(t *testing.T) {
+		// Generate a random UUID that doesn't exist in the database
+		nonExistentID := uuid.New()
+
+		// Create a context with timeout
+		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
+		defer cancel()
+
+		// Call the Delete method
+		err := userStore.Delete(ctx, nonExistentID)
+
+		// Verify the result
+		assert.Error(t, err, "Delete should return error for non-existent user")
+		assert.ErrorIs(t, err, store.ErrUserNotFound, "Error should be ErrUserNotFound")
+	})
+}
diff --git a/internal/store/user.go b/internal/store/user.go
new file mode 100644
index 0000000..434aec2
--- /dev/null
+++ b/internal/store/user.go
@@ -0,0 +1,49 @@
+package store
+
+import (
+	"context"
+	"errors"
+
+	"github.com/google/uuid"
+	"github.com/phrazzld/scry-api/internal/domain"
+)
+
+// Common store errors
+var (
+	// ErrUserNotFound indicates that the requested user does not exist in the store.
+	ErrUserNotFound = errors.New("user not found")
+
+	// ErrEmailExists indicates that a user with the given email already exists.
+	ErrEmailExists = errors.New("email already exists")
+)
+
+// UserStore defines the interface for user data persistence.
+type UserStore interface {
+	// Create saves a new user to the store.
+	// It handles domain validation and password hashing internally.
+	// Returns ErrEmailExists if the email is already taken.
+	// Returns validation errors from the domain User if data is invalid.
+	Create(ctx context.Context, user *domain.User) error
+
+	// GetByID retrieves a user by their unique ID.
+	// Returns ErrUserNotFound if the user does not exist.
+	// The returned user contains all fields except the plaintext password.
+	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
+
+	// GetByEmail retrieves a user by their email address.
+	// Returns ErrUserNotFound if the user does not exist.
+	// The returned user contains all fields except the plaintext password.
+	GetByEmail(ctx context.Context, email string) (*domain.User, error)
+
+	// Update modifies an existing user's details.
+	// It handles domain validation and password rehashing if needed.
+	// Returns ErrUserNotFound if the user does not exist.
+	// Returns ErrEmailExists if updating to an email that already exists.
+	// Returns validation errors from the domain User if data is invalid.
+	Update(ctx context.Context, user *domain.User) error
+
+	// Delete removes a user from the store by their ID.
+	// Returns ErrUserNotFound if the user does not exist.
+	// This operation is permanent and cannot be undone.
+	Delete(ctx context.Context, id uuid.UUID) error
+}
