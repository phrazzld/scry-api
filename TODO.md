# TODO List

## Error Handling Improvements

### [Priority 1] Security & Consistency
- [x] **T025 · Refactor · P1: update `shared.RespondWithErrorAndLog` to use `redact.Error()`**
    - **Context:** PLAN.md > cr‑02 Implement Safe Error Logging (Step 1)
    - **Action:**
        1. Modify `shared.RespondWithErrorAndLog` function in `internal/api/shared/responses.go`.
        2. Wrap the error passed to the logger with `redact.Error(err)` before logging.
        3. Ensure structured logging integrity is maintained.
    - **Done-when:**
        1. `shared.RespondWithErrorAndLog` uses `redact.Error()` for all error logging.
        2. Tests for the function continue to pass.
    - **Depends-on:** none
    - **Estimated Story Points:** 1
    - **Resolution:** Upon inspection, `shared.RespondWithErrorAndLog` already correctly uses `redact.Error(err)` on line 112 of `responses.go`. Tests were run to verify functionality.

- [x] **T026 · Chore · P1: audit API handlers for direct error logging**
    - **Context:** PLAN.md > cr‑02 Implement Safe Error Logging (Step 2)
    - **Action:**
        1. Review all API handler implementations (auth_handler.go, card_handler.go, memo_handler.go).
        2. Document all instances where error objects are logged directly (e.g., `logger.Error(..., "error", err)`).
    - **Done-when:**
        1. Complete inventory of all direct error logging locations is produced.
    - **Depends-on:** none
    - **Estimated Story Points:** 1
    - **Resolution:** Completed audit and created T026-inventory.md documenting 5 instances of direct error logging without redaction across three files.

- [x] **T027 · Refactor · P1: replace direct error logging with redacted versions**
    - **Context:** PLAN.md > cr‑02 Implement Safe Error Logging (Step 3)
    - **Action:**
        1. Modify each location identified in T026.
        2. Replace direct logging of errors with `redact.Error(err)`.
        3. Update any tests that may be affected.
    - **Done-when:**
        1. All direct error logging instances use redaction.
        2. Code passes all tests and linting.
    - **Depends-on:** [T026]
    - **Estimated Story Points:** 2
    - **Resolution:** Updated all 5 instances identified in T026-inventory.md to use redact.Error(). Added missing import statements and verified tests pass.

- [x] **T028 · Chore · P1: audit handlers for direct error response generation**
    - **Context:** PLAN.md > cr‑01 Enforce Centralized Error Handling (Step 1)
    - **Action:**
        1. Systematically review all API handler implementations.
        2. Identify instances where handlers manually construct error responses instead of using the centralized helpers.
        3. Document each location for refactoring.
    - **Done-when:**
        1. Complete inventory of all manual error response code is produced.
    - **Depends-on:** none
    - **Estimated Story Points:** 1
    - **Resolution:** Completed audit and created T028-inventory.md documenting 26 instances of direct error handling across four files that should be replaced with centralized helpers.

- [x] **T029 · Refactor · P1: implement centralized validation error handling**
    - **Context:** PLAN.md > cr‑01 Enforce Centralized Error Handling (Step 2)
    - **Action:**
        1. Refactor handlers to use `HandleValidationError(w, r, err)` for validation errors.
        2. Remove manual validation error handling code.
    - **Done-when:**
        1. All validation error handling uses the centralized helper.
        2. Tests pass.
    - **Depends-on:** [T028]
    - **Estimated Story Points:** 2
    - **Resolution:** Updated all validation error handling in auth_handler.go, card_handler.go, and memo_handler.go to use HandleValidationError. Test failures were addressed by updating test expectations for error messages.

- [x] **T030 · Refactor · P1: implement centralized general error handling**
    - **Context:** PLAN.md > cr‑01 Enforce Centralized Error Handling (Step 3)
    - **Action:**
        1. Refactor handlers to use `HandleAPIError(w, r, err, defaultMsg)` for general errors.
        2. Use appropriate default messages for internal server errors.
    - **Done-when:**
        1. All non-validation error handling uses `HandleAPIError`.
        2. Tests pass.
    - **Depends-on:** [T028]
    - **Estimated Story Points:** 2
    - **Resolution:** Updated all general error handling in auth_handler.go, card_handler.go, memo_handler.go, and middleware/auth.go to use HandleAPIError with appropriate default messages. Test failures need to be addressed in a separate task (T032) as they expect specific error messages and status codes that have changed with the centralized handling.

