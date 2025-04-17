# Task Recovery Integration Tests: Remediation Tasks

## Overview

This document lists the concrete tasks to implement the remediation plan for the Task Recovery Integration Tests from the code review. The plan addresses code quality issues, enhances maintainability, improves adherence to architectural principles, and makes the tests more robust.

## Tasks

### T001: Define Constant for Task Type String Literal [x]

- **Description:** Define a constant `TaskTypeMemoGeneration` in the `internal/task` package to replace the magic string `"memo_generation"`.
- **Acceptance Criteria:**
  - A constant `TaskTypeMemoGeneration` is defined in `internal/task/task.go` with the value `"memo_generation"`.
  - The constant is properly exported for use in test files.
  - Code compiles successfully.
- **Implementation Notes:**
  - Define the constant as `const TaskTypeMemoGeneration = "memo_generation"`.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** None

### T002: Replace Magic Strings with Task Type Constant [x]

- **Description:** Replace all occurrences of the string literal `"memo_generation"` in the test files with the newly defined `task.TaskTypeMemoGeneration` constant.
- **Acceptance Criteria:**
  - All instances of `"memo_generation"` in the test files are replaced with `task.TaskTypeMemoGeneration`.
  - Code compiles successfully.
  - All tests pass.
  - A search for `"memo_generation"` in the test files shows no remaining occurrences.
- **Implementation Notes:**
  - Ensure the `task` package is imported where the constant is used.
  - Focus on replacing the literal specifically where it represents the task type.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** T001

### T003: Refactor `getTaskStatusDirectly` Helper [x]

- **Description:** Modify the `getTaskStatusDirectly` helper function to accept `store.DBTX` instead of casting it to `*sql.Tx`.
- **Acceptance Criteria:**
  - Function signature is updated to accept `store.DBTX`.
  - Internal casting to `*sql.Tx` is removed.
  - Database operations are called directly on the `store.DBTX` parameter.
  - Code compiles successfully.
- **Implementation Notes:**
  - Change signature to `func getTaskStatusDirectly(t *testing.T, dbtx store.DBTX, taskID uuid.UUID) (task.TaskStatus, error)`.
  - Remove the `tx, ok := dbtx.(*sql.Tx)` block.
  - Call `dbtx.QueryRowContext(...)` directly.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** None

### T004: Refactor `getTaskIDForMemo` Helper [x]

- **Description:** Modify the `getTaskIDForMemo` helper function to accept `store.DBTX` instead of casting it to `*sql.Tx`.
- **Acceptance Criteria:**
  - Function signature is updated to accept `store.DBTX`.
  - Internal casting to `*sql.Tx` is removed.
  - Database operations are called directly on the `store.DBTX` parameter.
  - Code compiles successfully.
- **Implementation Notes:**
  - Change signature to `func getTaskIDForMemo(t *testing.T, dbtx store.DBTX, memoID uuid.UUID) (uuid.UUID, error)`.
  - Remove the `tx, ok := dbtx.(*sql.Tx)` block.
  - Call `dbtx.QueryRowContext(...)` directly.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** None

### T005: Refactor `getMemoStatusDirectly` Helper [x]

- **Description:** Modify the `getMemoStatusDirectly` helper function to accept `store.DBTX` instead of casting it to `*sql.Tx`.
- **Acceptance Criteria:**
  - Function signature is updated to accept `store.DBTX`.
  - Internal casting to `*sql.Tx` is removed.
  - Database operations are called directly on the `store.DBTX` parameter.
  - Code compiles successfully.
- **Implementation Notes:**
  - Change signature to `func getMemoStatusDirectly(t *testing.T, dbtx store.DBTX, memoID uuid.UUID) (domain.MemoStatus, error)`.
  - Remove the `tx, ok := dbtx.(*sql.Tx)` block.
  - Call `dbtx.QueryRowContext(...)` directly.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** None

### T006: Verify DB Helper Refactoring [x]

- **Description:** Ensure all callsites for the refactored DB helpers correctly pass the `store.DBTX` and run the test suite to confirm no regressions.
- **Acceptance Criteria:**
  - All calls to the refactored helpers pass the `dbtx` variable from `testutils.WithTx`.
  - The full test suite passes without errors related to these helpers.
  - No casting of `store.DBTX` to `*sql.Tx` remains within these helper functions.
- **Implementation Notes:**
  - Review all usages of the helpers within the test files.
  - Run the tests with `go test ./cmd/server/...` and `go test ./...`.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** T003, T004, T005

### T007: Create Default Task Runner Config Helper [x]

- **Description:** Create a helper function in `cmd/server/main_recovery_test.go` that returns a default `task.TaskRunnerConfig` struct.
- **Acceptance Criteria:**
  - A function `getDefaultTestTaskConfig() task.TaskRunnerConfig` exists in the test file.
  - The function returns a `task.TaskRunnerConfig` with the common test values.
  - Code compiles successfully.
