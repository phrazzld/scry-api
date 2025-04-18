# TODO

## Gemini Generator
- [x] **T108 · refactor · p1: refactor shared logic into gemini_utils.go**
    - **context:** plan.md · 1. potential code duplication in gemini generator implementations
    - **action:**
        1. analyze `createPrompt` and `parseResponse` in `gemini_generator.go` and `gemini_generator_mock.go` to confirm identical logic.
        2. create `internal/platform/gemini/gemini_utils.go` without build tags.
        3. move the shared function implementations into `gemini_utils.go`.
        4. update both implementations to call the shared utility functions.
    - **done-when:**
        1. shared logic resides only in `gemini_utils.go`.
        2. both implementations call the utility functions.
        3. all tests pass with and without the `test_without_external_deps` build tag.
    - **depends-on:** none

- [x] **T109 · test · p2: add error propagation tests for generateCards**
    - **context:** plan.md · 2. test coverage for error propagation
    - **action:**
        1. identify key error types (`ErrContentBlocked`, etc.) that `GenerateCards` should propagate.
        2. add table-driven tests for each error scenario in `gemini_generator_test.go`.
        3. configure the mock client to return specific errors.
        4. verify `GenerateCards` propagates errors correctly using `errors.Is()`.
    - **done-when:**
        1. tests fail if error types are not propagated, and pass when propagation is correct.
        2. code coverage includes error propagation paths.
    - **depends-on:** none

- [x] **T110 · refactor · p2: define and use default retry constants**
    - **context:** plan.md · 3. default retry constants
    - **action:**
        1. define `defaultMaxRetries = 3` and `defaultBaseDelaySeconds = 2` constants at the top of `gemini_generator.go`.
        2. replace magic numbers in retry logic with these constants.
        3. add explanatory comments describing why these defaults exist.
    - **done-when:**
        1. all magic numbers in retry logic are replaced by named constants.
        2. comments explain the purpose of these default values.
        3. tests and linter pass.
    - **depends-on:** none

- [x] **T111 · chore · p2: document purpose of var _ = generation.ErrGenerationFailed**
    - **context:** plan.md · 4. unclear `var _ = ...` usage
    - **action:**
        1. add a comment above `var _ = generation.ErrGenerationFailed` in `gemini_generator.go` explaining its purpose (ensuring package import for error types).
    - **done-when:**
        1. comment clearly explains the purpose of this line.
    - **depends-on:** none

## CI
- [ ] **T112 · chore · p3: add optional CI job for real Gemini API tests**
    - **context:** plan.md · 5. (optional) CI test path for real API
    - **action:**
        1. add a new job in `.github/workflows/ci.yml` (e.g., `test-integration-gemini`).
        2. configure the job to run manually (`workflow_dispatch`) or on specific events (scheduled or branch-specific).
        3. set up secure handling of the Gemini API key using GitHub Secrets.
        4. run Gemini tests without the `test_without_external_deps` build tag.
        5. document the job's purpose, cost implications, and potential flakiness.
    - **done-when:**
        1. CI job exists, is documented, runs successfully, and does not leak secrets.
    - **depends-on:** none

### Clarifications & Assumptions
- [ ] **issue:** should `gemini_utils.go` include only pure logic, or is some state-sharing acceptable?
    - **context:** 1. Potential Code Duplication in Gemini Generator Implementations
    - **blocking?:** no

- [ ] **issue:** should the new CI job run on all PRs, or only on demand/nightly/main branch?
    - **context:** 5. (Optional) CI Test Path for Real API
    - **blocking?:** no
