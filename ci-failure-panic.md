# CI Failure: Panic in Card Edit Integration Test

## Issue Analysis

We've fixed the previous validation and import issues, but now we're encountering a panic in the card edit integration test:

```
=== RUN   TestCardEditIntegration
=== RUN   TestCardEditIntegration/Success
=== RUN   TestCardEditIntegration/Card_Not_Found

 panic: runtime error: invalid memory address or nil pointer dereference

 -> log/slog.(*commonHandler).handle
 ->   /home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.2.linux-amd64/src/log/slog/handler.go:313
```

The stack trace shows this is happening in the card service's update functionality:

```
github.com/phrazzld/scry-api/internal/service.(*cardServiceImpl).UpdateCardContent
   /home/runner/work/scry-api/scry-api/internal/service/card_service.go:313
```

The issue appears to be a nil pointer dereference when trying to handle a "card not found" test case. This suggests that we're likely not handling the "not found" case correctly in the test setup.

## Root Cause

The test is trying to simulate a "card not found" scenario, but there's likely an issue with how the test is set up. When the code tries to access the logger or some other dependency, it encounters a nil pointer.

Looking at line 313 in card_service.go, this is likely in the logging functionality where a logger is being used but might be nil in the test context.

## Fix Required

1. Examine the `TestCardEditIntegration/Card_Not_Found` test case to understand how it's simulating the "not found" scenario.

2. Fix the service implementation to handle the "not found" case more gracefully, ensuring all necessary dependencies are initialized properly.

3. If appropriate, modify the test to provide proper mocks or test doubles for all dependencies that the service might use, even in error cases.

## Implementation Plan

1. Check the card_api_test.go file to understand how the "Card_Not_Found" test is implemented.

2. Examine the card_service.go implementation to see why it's panicking when a card is not found.

3. Update the code to handle the "not found" case properly, ensuring it doesn't try to use a nil dependency.

4. Consider adding test coverage specifically for graceful error handling in the card service.
