# Remediation Plan – Sprint 15

## Executive Summary
This plan addresses critical issues from the code review, focusing on security, consistency, and Go idioms. We'll implement three focused fixes: (1) refactor handler mutability to return new instances following Go's idiomatic patterns, (2) ensure all error logging passes through redaction utilities to prevent sensitive data leaks, and (3) enforce consistent use of centralized error handling helpers across all handlers to eliminate duplication and enhance security.

## Strike List
| Seq | CR‑ID | Title                                  | Effort | Owner?  |
|-----|-------|----------------------------------------|--------|---------|
| 1   | cr‑03 | Fix Non-idiomatic Handler Mutability   | xs     | backend |
| 2   | cr‑02 | Implement Safe Error Logging           | s      | backend |
| 3   | cr‑01 | Enforce Centralized Error Handling     | s      | backend |

## Detailed Remedies
### cr‑03 Fix Non-idiomatic Handler Mutability
- **Problem:** The `WithTimeFunc` method on `AuthHandler` mutates the handler in place rather than returning a new instance.
- **Impact:** Violates Go's idiomatic immutability pattern, creating potential issues in concurrent tests and reducing predictability.
- **Chosen Fix:** Refactor `WithTimeFunc` to return a new `AuthHandler` instance with the updated time function.
- **Steps:**
  1. Modify `AuthHandler.WithTimeFunc` signature to return `*AuthHandler`
  2. Create a new `AuthHandler` struct inside the method, copying all fields from the original
  3. Set the `timeFunc` field on the new instance and return it
  4. Update any calling code (tests) to capture the returned instance
- **Done‑When:** Method returns new instances, tests pass, code follows Go idioms

### cr‑02 Implement Safe Error Logging
- **Problem:** Raw error objects are logged directly, potentially leaking sensitive details into logs.
- **Impact:** Security vulnerability exposing internal information (database connection strings, queries, credentials, stack traces).
- **Chosen Fix:** Ensure all error logging passes through redaction utilities before logging.
- **Steps:**
  1. Update `shared.RespondWithErrorAndLog` to use `redact.Error()` before logging
  2. Audit all API handlers for direct error logging (e.g., `logger.Error("Failed to create memo", "error", err, ...)`)
  3. Replace direct error logging with redacted versions (`redact.Error(err)`)
  4. Add tests to confirm no sensitive patterns appear in logs
- **Done‑When:** No unredacted error messages appear in logs, security scan confirms no sensitive data patterns

### cr‑01 Enforce Centralized Error Handling
- **Problem:** Error handling logic is duplicated across handlers; centralized helpers are not used consistently.
- **Impact:** DRY violation, inconsistent API error responses, increased maintenance burden, potential security risks.
- **Chosen Fix:** Replace all direct error responses with calls to the existing centralized helpers.
- **Steps:**
  1. In all handlers, identify direct error status mapping and response generation
  2. Replace validation errors with calls to `HandleValidationError(w, r, err)`
  3. Replace other errors with calls to `HandleAPIError(w, r, err, defaultMsg)`
  4. Remove all manual error mapping logic from handlers
  5. Ensure tests verify consistent error responses
- **Done‑When:** All handlers use centralized error helpers, tests pass, API responses are consistent

## Standards Alignment
- **Simplicity:** Removes duplicated error handling logic and improves clarity with standard helpers
- **Modularity:** Centralizes error handling for better separation of concerns; handler immutability improves isolation
- **Testability:** Immutable handlers are safer in concurrent tests; consistent error handling simplifies testing
- **Coding Std:** Adheres to Go idioms (immutability) and DRY principles
- **Security:** Redacts sensitive data from logs; ensures consistent sanitization of error messages in responses

## Validation Checklist
- Automated tests (unit, integration) pass
- Static analyzers (`golangci-lint`) show no issues
- Manual review confirms consistent error helper usage
- Test logs confirm no sensitive data leaks
- API responses show consistent error formatting
- No new lint or audit warnings introduced
