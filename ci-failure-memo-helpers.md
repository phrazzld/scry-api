# CI Failure Audit - Missing Test Helper Functions

## Issue Description

The CI build is failing with errors related to missing test helper functions in the `memo_store_test.go` file. The specific errors are:

```
undefined: testutils.MustInsertMemo
undefined: testutils.CreateTestMemo
undefined: testutils.CountMemos
```

## Root Cause Analysis

After investigation, I've found that there's a dependency issue between test helper functions. The issue appears to be with build tags and package structure:

1. The `memo_store_test.go` file is importing test helpers from the `testutils` package
2. The file is using functions like `MustInsertMemo`, `CreateTestMemo`, and `CountMemos`
3. While functions like `MustCreateMemoForTest` and `CountMemos` exist in `/internal/testutils/card_helpers.go` and `/internal/testutils/api_helpers.go`, they have specific build tags:
   ```go
   //go:build (!compatibility && ignore_redeclarations) || test_without_external_deps
   ```
4. The `CreateTestMemo` function reference appears to be a typo - the function in the testutils package is named `MustCreateMemoForTest` or possibly `CreateMemoForTest`

## Build Tag Issues

The build tag on `card_helpers.go` is:
```go
//go:build (!compatibility && ignore_redeclarations) || test_without_external_deps
```

This build tag means the file is only included:
- When both `!compatibility` AND `ignore_redeclarations` tags are provided, OR
- When `test_without_external_deps` tag is provided

However, the `memo_store_test.go` file uses:
```go
//go:build integration
```

This means it's only compiled with the `integration` tag, which doesn't match the conditions needed to include the helper functions.

## Implementation Plan

There are two approaches to fix this issue:

### Option 1: Update Build Tags

Update the build tags in the `card_helpers.go` and possibly `api_helpers.go` files to include the `integration` tag, so they're compiled correctly:

```go
//go:build ((!compatibility && ignore_redeclarations) || test_without_external_deps || integration)
```

### Option 2: Fix Function References

1. Fix typos or update the function references in `memo_store_test.go`:
   - Replace `CreateTestMemo` with `CreateMemoForTest` or `MustCreateMemoForTest`
   - Replace `MustInsertMemo` with a proper test setup function that inserts memos

2. Create a dedicated `memo_helpers.go` file in `internal/testutils` that includes the missing helper functions with appropriate build tags that match the integration tests.

## Recommended Approach

Option 2 is more maintainable in the long run:

1. Create a new file `internal/testutils/memo_helpers.go` with the `integration` build tag
2. Implement the missing helper functions there
3. Ensure it matches what's being used in `memo_store_test.go`

This approach keeps the build tag organization clean and creates a proper separation between different test helper types.

## Next Steps

1. Create the `memo_helpers.go` file with the missing functions
2. Update any incorrect references in `memo_store_test.go`
3. Verify CI passes after the changes
