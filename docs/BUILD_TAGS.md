# Build Tags Documentation

This document defines the standard build tags used in the Scry API project and provides guidelines for their usage.

## Table of Contents

- [Overview](#overview)
- [Standard Build Tags](#standard-build-tags)
- [Tag Combination Rules](#tag-combination-rules)
- [Package-Specific Guidelines](#package-specific-guidelines)
- [Best Practices](#best-practices)
- [Common Patterns](#common-patterns)
- [Validation](#validation)

## Overview

Build tags in Go control which files are included during compilation. This project uses build tags to:
- Separate integration tests from unit tests
- Control function visibility across packages
- Manage test utilities and mocks
- Enable CI-specific configurations

## Standard Build Tags

### Core Tags

| Tag | Purpose | Usage |
|-----|---------|-------|
| `integration` | Tests requiring external services (DB, APIs) | Integration test files |
| `test_without_external_deps` | Mock external dependencies for CI | CI environments, local testing without services |
| `exported_core_functions` | Functions needed in production code | Utilities that cross test/production boundary |
| `integration_test_internal` | Internal control for test utilities | Preventing function redeclarations |

### Usage Examples

```go
// Integration tests requiring real database
//go:build integration

// CI-compatible tests
//go:build integration || test_without_external_deps

// Production-compatible utilities
//go:build exported_core_functions

// Complex combinations
//go:build (integration || test_without_external_deps) && exported_core_functions
```

## Tag Combination Rules

### 1. OR Logic (`||`)
Use OR when providing alternatives:
```go
//go:build integration || test_without_external_deps
```
This allows the file to be included in either integration tests OR CI environments.

### 2. AND Logic (`&&`)
Use AND when multiple conditions must be met:
```go
//go:build integration && exported_core_functions
```
This requires BOTH tags to be present.

### 3. Negation (`!`)
Use negation to exclude files in certain builds:
```go
//go:build !integration_test_internal
```
This excludes the file when the tag is present.

### 4. Grouping
Use parentheses for complex expressions:
```go
//go:build (integration || test_without_external_deps) && !integration_test_internal
```

## Package-Specific Guidelines

### testutils Package

The `testutils` package requires careful tag management due to function forwarding:

1. **compatibility.go**:
   ```go
   //go:build integration_test_internal
   ```
   Original implementations, excluded by default.

2. **db_forwarding.go**:
   ```go
   //go:build !integration_test_internal && !ignored_build_tag_file
   ```
   Forwards to testdb package, included by default.

3. **integration_exports.go**:
   ```go
   //go:build integration && !test_without_external_deps && !integration_test_internal
   ```
   Critical functions for postgres integration tests.

### Mock Packages

Mock implementations should use consistent tags:
```go
//go:build test_without_external_deps || integration
```

### Command Package Tests

Server tests often need both integration and export tags:
```go
//go:build (integration || test_without_external_deps) && exported_core_functions
```

## Best Practices

### 1. Prefer New Syntax
Always use `//go:build` instead of `// +build`:
```go
// Good
//go:build integration

// Avoid
// +build integration
```

### 2. Keep Expressions Simple
Avoid overly complex expressions. If you need more than 2 operators, consider refactoring:
```go
// Too complex
//go:build (a || b) && (c || d) && !e && !f

// Better - split into multiple files or simplify logic
//go:build (integration || test_without_external_deps) && production_ready
```

### 3. Document Non-Obvious Tags
Add comments explaining why specific tags are used:
```go
//go:build !integration_test_internal && !test_without_external_deps
// This file provides forwarding functions for standard builds
// while avoiding conflicts with the integration test environment
```

### 4. Ensure CI Compatibility
Always provide a fallback for CI environments:
```go
// Good - works in CI
//go:build integration || test_without_external_deps

// Bad - might not work in CI
//go:build integration
```

### 5. Avoid Tag Conflicts
Never use both positive and negative forms of the same tag in the same package without careful consideration.

## Common Patterns

### Integration Test Pattern
```go
//go:build integration || test_without_external_deps

package mypackage_test

// Test file that works with real or mocked dependencies
```

### Shared Test Utilities
```go
//go:build (integration || test_without_external_deps) && exported_core_functions

package testutils

// Utilities available in both test and production contexts
```

### CI-Only Code
```go
//go:build test_without_external_deps && !integration

package mocks

// Mock implementations only for CI
```

### Production-Compatible Test Helpers
```go
//go:build exported_core_functions

package helpers

// Helpers that can be imported by production code
```

## Validation

### Running Validation

Use the build tag validation script:
```bash
# Full validation
./scripts/validate-build-tags.sh

# Keep audit report for debugging
KEEP_AUDIT=true ./scripts/validate-build-tags.sh
```

### What Validation Checks

1. **Conflict Detection**: Identifies tags that are both included and excluded
2. **CI Compatibility**: Ensures critical functions are available in CI
3. **Complexity**: Warns about overly complex tag expressions
4. **Build Testing**: Verifies common tag combinations compile

### Pre-commit Hook

Add validation to your pre-commit hooks:
```yaml
- repo: local
  hooks:
    - id: validate-build-tags
      name: Validate Go build tags
      entry: scripts/validate-build-tags.sh
      language: script
      files: '\.go$'
      pass_filenames: false
```

## Troubleshooting

### Common Issues

1. **"undefined function" in CI**
   - Check if the function's file has `test_without_external_deps` in its build tags
   - Verify the function is in `integration_exports.go` if needed

2. **"function redeclared" errors**
   - Look for overlapping build tags
   - Check negation patterns

3. **Tests not running**
   - Verify build tags match test command
   - Check for typos in tag names

### Debug Commands

```bash
# See which files are included with specific tags
go list -f '{{.GoFiles}}' -tags=integration ./internal/testutils

# Check active build tags
go list -f '{{.BuildTags}}' ./...

# Test specific tag combination
go test -tags="integration,exported_core_functions" ./...
```
