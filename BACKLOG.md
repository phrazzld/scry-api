# Scry API - MVP Backlog

**Repository:** `scry-api` (Go Backend)
**Version:** 1.0-MVP
**Date:** 2025-04-13

This backlog outlines the major work items required to build the Minimum Viable Product (MVP) for the Scry Go backend API, hosted entirely on DigitalOcean. Items should generally be tackled in order, aligning with our core principles and engineering guidelines. Each item represents a meaningful chunk of work, intended to be broken down further into detailed plans and tasks.

* **Card Review API Implementation:**
    * Implement `store.CardStore` function `GetNextReviewCard(userID time.Time)` using the defined query logic (filtering by `next_review_at`, ordering).
    * Implement Fetch Next Card endpoint (`GET /cards/next`), using the store function and handling the 204 No Content case.
    * Implement `store.UserCardStatsStore` function `UpdateStats(userID, cardID, outcome)`.
    * Implement Submit Answer endpoint (`POST /cards/{id}/answer`), validating the outcome, calling the `srs.Service` to calculate new stats, and updating the DB via the store.

* **Card Management API Implementation:**
    * Implement `store.CardStore` function for updating Card content (`content` JSONB).
    * Implement Edit Card endpoint (`PUT /cards/{id}`).
    * Implement `store.CardStore` function for deleting a Card and its associated `user_card_stats`.
    * Implement Delete Card endpoint (`DELETE /cards/{id}`).
    * Implement Postpone Card endpoint (`POST /cards/{id}/postpone`), calculating and updating `next_review_at` via the store.

* **API Structure & Server Setup:**
    * Set up main HTTP server entry point (`cmd/server/main.go`).
    * Implement request routing using `chi` router.
    * Integrate standard middleware: Logging (request logging), Recovery (panic recovery), Auth (JWT validation), CORS.
    * Implement basic request body validation.

* Implement semantic versioning (ideally automatically managed somehow ... conventional commits?)

* **Testing:**
    * Adhere strictly to `DEVELOPMENT_PHILOSOPHY.md`.
    * Implement unit tests for `srs` service, utility functions, domain logic.
    * Implement integration tests for critical API flows (Auth register/login, Memo submission & generation trigger, Card review cycle: get next -> answer -> get next). Mock only at external boundaries (define `Generator` interface mock, potentially `Store` interface mock *if not* using test containers). Consider `testcontainers-go` for PostgreSQL integration tests.

* **Deployment & Infrastructure (DigitalOcean):**
    * Create comprehensive `Dockerfile` for the Go application.
    * Configure deployment via DigitalOcean App Platform, including environment variables (DB connection string, JWT secret, Gemini API key, log level, etc.).
    * Ensure DO Managed Postgres is configured correctly (version, extensions).

* **Monitoring & Logging (Basic):**
    * Implement structured logging (JSON) throughout the application, including contextual information (user ID, request ID, memo ID, card ID).
    * Ensure critical errors (panic recovery, background job failures, failed LLM calls, DB errors) are logged at ERROR/FATAL level.
    * Set up basic alerting via DO App Platform monitoring or external service for critical error logs.
    * Define a basic health check endpoint (`/healthz`).

* **API Documentation:**
    * Implement OpenAPI/Swagger documentation for all API endpoints.
    * Set up automatic documentation generation using comments and annotations.
    * Create a documentation server endpoint to serve the OpenAPI UI.
    * Ensure documentation is comprehensive and includes examples, responses, and error codes.

* **Technical Debt & Refactoring:**
    * **Over-Aggressive Redaction Patterns**: Fine-tune regex patterns in redaction utilities to balance security with debuggability. Add more comprehensive tests for edge cases and potential false positives/negatives.
    * **Mixed Responsibilities in Test Utilities**: Refactor test helpers for better separation of concerns by breaking them into focused packages (e.g., HTTP helpers, entity creation, DB helpers).
    * **Documentation and Test Parallelization**: Add consistent godoc comments to all public functions, mark deprecated test helpers clearly, and add t.Parallel() to compatible table-driven subtests.

## Completed Items