- **Implementation Notes:**
  - Identify the common values used across tests for `task.TaskRunnerConfig`.
  - Define a helper function that returns these values as a struct.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** None

### T008: Use Config Helper in Tests [x]

- **Description:** Replace duplicated `task.TaskRunnerConfig` struct literals in the test functions with calls to the `getDefaultTestTaskConfig` helper.
- **Acceptance Criteria:**
  - All duplicated `task.TaskRunnerConfig` struct literals are replaced with calls to the helper.
  - Code compiles successfully.
  - All tests pass.
- **Implementation Notes:**
  - Find all occurrences of `task.TaskRunnerConfig{...}` and replace them with `getDefaultTestTaskConfig()`.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** T007

### T009: Identify Logical Blocks for Setup Function [x]

- **Description:** Analyze the `setupRecoveryTestInstance` function and identify logical blocks that can be extracted into separate helper functions.
- **Acceptance Criteria:**
  - Logical blocks within the function are identified (e.g., auth setup, store setup, service setup).
  - A plan is created for which helpers to create with their signatures and responsibilities.
- **Implementation Notes:**
  - Aim for grouping related component initializations together.
  - Consider dependencies between components when planning the helpers.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** None

### T010: Create Helper Functions for Setup Blocks

- **Description:** Create private helper functions for each logical block identified in T009.
- **Acceptance Criteria:**
  - New private helper functions are created in the test file.
  - Each helper encapsulates one logical component setup block.
  - Helpers take appropriate parameters and return created components.
  - Code compiles successfully.
- **Implementation Notes:**
  - Create functions with clear names like `setupTestAuthComponents`, `setupTestStores`, etc.
  - Pass only the required dependencies to each helper.
- **Estimated Effort:** Medium (2 hours)
- **Depends On:** T009

### T011: Refactor Main Setup Function

- **Description:** Modify the `setupRecoveryTestInstance` function to call the newly created helper functions.
- **Acceptance Criteria:**
  - The function is significantly shorter and more readable.
  - It primarily consists of calls to the helper functions.
  - It correctly returns the same values as before.
  - All tests pass.
- **Implementation Notes:**
  - Ensure variables returned by helpers are correctly passed to subsequent helpers.
  - Maintain the same return signature: `*httptest.Server`, `*task.TaskRunner`, `error`.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** T010

### T012: Add Clarifying Comment for Manual State Updates

- **Description:** Add a comment in `TestTaskRecovery_API` explaining the purpose of manually updating the task and memo status.
- **Acceptance Criteria:**
  - A comment explaining the simulation of a crash during processing exists above the relevant `tx.Exec` calls.
  - The comment clearly explains why the manual updates are necessary.
- **Implementation Notes:**
  - Use a comment like: `// Simulate crash during processing: Manually update task and memo status to 'processing' before starting the recovery instance.`
- **Estimated Effort:** Small (30 minutes)
- **Depends On:** None

### T013: Investigate Unused Code

- **Description:** Search the codebase for usages of the `databaseTask` struct and `NewDatabaseTask` function to determine if they're used.
- **Acceptance Criteria:**
  - A conclusive determination is made whether these items are used anywhere in the codebase.
- **Implementation Notes:**
  - Use search tools like IDE search, `grep`, or `git grep`.
  - Check for both direct and indirect usages.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** None

### T014: Remove Unused Code

- **Description:** If confirmed unused in T013, remove the `databaseTask` struct and `NewDatabaseTask` function from the test file.
- **Acceptance Criteria:**
  - If unused, the definitions are removed from the test file.
  - Code compiles successfully after removal.
  - All tests pass.
- **Implementation Notes:**
  - Only remove if confirmed unused in T013.
  - Run tests after removal to confirm no regressions.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** T013

### T015: Adjust Timeout Values

- **Description:** Increase the timeout values in calls to `waitForRecoveryCondition` to improve test reliability in CI environments.
- **Acceptance Criteria:**
  - Timeout durations in calls to `waitForRecoveryCondition` are moderately increased.
  - All tests pass locally.
- **Implementation Notes:**
  - Locate all calls to `waitForRecoveryCondition` and increase the timeout values (e.g., from 10s to 15s).
  - Avoid excessively large timeouts that might mask real issues.
- **Estimated Effort:** Small (30 minutes)
- **Depends On:** None

### T016: Final Review and Validation

- **Description:** Perform a final review of all changes and validate that they meet the requirements specified in the remediation plan.
- **Acceptance Criteria:**
  - All code changes related to the remediation tasks are reviewed.
  - Static analysis tools pass without new warnings.
  - All tests pass.
  - The changes align with the project's development philosophy.
- **Implementation Notes:**
  - Run `golangci-lint run` on the modified code.
  - Ensure `go test ./...` passes.
  - Review all changes against the original remediation plan.
- **Estimated Effort:** Small (1 hour)
- **Depends On:** T002, T006, T008, T011, T012, T014, T015