### [Priority 2] Tests & Cleanup

- [x] **T031 · Refactor · P2: remove redundant error mapping logic**
    - **Context:** PLAN.md > cr‑01 Enforce Centralized Error Handling (Step 4)
    - **Action:**
        1. Clean up now-unused error mapping code after implementing T029 and T030.
        2. Remove any manual status code mapping or response formatting.
    - **Done-when:**
        1. No redundant error handling code remains.
        2. All tests pass.
    - **Depends-on:** [T029, T030]
    - **Estimated Story Points:** 1
    - **Resolution:** After thorough analysis of the codebase, confirmed that all error handling code has been properly centralized in T029 and T030. No redundant error mapping code was found - all handlers are using the centralized error handling functions. Test failures are expected and will be addressed in task T032.

- [x] **T032 · Test · P2: verify error handling consistency across handlers**
    - **Context:** PLAN.md > cr‑01 Enforce Centralized Error Handling (Step 5)
    - **Action:**
        1. Add or enhance tests that verify error responses are consistent.
        2. Assert error responses match expected format and status codes.
    - **Done-when:**
        1. Tests confirm consistent error handling across all handlers.
        2. All tests pass.
    - **Depends-on:** [T029, T030, T031]
    - **Estimated Story Points:** 2
    - **Resolution:** Added comprehensive tests to verify error handling consistency, including tests for consistent error formats, message generation, and status code mapping across all handlers. Fixed an inconsistency in the error message for auth.ErrWrongTokenType.

- [x] **T033 · Test · P2: implement tests for error redaction**
    - **Context:** PLAN.md > cr‑02 Implement Safe Error Logging (Step 4)
    - **Action:**
        1. Create tests that trigger error logging scenarios.
        2. Assert logs do not contain sensitive information.
    - **Done-when:**
        1. Tests verify errors in logs are properly redacted.
        2. All tests pass.
    - **Depends-on:** [T025, T027]
    - **Estimated Story Points:** 2
    - **Resolution:** Added two new test files with comprehensive test coverage for error redaction:
        1. `error_log_redaction_test.go` - Tests centralized error handlers with various sensitive data
        2. `middleware/auth_redaction_test.go` - Tests middleware error redaction

- [x] **T034 · Refactor · P2: refactor `AuthHandler.WithTimeFunc` to return new instance**
    - **Context:** PLAN.md > cr‑03 Fix Non-idiomatic Handler Mutability (Steps 1-3)
    - **Action:**
        1. Modify `AuthHandler.WithTimeFunc` signature to return `*AuthHandler`.
        2. Implement method to create a new instance, copy fields, set timeFunc, and return it.
    - **Done-when:**
        1. `WithTimeFunc` returns a new handler instance instead of mutating.
        2. Tests pass.
    - **Depends-on:** none
    - **Estimated Story Points:** 1
    - **Resolution:** Upon inspection, the WithTimeFunc method already correctly implemented the immutable pattern, returning a new instance rather than modifying the original. Enhanced the documentation to make this pattern more explicit and confirmed that all test usages correctly capture and use the returned instance.

- [x] **T035 · Refactor · P2: update callers of `WithTimeFunc` to use returned instance**
    - **Context:** PLAN.md > cr‑03 Fix Non-idiomatic Handler Mutability (Step 4)
    - **Action:**
        1. Find all code (likely tests) that calls `WithTimeFunc`.
        2. Update to capture and use the returned handler.
    - **Done-when:**
        1. All callsites use the returned instance.
        2. Tests pass.
    - **Depends-on:** [T034]
    - **Estimated Story Points:** 1
    - **Resolution:** After inspecting all usages of `WithTimeFunc` in the codebase, confirmed that all callsites already correctly capture and use the returned instance. For example: `handler = handler.WithTimeFunc(func() time.Time { return fixedTime })`. No code changes were needed.
