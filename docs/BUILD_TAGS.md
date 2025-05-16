# Go Build Tags Usage Policy

This document establishes guidelines for using Go build tags in the Scry API project. Build tags are a powerful feature but must be used judiciously to avoid compilation issues and maintain code clarity.

## What are Build Tags?

Build tags are special comments in Go that control which files are included during compilation. They can be specified in two formats:

```go
//go:build tag_name

// or the legacy format:
// +build tag_name
```

## When to Use Build Tags

### ✅ Approved Use Cases

1. **Test Isolation**
   - Use `test_without_external_deps` for test files that should run without external dependencies
   - Example: Mock implementations for CI environments
   ```go
   //go:build test_without_external_deps

   package gemini

   // Mock implementation for testing without Gemini API
   ```

2. **Platform-Specific Code**
   - Use OS/architecture tags for platform-specific implementations
   - Example: `//go:build linux` or `//go:build windows`
   ```go
   //go:build linux

   package system

   // Linux-specific implementation
   ```

3. **Integration vs Unit Tests**
   - Use `integration` tag for tests requiring external services
   - Keep unit tests without special tags
   ```go
   //go:build integration

   package api_test

   // Tests requiring database connection
   ```

## When NOT to Use Build Tags

### ❌ Prohibited Use Cases

1. **Core Application Logic**
   - NEVER use restrictive build tags on core application files
   - All main package files should be accessible in normal builds
   - Bad example:
   ```go
   //go:build exported_core_functions  // DON'T DO THIS!

   package main

   func loadAppConfig() { /* ... */ }
   ```

2. **Essential Business Logic**
   - Domain models, services, and repositories should not have build tags
   - These components must be available in all builds

3. **API Endpoints and Handlers**
   - HTTP handlers and API routes must be accessible without special tags
   - Build tags would prevent normal API functionality

## Common Pitfalls

### Issue: Undefined Functions in main.go

**Problem**: Using restrictive build tags like `//go:build exported_core_functions` on files containing functions needed by main.go causes compilation errors.

**Solution**: Remove build tags from all files in the main package that contain essential application functions.

### Issue: Test Dependencies in Production

**Problem**: Accidentally including test utilities in production builds.

**Solution**: Use `_test.go` suffix for test files or appropriate build tags for test-only utilities.

## Best Practices

1. **Document Tag Usage**
   - Always add a comment explaining why a build tag is necessary
   - Example:
   ```go
   //go:build test_without_external_deps

   // This file provides mock implementations for testing without external API calls.
   // Used primarily in CI environments where API keys may not be available.
   ```

2. **Prefer File Naming Over Tags**
   - Use `_test.go` suffix for test files instead of build tags when possible
   - Use clear file names like `mock_client.go` or `integration_test.go`

3. **Keep Tags Simple**
   - Avoid complex tag expressions unless absolutely necessary
   - If you need complex conditions, document them thoroughly

4. **CI/CD Considerations**
   - Document which tags are used in CI pipeline
   - Ensure CI configuration matches local development patterns
   - Current CI uses: `--build-tags=test_without_external_deps`

## Project-Specific Tags

| Tag | Purpose | Usage |
|-----|---------|--------|
| `test_without_external_deps` | Exclude external API calls in tests | CI environments, local testing without API keys |
| `integration` | Mark tests requiring external services | Database tests, API integration tests |

## Code Review Checklist

When reviewing code with build tags:

1. ✓ Is the build tag necessary?
2. ✓ Could the same goal be achieved without tags?
3. ✓ Is the tag documented?
4. ✓ Does it affect core application functionality?
5. ✓ Will it work in all required environments (local, CI, production)?

## Examples from This Project

### Good: Test Mock Implementation
```go
//go:build test_without_external_deps

package gemini

// MockGenerator provides a test implementation that doesn't call external APIs
type MockGenerator struct{}
```

### Bad: Core Application Function with Restrictive Tag
```go
//go:build exported_core_functions  // WRONG!

package main

func setupAppDatabase() (*sql.DB, error) {
    // This function is needed by main() and shouldn't have build tags
}
```

## References

- [Go Build Constraints Documentation](https://pkg.go.dev/go/build#hdr-Build_Constraints)
- [Go 1.17+ Build Tag Format](https://go.dev/doc/go1.17#gofmt)
- Project's [Development Philosophy](./DEVELOPMENT_PHILOSOPHY.md)

## Enforcement

- Pre-commit hooks check for proper build tag usage
- CI pipeline validates build with various tag combinations
- Code reviews must verify compliance with these guidelines
