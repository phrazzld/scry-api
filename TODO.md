# TODO

## Pre-commit Hook Enhancement

- [x] **Analyze Current Configuration:** Examine the `.pre-commit-config.yaml` file to understand its current state and identify gaps.
  - **Action:** Review the existing configuration file, noting the current hooks, their versions, and any missing useful hooks that could be added.
  - **Depends On:** None
  - **AC Ref:** Implicit from PLAN.md Section 2.1

- [x] **Implement Custom File Length Warning Hook:** Add a warning hook for long files that doesn't block commits.
  - **Action:** Add the `local` hook definition for `warn-long-files` to `.pre-commit-config.yaml` using the provided Python script. Configure `MAX_LINES=500`, `types: [text]`, `pass_filenames: true`, `verbose: true`. Ensure the script always exits with code 0.
  - **Depends On:** Analyze Current Configuration
  - **AC Ref:** Implicit from PLAN.md Section 2.2, 3.0

- [ ] **Update Existing Pre-commit Hooks:** Keep hooks up-to-date with latest versions.
  - **Action:** Run `pre-commit autoupdate` in the repository root. Review the changes made to `.pre-commit-config.yaml` for compatibility and correctness.
  - **Depends On:** Analyze Current Configuration
  - **AC Ref:** Implicit from PLAN.md Section 2.3

- [ ] **Add Standard Pre-commit Hooks:** Enhance configuration with additional useful standard hooks.
  - **Action:** Add the following hooks from the `pre-commit/pre-commit-hooks` repository (using a recent stable `rev`, e.g., v4.6.0): `trailing-whitespace`, `end-of-file-fixer`, `check-yaml`, `check-json`, `check-added-large-files`, `check-merge-conflict`.
  - **Depends On:** Update Existing Pre-commit Hooks
  - **AC Ref:** Implicit from PLAN.md Section 2.4

- [ ] **Organize and Document Configuration File:** Improve readability and maintainability of the configuration.
  - **Action:** Restructure the `.pre-commit-config.yaml` file. Group hooks logically (e.g., Formatting, Linting, Validation, Custom). Add comments above each hook or group explaining its purpose and any non-obvious configuration. Ensure consistent YAML formatting.
  - **Depends On:** Implement Custom File Length Warning Hook, Add Standard Pre-commit Hooks
  - **AC Ref:** Implicit from PLAN.md Section 2.5

- [ ] **Update README Documentation:** Inform developers about pre-commit features and usage.
  - **Action:** Add or update a "Pre-commit Hooks" section in the `README.md` file listing key hooks (including the new warning hook), their purposes, and installation instructions (`pre-commit install`).
  - **Depends On:** Organize and Document Configuration File
  - **AC Ref:** Implicit from PLAN.md Section 2.6

## Testing

- [ ] **Test Custom Hook Functionality:** Verify the file length warning works correctly.
  - **Action:** Create temporary test files: one short text file (<500 lines), one long text file (>500 lines), one binary file. Run `pre-commit run warn-long-files --files <test-files>`. Verify the long file generates a warning to stderr, the short file and binary file do not, and the hook exits with code 0 in all cases.
  - **Depends On:** Implement Custom File Length Warning Hook
  - **AC Ref:** Implicit from PLAN.md Section 3

- [ ] **Test Edge Cases:** Ensure the hook handles various special cases properly.
  - **Action:** Test with a very large text file (e.g., 10MB) to verify performance, and with files using unusual encodings (UTF-16) to verify encoding handling.
  - **Depends On:** Test Custom Hook Functionality
  - **AC Ref:** Implicit from PLAN.md Section 3 (Edge Cases)

- [ ] **Test Overall Pre-commit Configuration:** Verify all hooks work together correctly.
  - **Action:** Run `pre-commit run --all-files`. Verify that all hooks execute without unexpected errors or failures (the custom warning hook should issue warnings but pass). Address any hook failures unrelated to the intentional warning.
  - **Depends On:** Organize and Document Configuration File, Test Custom Hook Functionality
  - **AC Ref:** Implicit from PLAN.md Section 3

## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS

- [ ] **Issue/Assumption:** Assumed `MAX_LINES = 500` is an acceptable threshold for the file length warning.
  - **Context:** PLAN.md Section 2.2 (Python script configuration).

- [ ] **Issue/Assumption:** Assumed Python 3 is available in the execution environment for the custom `local` hook.
  - **Context:** PLAN.md Section 2.2 (hook definition `language: python`).

- [ ] **Issue/Assumption:** Assumed the existing pre-commit hooks should be kept with their functionality (only updated versions) rather than replaced entirely.
  - **Context:** PLAN.md Section 2.3 mentions updating hooks but doesn't specify if any should be removed.
