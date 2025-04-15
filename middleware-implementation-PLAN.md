# Add Tests for Middleware Components: Implementation Plan

## Selected Approach: In-Process HTTP Integration Testing (`httptest`)

Based on the analysis from both the gemini-2.0-flash and gemini-2.5-pro-exp-03-25 models, the recommended approach is to use in-process HTTP integration testing with `net/http/httptest`. This approach offers the best alignment with the project's standards, particularly the testing strategy that emphasizes behavior over implementation and minimal mocking.

## Implementation Plan

### 1. Set Up Test Infrastructure

1. Create a new test file `internal/api/middleware/auth_middleware_test.go` (once the middleware is implemented)
2. Import necessary packages:
   - `net/http/httptest`
   - `testing`
   - `github.com/stretchr/testify/assert`
   - `github.com/stretchr/testify/require`
   - Project-specific imports for auth and middleware

3. Create helper functions for token generation:
   ```go
   // generateValidToken creates a valid JWT token for testing
   func generateValidToken(t *testing.T, userID string, secret string) string {
       // Token generation logic
   }

   // generateExpiredToken creates an expired JWT token for testing
   func generateExpiredToken(t *testing.T, userID string, secret string) string {
       // Expired token generation logic
   }

   // generateMalformedToken creates an invalid JWT token for testing
   func generateMalformedToken(t *testing.T) string {
       // Malformed token generation
   }
   ```

### 2. Create Test Handler

Create a simple handler that will be protected by the middleware:

```go
func testHandler(w http.ResponseWriter, r *http.Request) {
    // Get user from context (added by middleware)
    user, ok := r.Context().Value(userContextKey).(*domain.User)
    if !ok {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    // Write user ID to response for verification
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(user.ID.String()))
}
```

### 3. Implement Test Cases

Implement the following test cases to cover all acceptance criteria:

#### 3.1 Valid Token

```go
func TestAuthMiddleware_ValidToken(t *testing.T) {
    // Arrange
    // - Set up middleware and auth configuration
    // - Generate valid token
    // - Create request with token in Authorization header
    // - Create response recorder

    // Act
    // - Call the middleware-wrapped handler

    // Assert
    // - Verify 200 OK status code
    // - Verify user ID in response matches expected user
}
```

#### 3.2 Invalid Token

```go
func TestAuthMiddleware_InvalidToken(t *testing.T) {
    // Arrange
    // - Set up middleware and auth configuration
    // - Generate invalid token (wrong signature)
    // - Create request with invalid token in Authorization header
    // - Create response recorder

    // Act
    // - Call the middleware-wrapped handler

    // Assert
    // - Verify 401 Unauthorized status code
    // - Verify error message in response body
}
```

#### 3.3 Expired Token

```go
func TestAuthMiddleware_ExpiredToken(t *testing.T) {
    // Arrange
    // - Set up middleware and auth configuration
    // - Generate expired token
    // - Create request with expired token in Authorization header
    // - Create response recorder

    // Act
    // - Call the middleware-wrapped handler

    // Assert
    // - Verify 401 Unauthorized status code
    // - Verify error message indicates token expiration
}
```

#### 3.4 Missing Token

```go
func TestAuthMiddleware_MissingToken(t *testing.T) {
    // Arrange
    // - Set up middleware and auth configuration
    // - Create request with no Authorization header
    // - Create response recorder

    // Act
    // - Call the middleware-wrapped handler

    // Assert
    // - Verify 401 Unauthorized status code
    // - Verify error message indicates missing token
}
```

#### 3.5 Role-Based Access Control (if implemented)

```go
func TestAuthMiddleware_RoleBasedAccess(t *testing.T) {
    // Arrange
    // - Set up middleware and auth configuration
    // - Generate valid token with specific role claims
    // - Create request with token in Authorization header
    // - Create response recorder
    // - Create middleware with specific role requirement

    // Act
    // - Call the middleware-wrapped handler

    // Assert
    // - Verify appropriate status code (200 for authorized, 403 for unauthorized)
}
```

### 4. Additional Test Cases for Edge Conditions

```go
func TestAuthMiddleware_MalformedToken(t *testing.T) {
    // Test behavior with a malformed token
}

func TestAuthMiddleware_WrongTokenFormat(t *testing.T) {
    // Test behavior when Authorization header doesn't use "Bearer" format
}

func TestAuthMiddleware_UserNotFound(t *testing.T) {
    // Test behavior when token is valid but user doesn't exist in system
}
```

### 5. Test Table-Driven Tests

Where appropriate, consider implementing table-driven tests to reduce code duplication:

```go
func TestAuthMiddleware(t *testing.T) {
    testCases := []struct {
        name           string
        token          string
        expectedStatus int
        expectedBody   string
    }{
        {"ValidToken", generateValidToken(t, userId, secret), http.StatusOK, userId},
        {"InvalidToken", generateMalformedToken(t), http.StatusUnauthorized, "invalid token"},
        // Additional test cases...
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Notes

1. This plan assumes the middleware will be implemented following the project's architecture guidelines, particularly:
   - Dependency inversion for external dependencies
   - Clear separation of concerns
   - Strong typing for user information in context

2. The tests will need to be updated if the actual middleware implementation differs from the assumptions made here.

3. No mocking of internal collaborators should be needed. If the tests become difficult to write without extensive mocking, it suggests the middleware implementation itself should be refactored for better testability before proceeding with testing.

4. Implementation of these tests should begin once the actual authentication middleware is complete or nearing completion.
