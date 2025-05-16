# API Test Helpers

This package provides standardized helpers for API integration testing in the Scry API project.

## Overview

The test helpers are structured to provide a consistent approach to:

1. Setting up test servers with appropriate middleware and routes
2. Making HTTP requests with proper authentication and headers
3. Creating test data (cards, stats, etc.) for testing
4. Asserting on responses (status codes, body contents, error messages)

## Modules

### 1. Server Setup (`server_setup.go`)

Provides functions for creating test HTTP servers with consistent configuration:

```go
// Basic usage
server := api.SetupTestServer(t, api.TestServerOptions{
    Tx: tx,  // Required transaction for test isolation
    Logger: logger,  // Optional
    JWTService: jwtService, // Optional
})

// Specialized server setup for specific API types
server := api.SetupCardManagementTestServer(t, tx)
server := api.SetupCardReviewTestServer(t, tx)
server := api.SetupAuthTestServer(t, tx)
```

### 2. Request Helpers (`request_helpers.go`)

Makes HTTP requests to test servers with consistent configuration:

```go
// Basic request
resp, err := api.ExecuteRequest(t, server, "GET", "/api/path", nil)

// With authentication
resp, err := api.ExecuteAuthenticatedRequest(t, server, "GET", "/api/path", nil, authToken)

// JSON requests
resp, err := api.ExecuteJSONRequest(t, server, "POST", "/api/path", payload)
resp, err := api.ExecuteAuthenticatedJSONRequest(t, server, "POST", "/api/path", payload, authToken)
```

### 3. Card Helpers (`card_helpers.go`)

Creates and validates card-related test data:

```go
// Create test cards
card := api.CreateCardForAPITest(t,
    api.WithCardUserID(userID),
    api.WithCardContent(map[string]interface{}{"front": "Question", "back": "Answer"}),
)

// Create test stats
stats := api.CreateStatsForAPITest(t,
    api.WithStatsUserID(userID),
    api.WithStatsCardID(cardID),
)
```

### 4. Response Assertions

Assert on response status codes and content:

```go
// Assert status code
api.AssertResponse(t, resp, http.StatusOK)

// Assert JSON response
var result MyResponseType
api.AssertJSONResponse(t, resp, http.StatusOK, &result)

// Assert error response
api.AssertErrorResponse(t, resp, http.StatusBadRequest, "error message part")

// Assert specific response types
api.AssertCardResponse(t, resp, expectedCard)
api.AssertStatsResponse(t, resp, expectedStats)
api.AssertValidationError(t, resp, "field", "error message part")
```

## Using in Tests

To use these helpers in your integration tests:

1. Import the package:
   ```go
   import "github.com/phrazzld/scry-api/internal/testutils/api"
   ```

2. Set up transaction isolation:
   ```go
   testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
       // Test code here
   })
   ```

3. Create a test server:
   ```go
   server := api.SetupTestServer(t, api.TestServerOptions{
       Tx: tx,
   })
   ```

4. Make requests and assert on responses:
   ```go
   resp, err := api.ExecuteAuthenticatedRequest(t, server, "GET", "/api/endpoint", nil, authToken)
   require.NoError(t, err)
   api.AssertResponse(t, resp, http.StatusOK)
   ```

## Best Practices

1. Always use transaction isolation for database operations
2. Make use of the helper functions to reduce boilerplate code
3. Prefer the standardized helpers to custom implementations
4. Create data objects with the functional options pattern for flexibility
5. Use table-driven tests to test multiple scenarios
6. Explicitly check for errors in responses

## Notes

- Resources are automatically cleaned up using `t.Cleanup()`
- Authentication tokens are generated using the test JWT service
- All helper methods have been designed to work with the testing package

For more examples, see the integration tests in the `cmd/server` package.
