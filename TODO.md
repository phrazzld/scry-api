# Todo

## CI Fixes (cr-06)
- [x] **T029 · Fix · P1: resolve build tag compatibility issues in testutils package**
    - **Context:** CI is using test_without_external_deps build tag, but some internal tests require different tags
    - **Action:**
        1. Standardize build tags across test helper files
        2. Update pre-commit hooks to use consistent build tags
        3. Ensure test files are compatible with both local development and CI environments
    - **Done‑when:**
        1. All tests pass in both local environment and CI pipeline
        2. Pre-commit hooks run successfully with the same build tags as CI
    - **Depends‑on:** none

- [x] **T020 · Fix · P0: resolve test failures in CI**
    - **Context:** CI is failing on the latest PR #26 due to test failures
    - **Action:**
        1. Investigate the root cause of test failures in the CI pipeline
        2. Implement fixes to ensure tests pass in the CI environment
        3. Update test configuration if necessary
    - **Done‑when:**
        1. CI pipeline passes all tests successfully
        2. Changes are merged to the feature branch
    - **Depends‑on:** T023, T024, T025, T026, T027

### Test Helper Refactoring Tasks
- [x] **T023 · Refactor · P1: refactor testutils API helpers to remove compatibility stubs**
    - **Context:** CI failures are partly due to stub implementations in integration test helpers
    - **Action:**
        1. Remove stub implementations in `integration_helpers.go` and `integration_test_helpers.go`
        2. Implement actual functionality for all helper methods
        3. Update affected tests to use correct function calls
    - **Done‑when:**
        1. All stub implementations are replaced with actual implementations
        2. Tests using these helpers pass locally
    - **Depends‑on:** none

- [x] **T024 · Refactor · P1: remove deprecated CardWithStatsOptions struct and old test helpers**
    - **Context:** Deprecated test helpers are causing inconsistencies in tests
    - **Action:**
        1. Remove deprecated `CardWithStatsOptions` struct
        2. Refactor tests to use the current pattern for creating test cards
        3. Update documentation to reflect the new approach
    - **Done‑when:**
        1. Deprecated struct and associated helpers are removed
        2. All tests consistently use the same pattern
    - **Depends‑on:** none

- [x] **T025 · Refactor · P1: audit and deduplicate JWT service and auth helpers**
    - **Context:** Multiple JWT service and auth helpers create confusion and inconsistency
    - **Action:**
        1. Identify all JWT service and auth helper implementations
        2. Consolidate them into a single, reusable implementation
        3. Update imports and usage across the codebase
    - **Done‑when:**
        1. Single, consistent JWT and auth helper implementation exists
        2. All tests using these helpers pass locally
    - **Depends‑on:** none

- [x] **T026 · Refactor · P1: standardize test server setup and request helpers**
    - **Context:** Test server setup and request helpers lack standardization
    - **Action:**
        1. Create consistent interfaces for test server setup
        2. Standardize request helper functions across all test files
        3. Ensure proper cleanup of resources in all test setup/teardown
    - **Done‑when:**
        1. Test server setup is consistent across all tests
        2. Request helpers follow a standardized pattern
    - **Depends‑on:** T023, T025

- [x] **T027 · Refactor · P1: remove all references to compatibility.go and integration_helpers.go stubs**
    - **Context:** References to compatibility layer stubs need to be removed
    - **Action:**
        1. Search for and replace all references to compatibility layer stubs
        2. Update imports to use the standardized helpers
        3. Ensure tests are using the actual implementations
    - **Done‑when:**
        1. No references to compatibility stubs remain
        2. All tests pass with the updated helpers
    - **Depends‑on:** T023, T024, T026

- [x] **T028 · Chore · P1: final validation and task completion**
    - **Context:** Verify that all refactoring tasks resolve the CI failures
    - **Action:**
        1. Run the entire test suite to verify all tests pass
        2. Check CI pipeline to confirm the issue is resolved
        3. Document the changes and update relevant documentation
    - **Done‑when:**
        1. CI pipeline passes all tests
        2. Documentation is updated to reflect changes
        3. T020 is marked complete
    - **Depends‑on:** T023, T024, T025, T026, T027
