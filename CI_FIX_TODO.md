# CI Fix Tasks

## Implementation Plan

- [x] **T101:** Create test helpers for the mock implementation
    - **Action:** Create a new file `internal/platform/gemini/gemini_test_helpers.go` with build tags for test environment that provides helper functions for testing
    - **Depends On:** None
    - **AC Ref:** None

- [x] **T102:** Update CI workflow test job to use build tags
    - **Action:** Modify `.github/workflows/ci.yml` to add the `-tags=test_without_external_deps` flag to the test command:
      ```yaml
      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out -tags=test_without_external_deps ./...
      ```
    - **Depends On:** None
    - **AC Ref:** None

- [ ] **T103:** Update CI workflow lint job to use build tags
    - **Action:** Modify the lint action in `.github/workflows/ci.yml` to include build tags:
      ```yaml
      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v7
        with:
          version: v2.1.1
          args: --verbose --build-tags=test_without_external_deps
      ```
    - **Depends On:** None
    - **AC Ref:** None

- [ ] **T104:** Add dependency information to go.mod and go.sum
    - **Action:** Run `go mod tidy` locally and commit the changes to `go.mod` and `go.sum` to ensure proper dependency tracking
    - **Depends On:** None
    - **AC Ref:** None

- [ ] **T105:** Document build tag usage in README
    - **Action:** Add a section to the project README.md explaining how to work with build tags for testing with and without external dependencies
    - **Depends On:** [T102, T103]
    - **AC Ref:** None

- [ ] **T106:** Test the updated CI workflow
    - **Action:** Create a test PR to verify that the CI workflow succeeds with the updated configuration
    - **Depends On:** [T102, T103, T104]
    - **AC Ref:** None

- [ ] **T107:** Complete the implementation and mark original task as done
    - **Action:** Review all changes, make any necessary adjustments, and mark task F001 as completed
    - **Depends On:** [T101, T102, T103, T104, T105, T106]
    - **AC Ref:** None

