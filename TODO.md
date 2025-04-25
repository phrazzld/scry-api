# TODO List

## Backlog

### [Priority 1] Security Remediation
- [x] [T017] Harden error log output [SECURITY]
  - Description: Refactor `RespondWithErrorAndLog` to sanitize and redact sensitive information (stack traces, connection strings, PII) before logging or returning error responses. Implement a central redaction utility for known sensitive patterns.
  - Acceptance Criteria:
    - AC1: All calls to `RespondWithErrorAndLog` log only sanitized messages; no raw error strings.
    - AC2: A central redaction utility is implemented that handles common sensitive patterns (connection strings, passwords, API keys, file paths, stack traces).
    - AC3: No direct logging of raw `error.Error()` strings in API handlers or shared response logic.
    - AC4: Integration tests verify no sensitive data appears in logs or HTTP responses under various error conditions.
  - Depends On: []
  - Estimated Story Points: 5

- [x] [T018] Strengthen trace ID generation & error handling [SECURITY]
  - Description: Replace the 8-byte trace ID with a 16-byte (32 hex chars) or UUIDv4 implementation. Implement robust error handling for trace ID generation with proper logging and secure fallback mechanism.
  - Acceptance Criteria:
    - AC1: `generateTraceID` produces 32-char hex strings or valid UUIDv4.
    - AC2: Errors during trace ID generation are logged at ERROR level with context.
    - AC3: On failure, a securely generated non-constant ID is returned; no static fallback.
    - AC4: Tests verify trace ID uniqueness, format, and proper error handling.
  - Depends On: []
  - Estimated Story Points: 3

### [Priority 2] Logging & Observability Improvements
- [x] [T019] Enforce injected logger pattern
  - Description: Remove implicit `slog.Default` usage in handlers. Modify all handler constructors to require a non-nil `Logger` parameter and enforce this requirement at compile time.
  - Acceptance Criteria:
    - AC1: All API handler constructors (auth, memo, card) require a non-nil logger.
    - AC2: No fallback logic to `slog.Default` in any handler code.
    - AC3: Server setup and all tests inject explicit logger instances.
    - AC4: Static analysis confirms absence of `slog.Default` usage in handler packages.
  - Depends On: []
  - Estimated Story Points: 5

- [ ] [T020] Centralize error mapping logic
  - Description: Move duplicated error-to-HTTP status mapping and sanitization logic into shared helpers in `errors.go`. Ensure all handlers use these consistent, centralized error mapping functions.
  - Acceptance Criteria:
    - AC1: `MapErrorToStatusCode` function in `errors.go` handles all error types used in the API.
    - AC2: All handlers use the centralized error mapping functions.
    - AC3: Unit tests verify correct mapping for each error type.
    - AC4: Integration tests confirm consistent error responses after refactoring.
  - Depends On: [T017]
  - Estimated Story Points: 3

- [x] [T021] Correct 4xx log level strategy
  - Description: Modify `RespondWithErrorAndLog` to log client errors (4xx) at DEBUG level by default, elevating only specific operational issues to WARN level.
  - Acceptance Criteria:
    - AC1: Default log level for 4xx errors is DEBUG, not WARN.
    - AC2: Selected operational 4xx scenarios (rate limiting, repeated auth failures) still log at WARN.
    - AC3: Tests verify correct log levels for various error scenarios.
    - AC4: Documentation updated to reflect the new log level strategy.
  - Depends On: []
  - Estimated Story Points: 3

### [Priority 3] Testability & Maintainability
- [x] [T022] Refactor brittle shared tests
  - Description: Improve test quality in the shared package by using struct-based JSON assertions instead of string comparisons, implementing more realistic mocks, and ensuring log checks are robust against format changes.
  - Acceptance Criteria:
    - AC1: JSON response tests unmarshal to structs for field assertions rather than comparing strings.
    - AC2: Log tests check for presence of key attributes rather than exact string matches.
    - AC3: Mock implementations accurately reflect the interfaces they're replacing.
    - AC4: Tests pass regardless of JSON field ordering or minor format changes.
  - Depends On: []
  - Estimated Story Points: 3

### [Priority 4] Code Cleanup & Conventions
- [x] [T023] Implement singleton validator in requests
  - Description: Replace per-call validator creation with a package-level singleton instance to improve performance and reduce allocations.
  - Acceptance Criteria:
    - AC1: Package-level validator instance defined in `internal/api/shared/requests.go`.
    - AC2: `ValidateRequest` uses the singleton validator.
    - AC3: No validator creation per validation call.
    - AC4: Tests verify validator behavior still works correctly.
  - Depends On: []
  - Estimated Story Points: 1

- [ ] [T024] Make handler mutation immutable or test-only
  - Description: Address the non-idiomatic mutable handler pattern (`WithTimeFunc`) by either making it return a new instance or clearly documenting it as test-only with appropriate safeguards.
  - Acceptance Criteria:
    - AC1: Either: `WithTimeFunc` returns a new handler instance, leaving the original unchanged
           Or: Clear documentation marks it as test-only with code comments.
    - AC2: If kept mutable, a test verifies no concurrency issues can occur.
    - AC3: No production code mutates handlers in place.
  - Depends On: []
  - Estimated Story Points: 1
