# Remediation Plan – Sprint <n>

## Executive Summary

This plan addresses critical issues identified in the recent code review of the feature/card-review-api branch. We prioritize sealing security vulnerabilities in error handling and JWT testing, followed by refactoring brittle error mapping patterns. By tackling issues in order of severity and dependency, we will improve security, maintainability, and testability while aligning with our development philosophy principles.

## Strike List

| Seq | CR‑ID | Title                                | Effort | Owner?  |
|-----|-------|--------------------------------------|--------|---------|
| 1   | cr‑02 | Seal Error Wrapping Details Leak     | m      | backend |
| 2   | cr‑01 | Use Error Types for Mapping          | m      | backend |
| 3   | cr‑03 | Fix Dangerous Default JWT Stubbing   | s      | backend |
| 4   | cr‑08 | Fix Error Logging Level              | xs     | backend |
| 5   | cr‑04 | Simplify Card Review API Test Harness| m      | backend |
| 6   | cr‑05 | Consolidate Test Helper Abstractions | s      | backend |
| 7   | cr‑06 | Replace Stub Test Files              | m      | backend |

## Detailed Remedies

### cr‑02 Seal Error Wrapping Details Leak
- **Problem:** Error wrapping pattern `fmt.Errorf("%w: ...", errType, err)` can leak internal error messages to clients.
- **Impact:** Security vulnerability exposing sensitive information like database errors or file paths; violates Logging strategy principles.
- **Chosen Fix:** Implement structured logging for internal errors and return only standardized, safe messages to clients.
- **Steps:**
  1. Audit service and API handler error paths (`internal/api/errors.go`, `internal/store/errors.go`, service implementations).
  2. Modify `RespondWithErrorAndLog` in `internal/api/shared/responses.go` to log the full internal error details but only send safe messages to clients.
  3. Ensure error response helpers never include raw error strings in the client-facing message.
  4. Add tests that verify no internal errors leak to API responses.
- **Done‑When:** No internal error details appear in API responses, full error details are properly logged, and all tests pass.

### cr‑01 Use Error Types for Mapping
- **Problem:** Error mapping relies on matching error message strings rather than error types.
- **Impact:** Brittle code that breaks with minor error message changes; violates error handling principles.
- **Chosen Fix:** Define and use sentinel errors with `errors.Is`/`errors.As` for error mapping.
- **Steps:**
  1. Define exported sentinel errors in domain/store packages (e.g., `var ErrNotFound = errors.New("not found")`).
  2. Refactor `MapErrorToStatusCode` and `GetSafeErrorMessage` in `internal/api/errors.go` to use `errors.Is`/`errors.As`.
  3. Update service and store implementations to use these sentinel errors.
  4. Remove all string-based error checks (`err.Error() ==`, `strings.Contains(err.Error(), ...)`).
  5. Add tests to verify error mapping based on types, not strings.
- **Done‑When:** All error handling uses `errors.Is`/`errors.As` with sentinel errors, string comparisons are removed, and tests pass.

### cr‑03 Fix Dangerous Default JWT Stubbing
- **Problem:** Default `ValidateTokenFn` in test helpers grants blanket access for "Bearer test-token".
- **Impact:** Allows tests to pass without proper authentication context; potentially masks security issues.
- **Chosen Fix:** Use real JWT generation/validation with a test secret in test helpers.
- **Steps:**
  1. Remove the default `ValidateTokenFn` in `internal/testutils/api_helpers.go`.
  2. Create a proper test JWT service that uses a dedicated test secret key.
  3. Modify test setup to generate real, signed JWTs with specific claims.
  4. Update API tests to explicitly provide required claims for authenticated requests.
- **Done‑When:** Default stub is removed, tests use proper JWT validation, and security vulnerabilities are mitigated.

### cr‑08 Fix Error Logging Level
- **Problem:** Shared error response helpers log all errors at DEBUG level regardless of severity.
- **Impact:** Violates logging strategy principle; hinders effective monitoring and debugging.
- **Chosen Fix:** Adjust logging level based on HTTP status code.
- **Steps:**
  1. Modify `RespondWithErrorAndLog` in `internal/api/shared/responses.go`.
  2. Log 4xx status codes at WARN level and 5xx status codes at ERROR level.
  3. Ensure contextual information (trace ID, endpoint, etc.) is included in logs.
- **Done‑When:** Error responses are logged at appropriate severity levels while preserving all context.

### cr‑04 Simplify Card Review API Test Harness
- **Problem:** The test harness introduces excessive layers of mock configuration and abstractions.
- **Impact:** Increases complexity and maintenance burden; violates Simplicity first principle.
- **Chosen Fix:** Remove unnecessary abstraction and collapse helpers to a minimal surface.
- **Steps:**
  1. Review and simplify `CardReviewServerOptions` in `internal/testutils/api_helpers.go`.
  2. Remove or consolidate unnecessary configuration options.
  3. Make test setup more explicit and straightforward.
- **Done‑When:** Test harness is simplified while maintaining effective test coverage.

### cr‑05 Consolidate Test Helper Abstractions
- **Problem:** Multiple overlapping test helpers exist, increasing maintenance burden.
- **Impact:** Confusion for developers; violates DRY principle; increases technical debt.
- **Chosen Fix:** Consolidate helpers into a single, consistent pattern per domain.
- **Steps:**
  1. Audit test helpers in `internal/testutils/api_helpers.go` and `internal/testutils/card_helpers.go`.
  2. Eliminate duplicate functionality; create a single pattern for common operations.
  3. Document the consolidated helpers clearly.
- **Done‑When:** Test helpers are consolidated with no duplication, and documentation is clear.

### cr‑06 Replace Stub Test Files
- **Problem:** Stub test files exist purely to suppress test failures or coverage holes.
- **Impact:** Masks actual test coverage gaps; violates testing strategy principles.
- **Chosen Fix:** Remove stubs and implement proper tests or explicit TODOs.
- **Steps:**
  1. Delete stub test files (`internal/api/auth_handler_test.go`, `internal/api/memo_handler_test.go`).
  2. Either implement proper tests for these components or document the gaps as TODOs.
  3. Ensure test coverage metrics accurately reflect the actual coverage.
- **Done‑When:** Stub files are replaced with real tests or explicit TODOs, and coverage metrics are accurate.

## Standards Alignment

This remediation plan directly addresses violations of our core development philosophy:

- **Simplicity First**: Removing string-based error checking (cr-01), simplifying test infrastructure (cr-04, cr-05), and consolidating helpers (cr-05) all improve simplicity.
- **Modularity**: Using proper error types (cr-01) improves module boundaries and interfaces between components.
- **Design for Testability**: Fixing JWT stubs (cr-03), simplifying test harness (cr-04), and replacing stub tests (cr-06) directly enhance testability.
- **Security**: Sealing error detail leaks (cr-02) and fixing JWT test stubbing (cr-03) address significant security concerns.
- **Logging Strategy**: Fixing incorrect log levels (cr-08) and improving error detail handling (cr-02) align with proper logging practices.

## Validation Checklist

- Automated tests pass (`go test ./...`).
- Static analysis (`golangci-lint run`) shows no warnings.
- Pre-commit hooks pass (`pre-commit run --all-files`).
- Manual review confirms no internal error details leak to API responses.
- Test coverage metrics are accurate with no stub files.
- Test helpers follow consistent patterns.
- No security vulnerabilities in token validation.
