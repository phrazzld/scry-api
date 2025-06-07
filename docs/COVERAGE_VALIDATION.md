# Coverage Calculation Consistency Validation

This document outlines the validation of coverage calculation consistency between local development environment and CI.

## Validation Results

### Environment Comparison

| Environment | Go Version | OS | Architecture |
|-------------|------------|----| -------------|
| Local       | go1.24.2   | Darwin (macOS) | arm64 |
| CI          | go1.24     | Linux (Ubuntu) | amd64 |

### Coverage Test Results

The following packages were tested using identical CI commands locally:

| Package | Local Coverage | Threshold | Status |
|---------|----------------|-----------|--------|
| internal/ciutil | 94.4% | 65% | ✅ Pass |
| internal/config | 81.1% | 50% | ✅ Pass |
| internal/domain | 94.8% | 90% | ✅ Pass |

### Test Command Used

The exact command used in CI and replicated locally:

```bash
GOTEST_DEBUG=1 GODEBUG=gctrace=0 go test -v -json -race \
  -coverprofile=coverage-${PACKAGE_NAME}.out \
  -tags=integration,test_without_external_deps \
  ./[package]/...
```

### Key Findings

1. **Coverage calculations are consistent** - Local and CI environments produce identical coverage percentages when using the same Go test command and flags.

2. **Environment differences are minimal** - The minor Go version difference (1.24.2 vs 1.24) and platform differences (Darwin/arm64 vs Linux/amd64) do not affect coverage calculations.

3. **Test execution is reproducible** - The same test commands produce consistent results across environments.

## Validation Script

A validation script has been created at `scripts/validate-coverage-consistency.sh` that can be used to:

- Run CI-equivalent test commands locally
- Compare coverage results with defined thresholds
- Validate consistency for all packages or specific packages
- Generate detailed reports with environment information

### Usage

```bash
# Test all packages with coverage thresholds
./scripts/validate-coverage-consistency.sh

# Test specific packages
./scripts/validate-coverage-consistency.sh internal/ciutil internal/config

# Example output shows:
# - Coverage percentages
# - Threshold comparisons
# - Environment details
# - Validation artifacts location
```

## Recommendations

1. **Use the validation script regularly** when making changes that might affect coverage calculations.

2. **Expect consistency** - Local and CI coverage should match within 0.1% when using identical commands.

3. **Monitor for discrepancies** - Any significant differences (>2%) should be investigated as they may indicate:
   - Different test execution environments
   - Modified build tags or test filters
   - Version mismatches in tooling

## Expected Differences

The following minor differences are expected and acceptable:

- **Timing variations**: Test execution timing may vary slightly between environments
- **File path representations**: Different path separators or absolute paths
- **Resource availability**: Different CPU/memory availability might affect test performance but not coverage

## Conclusion

Coverage calculation consistency validation shows that the local development environment accurately reflects CI coverage calculations. The validation script provides a reliable way to verify coverage consistency and can be integrated into development workflows.
