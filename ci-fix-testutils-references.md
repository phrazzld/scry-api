# CI Fix: TestUtils Package References

## Issue Analysis

The CI build is failing because the `cmd/server` tests try to use functions from the `testutils` package that are now defined in the `testutils/db` subpackage:

```
cmd/server/main_integration_test.go:102:12: undefined: testutils.WithTx
cmd/server/main_integration_test.go:292:12: undefined: testutils.WithTx
cmd/server/main_task_test.go:54:25: undefined: testutils.SetupTestDatabaseSchema
cmd/server/main_task_test.go:253:12: undefined: testutils.WithTx
```

## Root Cause

The codebase has undergone a refactoring where the database-related test utilities were moved from the `internal/testutils` package to a more specific `internal/testutils/db` package. However, some test files in `cmd/server` still directly reference the old paths.

The refactoring is evident by comparing two implementations:
1. `internal/testutils/db/db.go` - the new location with proper implementations
2. `internal/testutils/db.go` - appears to have forwarding functions for compatibility

## Fix Implementation

I've implemented the following changes:

1. Added the missing import in the `cmd/server` test files:
   ```go
   import (
       // ... existing imports
       "github.com/phrazzld/scry-api/internal/testutils/db"
   )
   ```

2. Updated function calls to use the correct package:
   - Changed `testutils.WithTx(...)` to `db.WithTx(...)`
   - Changed `testutils.SetupTestDatabaseSchema(...)` to `db.SetupTestDatabaseSchema(...)`

## Future Recommendations

1. **Consistent Package Structure**: When refactoring utilities into subpackages, it's helpful to either:
   - Provide forwarding functions in the parent package, or
   - Update all references to use the new package structure

2. **Import Path Consistency**: Consider using a linting tool like `importas` to enforce consistent import paths for specific packages across the codebase.

3. **Documentation**: Add package documentation that explains the package structure and preferred import patterns to help developers use the right imports.
