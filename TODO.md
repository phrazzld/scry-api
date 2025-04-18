# Gemini API Modernization Tasks

## Implementation Plan

- [ ] **M001:** Research modern Google AI APIs
    - **Action:** Thoroughly investigate `google.golang.org/genai` and `cloud.google.com/go/ai/generativelanguage/apiv1` to determine the most appropriate replacement for `google.golang.org/api/ai/generativelanguage/v1beta`
    - **Deliverable:** Evaluation document with pros/cons of each option and recommendation
    - **Depends On:** None
    - **AC Ref:** None

- [ ] **M002:** Update dependency management
    - **Action:** Update go.mod to use the selected modern package and ensure all dependencies are compatible
    - **Deliverable:** Updated go.mod and go.sum files with correct dependencies
    - **Depends On:** [M001]
    - **AC Ref:** None

- [ ] **M003:** Refactor GeminiGenerator client initialization
    - **Action:** Modify the NewGeminiGenerator function in gemini_generator.go to use the new client initialization methods
    - **Deliverable:** Updated initialization code with proper error handling and configuration
    - **Depends On:** [M002]
    - **AC Ref:** None

- [ ] **M004:** Refactor API call methods
    - **Action:** Replace `callGeminiWithRetry` method implementation to use new API call patterns
    - **Deliverable:** Updated implementation that maintains the same retry and error handling logic but uses new API methods
    - **Depends On:** [M003]
    - **AC Ref:** None

- [ ] **M005:** Update response parsing
    - **Action:** Modify `parseResponse` method to handle the new response format from the modern API
    - **Deliverable:** Updated parser that correctly extracts card data from the new response structure
    - **Depends On:** [M004]
    - **AC Ref:** None

- [ ] **M006:** Update GenerateCards implementation
    - **Action:** Update the main interface method to use the new underlying implementations
    - **Deliverable:** Fully functional GenerateCards method using modern APIs while maintaining the same interface
    - **Depends On:** [M004, M005]
    - **AC Ref:** None

- [ ] **M007:** Update tests for the new implementation
    - **Action:** Update all GeminiGenerator test cases to work with the new implementation
    - **Deliverable:** Complete test coverage for the modernized implementation
    - **Depends On:** [M006]
    - **AC Ref:** None

- [ ] **M008:** Update mock implementation for testing
    - **Action:** Ensure the mock implementation in gemini_generator_mock.go is compatible with the new real implementation
    - **Deliverable:** Updated mock implementation that provides consistent test behavior
    - **Depends On:** [M007]
    - **AC Ref:** None

- [ ] **M009:** Complete verification and documentation
    - **Action:** Ensure all tests pass with both real and mock implementations, update all relevant documentation
    - **Deliverable:** Passing tests, updated documentation, and successful lint checks
    - **Depends On:** [M007, M008]
    - **AC Ref:** None

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

- [x] **T103:** Update CI workflow lint job to use build tags
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

- [x] **T104:** Add dependency information to go.mod and go.sum
    - **Action:** Run `go mod tidy` locally and commit the changes to `go.mod` and `go.sum` to ensure proper dependency tracking
    - **Depends On:** None
    - **AC Ref:** None

- [x] **T105:** Document build tag usage in README
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