* **Memo & Card Generation Implementation (Completed):**
    * ✅ Implement `store.MemoStore` and `store.CardStore` interfaces and Postgres implementations for Memo/Card/Stats persistence and status updates.
    * ✅ Implement Background Job logic (`internal/task/processor.go`): `GenerateCardsFromMemo(memoID, userID, memoText)`:
        * Update Memo status to `processing` in DB.
        * Call `generation.Generator.GenerateCardsFromMemo`.
        * Parse/validate the returned Card data structures.
        * Save the batch of generated Cards and their initial `user_card_stats` within a transaction.
        * **Error Handling:**
            * Handle individual Card/Stat saving failures with appropriate retries and logging.
            * Handle partial success cases with appropriate status updates (`completed_with_errors`).
            * Handle complete failures with proper error logging and status updates (`failed`).
            * Update status to `completed` on full success.
    * ✅ Implement Submit Memo endpoint (`POST /memos`): Authenticates user, saves Memo via `store` with `pending` status, enqueues `GenerateCardsFromMemo` job via event system and task service, returns HTTP 202 Accepted.

* **Asynchronous Task Runner Setup (Completed):**
    * ✅ Implement basic in-memory background task queue & worker pool (`internal/task`) using goroutines/channels.
    * ✅ Implement recovery mechanism: On application startup, query `memos` table for entries with `status = 'processing'`, enqueue generation tasks for these Memos to handle potential restarts during processing.
    * ✅ Implement comprehensive integration tests for task recovery and lifecycle.
    * ✅ Add pre-commit hook to fail on extremely long files while keeping warnings for moderately long files.

* **Set Up Database Migration Framework (Completed):**
    * ✅ Research and select a suitable database migration tool for the project (chosen: `pressly/goose`).
    * ✅ Set up the migration framework in the codebase.
    * ✅ Create migration directory structure.
    * ✅ Configure the migration tool to work with PostgreSQL.
    * ✅ Implement connection and migration execution logic.
    * ✅ Add proper logging and error handling.
    * ✅ Implement all migration commands (up, down, status, create, version).
    * ✅ Add unit tests for migration functionality.
    * ✅ Document migration usage in README.md and dedicated migration guide.

* **Define Core Domain Models (Completed):**
    * ✅ Define core domain models/structs in Go (`internal/domain`: `User`, `Memo`, `Card`, `UserCardStats`).
    * ✅ Implement strong typing following type standards (`DEVELOPMENT_PHILOSOPHY.md`).
    * ✅ Ensure models include necessary validation methods.
    * ✅ Document the domain model relationships and purpose.

* **Create Initial Database Schema Migrations (Completed):**
    * ✅ Create initial database schema migration scripts.
    * ✅ Define `users` table structure with appropriate fields and constraints.
    * ✅ Define `memos` table including `status` field ('pending', 'processing', 'completed', 'completed_with_errors', 'failed').
    * ✅ Define `cards` table with `content` JSONB structure.
    * ✅ Define `user_card_stats` table with appropriate fields.
    * ✅ Add essential indexes (esp. on `user_card_stats` for `next_review_at`).
    * ✅ Implement rollback migrations.

* **Provision Database Infrastructure (Completed):**
    * ✅ Provision DigitalOcean Managed PostgreSQL instance using Infrastructure as Code (Terraform).
    * ✅ Configure PostgreSQL settings for optimal performance.
    * ✅ Enable `pgvector` extension on the DO Managed Postgres instance.
    * ✅ Set up backup and monitoring.
    * ✅ Document connection and access procedures.
    * ✅ Create local development database setup with Docker.

* **Core Domain Logic Implementation (SRS) (Completed):**
    * ✅ Define `srs.Service` interface within the core domain/application layer.
    * ✅ Implement basic SRS algorithm logic (SM-2 variant) within the `srs` service.
    * ✅ Define precise MVP parameters (initial intervals, ease factors, lapse handling) in a separate design doc.
    * ✅ Adhere to pure function principles as specified in `DEVELOPMENT_PHILOSOPHY.md`

