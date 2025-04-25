# Remediation Plan – Sprint 9

## Executive Summary
This sprint targets critical security and architectural flaws identified in code review, with the highest priority given to error log sanitization to eliminate leakage of sensitive data. Issues are sequenced to address blockers and high-severity items first, followed by architectural and consistency improvements; this order ensures security holes are closed immediately and foundational problems do not block downstream enhancements. The plan delivers quick wins early that unlock further architectural cleanup and codebase hardening.

## Strike List
| Seq | CR‑ID | Title | Effort | Owner |
|-----|-------|-------|--------|-------|
| 1   | cr‑03 | Harden error log output | s | backend |
| 2   | cr‑01 | Strengthen trace ID & error handling | s | backend |
| 3   | cr‑05 | Enforce injected logger pattern | m | backend |
| 4   | cr‑08 | Centralize error mapping | m | backend |
| 5   | cr‑06 | Correct 4xx log level strategy | s | backend |
| 6   | cr‑07 | Refactor brittle shared tests | m | backend |
| 7   | cr‑02 | Singleton validator in requests | xs | backend |
| 8   | cr‑04 | Go idiomatic handler mutability | xs | backend |

## Detailed Remedies

### cr‑03 Harden error log output
- **Problem:** Internal error logs in API responses leak sensitive information (stack traces, connection strings, PII).
- **Impact:** Direct security vulnerability—potential exposure of secrets, infrastructure, and internal logic to clients; risk of regulatory non-compliance and breach escalation.
- **Chosen Fix:** All API error responses must explicitly sanitize and redact sensitive data; logs should only contain minimal, non-sensitive context; implement redaction for known patterns (connection strings, secrets, stack traces, IPs).
- **Steps:**
  1. Refactor `RespondWithErrorAndLog` to filter all error messages before logging—never log raw error strings.
  2. Implement a central redaction utility (regex or explicit patterns) for known sensitive data (e.g., "postgres://", "password=", "AKIA", file paths, stack traces).
  3. Update all top-level error log calls in handlers and shared responses to use this sanitizer.
  4. Add regression tests in `errors_leak_test.go` and API integration tests to verify no sensitive data is present in logs or responses.
- **Done‑When:** All error logs and responses verified clean by tests; regression suite covers main attack vectors.

### cr‑01 Strengthen trace ID & error handling
- **Problem:** Trace ID generation is too short (8 bytes), not compliant with tracing standards; error handling is brittle and returns fixed zeroes.
- **Impact:** Breaks traceability across distributed systems, increases risk of collision; undermines observability and incident forensics.
- **Chosen Fix:** Increase trace ID to 16 bytes (32 hex chars) or use UUIDv4; improve error fallback to log and generate a random or nil-safe ID, never a constant.
- **Steps:**
  1. Refactor `generateTraceID` to use 16 random bytes or UUIDv4.
  2. On error, log at ERROR level and fall back to a securely generated value, never a static string.
  3. Update all usages and tests to expect new trace ID length.
  4. Add regression test for trace ID uniqueness and length.
- **Done‑When:** All trace IDs are 32 hex chars/UUID, errors fallback as per spec, and tests reflect new standard.

### cr‑05 Enforce injected logger pattern
- **Problem:** Handlers default to `slog.Default` if no logger is provided, violating explicit dependency injection.
- **Impact:** Logging configuration becomes implicit, leading to inconsistent logs and hidden coupling; breaks explicitness and testability mandates.
- **Chosen Fix:** Require logger in all handler constructors; remove all fallback-to-default logic; update all instantiations.
- **Steps:**
  1. Update all API handler constructors (auth, memo, card) to require a non-nil logger.
  2. Remove any code that sets or falls back to `slog.Default`.
  3. Update all tests and server setup to inject an explicit logger.
  4. Add compile-time checks to enforce non-nil logger.
- **Done‑When:** All handlers receive explicit loggers; codebase has zero direct `slog.Default` usages in handlers.

