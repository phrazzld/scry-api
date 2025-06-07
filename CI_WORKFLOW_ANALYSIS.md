# CI Workflow Test Execution Configuration Analysis

## Issues Found

### Critical Issue 1: Inconsistent Build Tags
**Problem**: CI test execution uses inconsistent build tags that miss test files.

- **Lint Job**: Uses `--build-tags=test_without_external_deps`
- **Test Job**: Uses `-tags=integration`

**Impact**: Test files tagged with only `test_without_external_deps` are excluded from CI test runs, causing:
- Lower coverage than expected (0.0% vs 53.6% for card_review)
- Missing test execution for critical functionality
- False confidence in CI results

**Evidence**:
```bash
# With integration tag only (CI current):
$ go list -f '{{.XTestGoFiles}}' -tags="integration" ./internal/service/card_review/
[comprehensive_unit_test.go service_error_handling_test.go service_errors_test.go service_integration_test.go service_outcome_validation_test.go service_test.go submit_answer_simple_test.go]

# Missing: service_mock_test.go, submit_answer_coverage_test.go

# With both tags (correct):
$ go list -f '{{.XTestGoFiles}}' -tags="integration,test_without_external_deps" ./internal/service/card_review/
[comprehensive_unit_test.go service_error_handling_test.go service_errors_test.go service_integration_test.go service_mock_test.go service_outcome_validation_test.go service_test.go submit_answer_coverage_test.go submit_answer_simple_test.go]
```

**Solution**: Update CI test command to use both tags: `-tags=integration,test_without_external_deps`

### Issue 2: Go Version Mismatch
**Problem**: Version drift between CI and development environments.

- **CI Environment**: Go 1.22
- **Local Environment**: Go 1.24.2

**Impact**:
- Potential behavior differences in compilation/runtime
- Different optimization strategies
- Possible test result variations

**Solution**: Update CI to use Go 1.24.2 or standardize local development to Go 1.22

### Issue 3: Test Command Differences
**Problem**: CI uses different test flags than local development.

**CI Command**:
```bash
go test -v -json -race -coverprofile=coverage-${PACKAGE_NAME}.out -tags=integration ./${{ matrix.package }}/...
```

**Local Commands**:
```bash
make test                # go test ./...
make test-integration    # go test -v -tags=integration ./...
make test-no-deps       # go test -v -tags=test_without_external_deps ./...
make test-coverage      # go test -cover -tags="test_without_external_deps" ./...
```

**Impact**:
- Different test discovery due to tags
- Different output formats (JSON vs standard)
- Race detection only in CI
- Coverage collection varies

## Recommended Fixes

### 1. Fix CI Test Tags (Critical)
Update `.github/workflows/ci.yml` line 343:
```yaml
# FROM:
go test -v -json -race -coverprofile=coverage-${PACKAGE_NAME}.out -tags=integration ./${{ matrix.package }}/...

# TO:
go test -v -json -race -coverprofile=coverage-${PACKAGE_NAME}.out -tags=integration,test_without_external_deps ./${{ matrix.package }}/...
```

### 2. Standardize Go Version
Update `.github/workflows/ci.yml` line 48:
```yaml
# FROM:
GO_VERSION: '1.22'

# TO:
GO_VERSION: '1.24'
```

### 3. Add Local Commands to Match CI
Add to Makefile:
```makefile
.PHONY: test-ci-local
test-ci-local: ## Run tests matching CI environment
	go test -v -race -cover -tags=integration,test_without_external_deps ./...

.PHONY: test-ci-package
test-ci-package: ## Run tests for specific package matching CI (use PKG=package/path)
	go test -v -json -race -coverprofile=coverage.out -tags=integration,test_without_external_deps ./$(PKG)/...
```

### 4. Update Lint Configuration
Ensure golangci-lint uses consistent build tags:
```yaml
# In .github/workflows/ci.yml line 149:
args: --verbose --build-tags=integration,test_without_external_deps
```

## Environment Variables Verification

CI sets extensive environment variables that may not be present locally:
- Database URLs: `DATABASE_URL`, `SCRY_TEST_DB_URL`, `SCRY_DATABASE_URL`
- Auth config: `SCRY_AUTH_JWT_SECRET`, `SCRY_AUTH_BCRYPT_COST`, etc.
- LLM config: `SCRY_LLM_GEMINI_API_KEY`, `SCRY_LLM_MODEL_NAME`, etc.
- Server config: `SCRY_SERVER_PORT`, `SCRY_SERVER_LOG_LEVEL`, etc.

**Action**: Document required environment variables for local testing in development docs.

## Test Count Verification

After applying fixes, verify test execution matches between environments:

```bash
# Local verification:
go test -tags=integration,test_without_external_deps -run=Test ./internal/service/card_review/ -v | grep "=== RUN" | wc -l

# Should match CI test count for this package
```

## Priority

1. **P0**: Fix CI build tags (critical for test coverage)
2. **P1**: Update Go version consistency  
3. **P2**: Add local CI-matching commands
4. **P3**: Document environment setup for local testing