* **Pre-commit Hook Enhancement (Completed):**
    * ✅ Added pre-commit hook to warn (but not fail) when files are too long.
    * ✅ Implemented binary file detection and UTF-16 encoding support.
    * ✅ Added standard hooks for common validation tasks.
    * ✅ Organized and improved documentation in pre-commit config.
    * ✅ Enhanced README with comprehensive pre-commit hooks documentation.

* **Improve Test Data Isolation (Completed):**
    * ✅ Implemented transaction-based test isolation to enable parallel test execution.
    * ✅ Created DBTX interface to support both *sql.DB and *sql.Tx.
    * ✅ Refactored PostgresUserStore to use the DBTX interface.
    * ✅ Added WithTx helper function to testutils package.
    * ✅ Updated tests to use transaction-based isolation and enabled t.Parallel().

* **Logging Framework Setup (Completed):**
    * ✅ Set up basic structured logging framework using `log/slog`.
    * ✅ Configure the logging system using the application configuration.
    * ✅ Implement appropriate log levels and contextual logging helpers.

* **Project Setup & Configuration (Completed):**
    * ✅ Initialize Go module (`scry-api`) and standard project structure (e.g., `/cmd/server`, `/internal/domain`, `/internal/service`, `/internal/store`, `/internal/api`, `/internal/generation`, `/internal/task`, `/internal/config`, `/internal/platform/postgres`).

* **Configuration Management Implementation (Completed):**
    * ✅ Implement configuration loading (env vars primary, config files for local dev via Viper) adhering to `DEVELOPMENT_PHILOSOPHY.md`
    * ✅ Create a strongly-typed configuration structure containing all application settings.
    * ✅ Implement validation logic to ensure all required configuration values are present and valid.

* **CI/CD Setup (Completed):**
    * ✅ Set up GitHub Actions for continuous integration.
    * ✅ Configure workflows for:
      * Building and testing the code on every push and PR.
      * Linting code with golangci-lint.
      * Running security scanning.
      * Automating deployment to DigitalOcean on main branch merges.
    * ✅ Implement pre-commit hooks (using pre-commit framework) to ensure code quality:
      * gofmt/goimports for consistent formatting.
      * golangci-lint for catching issues early.
      * Commit message validation for conventional commits.
      * Potentially run fast tests pre-commit.

* **User Store Implementation (Completed):**
    * ✅ Define `store.UserStore` interface with methods for CRUD operations.
    * ✅ Implement PostgreSQL implementation (`internal/platform/postgres`) for user CRUD operations.
    * ✅ Implement secure password hashing using `bcrypt`.
    * ✅ Ensure validation of all user data before storage.
    * ✅ Add comprehensive tests for store implementation.

* **JWT Authentication Service (Completed):**
    * ✅ Implement JWT generation and validation logic in `auth.Service`.
    * ✅ Implement token refresh mechanisms.
    * ✅ Add necessary configuration for JWT secrets and token lifetimes.
    * ✅ Add comprehensive tests for authentication service.

* **Authentication API Endpoints (Completed):**
    * ✅ Implement User Registration endpoint (`POST /auth/register`) in `internal/api`.
    * ✅ Implement User Login endpoint (`POST /auth/login`).
    * ✅ Implement Token Refresh endpoint (`POST /auth/refresh`).
    * ✅ Ensure proper error handling and validation for all endpoints.
    * ✅ Add integration tests for authentication endpoints.

* **Authentication Middleware (Completed):**
    * ✅ Implement JWT validation middleware for protecting API routes.
    * ✅ Integrate middleware with the router.
    * ✅ Add tests for middleware functionality.

* **Generation Service Implementation (Completed):**
    * ✅ Define `generation.Generator` interface (e.g., `GenerateCardsFromMemo(...)`) within the core application layer.
    * ✅ Implement `geminiGenerator` struct implementing the `Generator` interface (`internal/platform/gemini` or similar).
        * ✅ Load prompt templates from external configuration (not hardcoded).
        * ✅ Implement logic to call Gemini API using the configured model(s).
        * ✅ Implement error handling & basic retry logic for transient Gemini API errors.
        * ✅ Securely load and manage the Gemini API key via configuration.
    * ✅ Design service to be swappable per architecture guidelines in `DEVELOPMENT_PHILOSOPHY.md`