### cr‑08 Centralize error mapping
- **Problem:** Error-to-status-code and sanitization logic is duplicated in multiple handlers.
- **Impact:** Inconsistent error responses, DRY violation, and maintenance risk as new error types are added.
- **Chosen Fix:** Move all mapping and sanitization to centralized helpers (`errors.go`); update all handlers to use these consistently.
- **Steps:**
  1. Refactor any handler-side error mapping to use the new centralized helpers.
  2. Remove duplicate error-to-status-code and error message logic.
  3. Add tests covering the mapping for all error types.
- **Done‑When:** No handler contains custom error mapping; all error mapping is tested and centralized.

### cr‑06 Correct 4xx log level strategy
- **Problem:** All 4xx errors are logged at WARN, which pollutes logs; only operationally significant 4xxs should be elevated.
- **Impact:** Excessive log noise, increased monitoring costs, and obscured true operational issues.
- **Chosen Fix:** Default to DEBUG log level for most 4xx errors; only log at WARN for rate-limiting or suspicious 4xxs.
- **Steps:**
  1. Refactor `RespondWithErrorAndLog` to set log level for 4xx errors to DEBUG by default.
  2. Add specific cases to elevate to WARN when needed (e.g., repeated auth failures).
  3. Update log aggregation and alerting docs to reflect new strategy.
- **Done‑When:** 4xx errors are logged at correct level; log volume for client errors drops measurably.

### cr‑07 Refactor brittle shared tests
- **Problem:** Shared package tests use string comparisons for JSON and expect fixed log outputs; mocks do not match production validation patterns.
- **Impact:** Tests are brittle, fail on field order changes, and do not reflect true validation logic; maintainability suffers.
- **Chosen Fix:** Parse JSON in tests as objects for assertions; update mocks to match validator interfaces; check log output for key fragments, not full strings.
- **Steps:**
  1. Refactor all shared tests to unmarshal JSON for assertions.
  2. Replace string log checks with key fragment assertions.
  3. Update validation mocks to implement the same interface as production code.
  4. Add regression tests for log output and JSON field order independence.
- **Done‑When:** Shared tests are robust, maintainable, and reflect production code paths.

### cr‑02 Singleton validator in requests
- **Problem:** `ValidateRequest` creates a new validator instance on every call.
- **Impact:** Unnecessary allocation, minor performance and memory waste.
- **Chosen Fix:** Use a package-level singleton validator instance.
- **Steps:**
  1. Move validator instantiation to a package-level `var` and re-use in all calls.
  2. Update any tests that depend on validator state.
- **Done‑When:** Validator is single-instanced, code is idiomatic Go.

### cr‑04 Go idiomatic handler mutability
- **Problem:** Mutable handler pattern (`WithTimeFunc`) is non-idiomatic; risks bugs in concurrent tests.
- **Impact:** Subtle concurrency bugs, non-idiomatic Go, possible test leakage.
- **Chosen Fix:** Make handler immutable (return new instance) or document/test-only usage.
- **Steps:**
  1. Update method to return a copy, not mutate the receiver, or document as test-only.
  2. Add comments to enforce non-production use.
  3. Add test to verify no concurrent mutation occurs.
- **Done‑When:** Handler is immutable or clearly test-only; code is idiomatic Go.

## Standards Alignment
- All remedies are directly aligned with the Scry API development philosophy:
  - **Security**: No sensitive data in logs (cr‑03).
  - **Simplicity**: Error mapping and logging logic is centralized, not duplicated (cr‑08).
  - **Explicitness**: All dependencies are injected, not implicit (cr‑05, cr‑04).
  - **Testability**: Tests are robust, not brittle (cr‑07).
  - **Maintainability**: Singleton validator pattern and error mapping aid future changes (cr‑02, cr‑08).
  - **Observability**: Trace IDs are globally unique and log context is consistent (cr‑01).
  - **Logging Strategy**: 4xx log levels and handler logger injection match philosophy (cr‑06, cr‑05).

## Validation Checklist
- Automated tests green.
- Static analyzers clear.
- Manual pen‑test of error logging and auth flows passes.
- No new lint or audit warnings.
