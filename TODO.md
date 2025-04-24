# Todo

## Error Handling & Logging
- [x] **T001 · Security · P0: modify error response helper to prevent internal detail leaks**
    - **Context:** PLAN.md / cr‑02 Seal Error Wrapping Details Leak / Step 2 & 3
    - **Action:**
        1. Modify `RespondWithErrorAndLog` in `internal/api/shared/responses.go`.
        2. Log the full internal error details using structured logging.
        3. Ensure only standardized, safe error messages (without raw error strings) are sent in the client response.
    - **Done‑when:**
        1. `RespondWithErrorAndLog` logs full internal error details via structured logging.
        2. API responses contain only safe, generic error messages, not raw internal error strings.
        3. Existing relevant tests pass.
    - **Depends‑on:** none

- [x] **T002 · Test · P1: add tests verifying no internal error details leak via api**
    - **Context:** PLAN.md / cr‑02 Seal Error Wrapping Details Leak / Step 4
    - **Action:**
        1. Create specific test cases that generate internal errors (e.g., mock database errors).
        2. Assert that the API responses for these errors do *not* contain the internal error strings.
    - **Done‑when:**
        1. New tests exist demonstrating that internal error messages are not exposed in API responses.
        2. All tests pass.
    - **Depends‑on:** [T001]

- [x] **T003 · Refactor · P1: define exported sentinel errors for domain/store packages**
    - **Context:** PLAN.md / cr‑01 Use Error Types for Mapping / Step 1
    - **Action:**
        1. Identify common error conditions (e.g., not found, conflict, validation) in domain and store packages.
        2. Define exported sentinel errors (e.g., `var ErrNotFound = errors.New("not found")`) for these conditions.
    - **Done‑when:**
        1. Sentinel errors are defined and exported in relevant domain/store packages.
    - **Depends‑on:** none

- [x] **T004 · Refactor · P1: refactor error mapping functions to use errors.is/errors.as**
    - **Context:** PLAN.md / cr‑01 Use Error Types for Mapping / Step 2
    - **Action:**
        1. Modify `MapErrorToStatusCode` in `internal/api/errors.go` to use `errors.Is` and `errors.As` for checking against sentinel/custom error types.
        2. Modify `GetSafeErrorMessage` in `internal/api/errors.go` similarly, removing reliance on error strings.
    - **Done‑when:**
        1. `MapErrorToStatusCode` and `GetSafeErrorMessage` use type checking (`errors.Is`/`errors.As`) instead of string matching.
        2. Functions correctly map defined sentinel/custom errors.
    - **Depends‑on:** [T003]

- [x] **T005 · Refactor · P1: update service/store implementations to return defined sentinel errors**
    - **Context:** PLAN.md / cr‑01 Use Error Types for Mapping / Step 3 & 4
    - **Action:**
        1. Audit service and store implementations for places where errors are returned.
        2. Replace generic errors or string-matched errors with the appropriate defined sentinel errors.
        3. Remove all error checks based on `err.Error() ==` or `strings.Contains(err.Error(), ...)`.
    - **Done‑when:**
        1. Service/store code returns sentinel errors where appropriate.
        2. All string-based error comparisons are removed.
        3. Existing relevant tests pass.
    - **Depends‑on:** [T004]

- [x] **T006 · Test · P1: add tests verifying error mapping based on error types**
    - **Context:** PLAN.md / cr‑01 Use Error Types for Mapping / Step 5
    - **Action:**
        1. Write new tests for `MapErrorToStatusCode` and `GetSafeErrorMessage`.
        2. Ensure tests cover mapping based on sentinel errors and potentially custom error types, verifying the correct status code and safe message are returned.
    - **Done‑when:**
        1. Tests exist confirming error mapping works correctly based on error types/identity.
        2. All tests pass.
    - **Depends‑on:** [T005]

- [x] **T007 · Refactor · P1: adjust error logging levels based on http status code**
    - **Context:** PLAN.md / cr‑08 Fix Error Logging Level / Step 1, 2 & 3
    - **Action:**
        1. Modify `RespondWithErrorAndLog` in `internal/api/shared/responses.go`.
        2. Implement logic to log errors associated with 4xx status codes at WARN level.
        3. Implement logic to log errors associated with 5xx status codes at ERROR level, ensuring necessary context (trace ID, etc.) is included.
    - **Done‑when:**
        1. Errors resulting in 4xx responses are logged at WARN level.
        2. Errors resulting in 5xx responses are logged at ERROR level.
        3. Logged errors include required contextual information.
    - **Depends‑on:** [T001]

## Security & Testing
- [x] **T008 · Security · P0: implement test jwt service for generating/validating real tokens**
    - **Context:** PLAN.md / cr‑03 Fix Dangerous Default JWT Stubbing / Step 2
    - **Action:**
        1. Create a JWT utility/service within `internal/testutils` using a dedicated, non-production test secret key.
        2. Implement functions to generate valid, signed JWTs with specified claims for testing.
        3. Implement a validation function suitable for use in test servers.
    - **Done‑when:**
        1. Test utility exists to create signed JWTs with arbitrary claims using a test secret.
        2. Test utility exists to validate tokens signed with the test secret.
    - **Depends‑on:** none

- [✓] **T009 · Security · P0: remove default jwt stub and update api tests to use real tokens**
    - **Context:** PLAN.md / cr‑03 Fix Dangerous Default JWT Stubbing / Step 1, 3 & 4
    - **Action:**
        1. Remove the default `ValidateTokenFn` stub from `internal/testutils/api_helpers.go`.
        2. Modify test setup to generate real, signed JWTs with the test JWT service.
        3. Update API tests requiring authentication to explicitly provide required claims for authenticated requests.
    - **Done‑when:**
        1. Default JWT stub is removed.
        2. API tests requiring auth generate and use real, signed JWTs via the test JWT service.
        3. All relevant API tests pass with proper token validation.
    - **Depends‑on:** [T008]

## Test Infrastructure
- [✓] **T010 · Refactor · P2: simplify card review api test harness options**
    - **Context:** PLAN.md / cr‑04 Simplify Card Review API Test Harness / Step 1 & 2
    - **Action:**
        1. Review and simplify `CardReviewServerOptions` in `internal/testutils/card_api_helpers.go`.
        2. Identify and remove unnecessary configuration options.
        3. Refactor impacted tests as needed.
    - **Done‑when:**
        1. `CardReviewServerOptions` struct and its usage are simplified.
        2. No unused or redundant options remain.
        3. All tests that use the options pass.
    - **Depends‑on:** none

- [✓] **T011 · Refactor · P2: make test setup more explicit and straightforward**
    - **Context:** PLAN.md / cr‑04 Simplify Card Review API Test Harness / Step 3
    - **Action:**
        1. Update test setup code to use simplified options and be more explicit.
        2. Remove indirect configuration layers where appropriate.
    - **Done‑when:**
        1. Test setup is more explicit and less reliant on complex option patterns.
        2. All tests pass with the updated setup.
    - **Depends‑on:** [T010]

- [✓] **T012 · Refactor · P2: consolidate overlapping test helpers**
    - **Context:** PLAN.md / cr‑05 Consolidate Test Helper Abstractions / Step 1 & 2
    - **Action:**
        1. Audit helpers in `internal/testutils/api_helpers.go` and `internal/testutils/card_helpers.go`.
        2. Identify and merge helpers with duplicate or overlapping functionality.
        3. Establish a consistent pattern for common operations.
    - **Done‑when:**
        1. Redundant test helpers are eliminated or merged.
        2. A clear, consistent pattern exists for common test operations.
        3. Tests using the consolidated helpers pass.
    - **Depends‑on:** none

- [✓] **T013 · Chore · P2: document consolidated test helpers**
    - **Context:** PLAN.md / cr‑05 Consolidate Test Helper Abstractions / Step 3
    - **Action:**
        1. Add GoDoc comments explaining the purpose and usage of the primary consolidated test helpers.
    - **Done‑when:**
        1. Consolidated test helpers have clear GoDoc documentation.
    - **Depends‑on:** [T012]

- [✓] **T014 · Test · P2: replace auth_handler_test.go stub with real tests**
    - **Context:** PLAN.md / cr‑06 Replace Stub Test Files / Step 1 & 2
    - **Action:**
        1. Delete `internal/api/auth_handler_test.go` stub file.
        2. Implement meaningful unit or integration tests for the auth handler functionality.
    - **Done‑when:**
        1. Stub file is replaced with real tests.
        2. Tests cover key auth handler functionality.
        3. All tests pass.
    - **Depends‑on:** none

- [✓] **T015 · Test · P2: replace memo_handler_test.go stub with real tests**
    - **Context:** PLAN.md / cr‑06 Replace Stub Test Files / Step 1 & 2
    - **Action:**
        1. Delete `internal/api/memo_handler_test.go` stub file.
        2. Implement meaningful unit or integration tests for the memo handler functionality.
    - **Done‑when:**
        1. Stub file is replaced with real tests.
        2. Tests cover key memo handler functionality.
        3. All tests pass.
    - **Depends‑on:** none

- [ ] **T016 · Chore · P2: ensure accurate test coverage metrics**
    - **Context:** PLAN.md / cr‑06 Replace Stub Test Files / Step 3
    - **Action:**
        1. After implementing real tests for auth and memo handlers, run coverage report.
        2. Verify coverage accurately reflects test status with no artificial inflation.
    - **Done‑when:**
        1. Coverage metrics accurately reflect the actual test coverage.
    - **Depends‑on:** [T014, T015]

### Clarifications & Assumptions
- [ ] **Issue:** Determine whether to implement full tests or add TODOs for auth/memo handlers
    - **Context:** cr‑06 Replace Stub Test Files / Step 2
    - **Blocking?:** yes
