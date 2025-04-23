package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorDetailsLeak verifies that internal error details are not leaked in API responses
func TestErrorDetailsLeak(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	// Test cases with different types of sensitive error details
	testCases := []struct {
		name               string
		error              error
		expectedStatusCode int
		expectedMessage    string
		endpoint           string
	}{
		{
			name: "SQL error details should not leak",
			error: errors.New(
				"SQL error: syntax error at line 42 in query SELECT * FROM cards WHERE user_id = '...'",
			),
			expectedStatusCode: http.StatusInternalServerError,
			expectedMessage:    "Failed to get next review card",
			endpoint:           "next-card",
		},
		{
			name: "Database connection details should not leak",
			error: errors.New(
				"connection error: could not connect to database at postgres://user:password@localhost:5432/db",
			),
			expectedStatusCode: http.StatusInternalServerError,
			expectedMessage:    "Failed to get next review card",
			endpoint:           "next-card",
		},
		{
			name: "Stack trace details should not leak",
			error: fmt.Errorf(
				"runtime error: %w",
				errors.New(
					"panic: invalid memory address or nil pointer dereference [recovered]\n\tstack trace: goroutine 42...",
				),
			),
			expectedStatusCode: http.StatusInternalServerError,
			expectedMessage:    "Failed to get next review card",
			endpoint:           "next-card",
		},
		{
			name: "System path details should not leak",
			error: fmt.Errorf(
				"file not found: %w",
				errors.New("/var/lib/postgresql/data/base/16384/2619: No such file or directory"),
			),
			expectedStatusCode: http.StatusInternalServerError,
			expectedMessage:    "Failed to get next review card",
			endpoint:           "next-card",
		},
		{
			name: "AWS credentials should not leak",
			error: errors.New(
				"AccessDenied: User: arn:aws:iam::123456789012:user/admin is not authorized; AWSAccessKeyId: AKIAIOSFODNN7EXAMPLE",
			),
			expectedStatusCode: http.StatusInternalServerError,
			expectedMessage:    "Failed to get next review card",
			endpoint:           "next-card",
		},
		{
			name: "Card submission error details should not leak",
			error: errors.New(
				"failed to update card_stats record for user abc123 with error: database constraints violation",
			),
			expectedStatusCode: http.StatusInternalServerError,
			expectedMessage:    "Failed to submit answer",
			endpoint:           "submit-answer",
		},
	}

	for _, testCase := range testCases {
		tc := testCase // Create a new variable to avoid loop variable capture
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var resp *http.Response
			var err error

			// Setup test server with specific error
			if tc.endpoint == "next-card" {
				// Test GetNextCard endpoint
				server := testutils.SetupCardReviewTestServer(t, testutils.CardReviewServerOptions{
					UserID: userID,
					GetNextCardFn: func(ctx context.Context, uid uuid.UUID) (*domain.Card, error) {
						return nil, tc.error
					},
				})

				// Execute request
				resp, err = testutils.ExecuteGetNextCardRequest(t, server)
				require.NoError(t, err)
			} else {
				// Test SubmitAnswer endpoint
				server := testutils.SetupCardReviewTestServer(t, testutils.CardReviewServerOptions{
					UserID: userID,
					SubmitAnswerFn: func(ctx context.Context, uid uuid.UUID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error) {
						return nil, tc.error
					},
				})

				// Execute request
				resp, err = testutils.ExecuteSubmitAnswerRequest(t, server, uuid.New(), domain.ReviewOutcome("good"))
				require.NoError(t, err)
			}

			// Register cleanup for the response body
			testutils.CleanupResponseBody(t, resp)

			// Verify status code
			assert.Equal(t, tc.expectedStatusCode, resp.StatusCode)

			// Read and parse response body
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var errResp shared.ErrorResponse
			err = json.Unmarshal(body, &errResp)
			require.NoError(t, err)

			// Check that the expected safe message is returned
			assert.Equal(t, tc.expectedMessage, errResp.Error)

			// Check that no sensitive internal error details are leaked
			// The error message should NOT contain the original internal error string
			assert.NotContains(t, errResp.Error, tc.error.Error())

			// Check for specific sensitive patterns
			sensitivePatterns := []string{
				"SQL", "SELECT", "INSERT", "UPDATE", "DELETE", "syntax error",
				"postgres://", "user:", "password",
				"stack trace", "goroutine", "panic",
				"/var/", "/home/", "/etc/", "/usr/",
				"AKIA", "AccessKey", "SecretKey", "AWSAccessKeyId",
				"database error", "connection error", "internal error",
			}

			for _, pattern := range sensitivePatterns {
				if strings.Contains(tc.error.Error(), pattern) {
					assert.NotContains(t, errResp.Error, pattern,
						"Response should not contain sensitive pattern '%s'", pattern)
				}
			}
		})
	}
}

// TestErrorDetailsWithWrappedErrors tests that internal error details don't leak
// when using wrapped errors with fmt.Errorf and %w
func TestErrorDetailsWithWrappedErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	// Create a deeply nested wrapped error
	baseErr := errors.New("database connection string: postgres://user:password@localhost:5432/db")
	wrappedOnce := fmt.Errorf("repository error: %w", baseErr)
	wrappedTwice := fmt.Errorf("service error: %w", wrappedOnce)
	deeplyWrappedError := fmt.Errorf("controller error: %w", wrappedTwice)

	// Setup test server with the deeply wrapped error
	server := testutils.SetupCardReviewTestServer(t, testutils.CardReviewServerOptions{
		UserID: userID,
		GetNextCardFn: func(ctx context.Context, uid uuid.UUID) (*domain.Card, error) {
			return nil, deeplyWrappedError
		},
	})

	// Execute request
	resp, err := testutils.ExecuteGetNextCardRequest(t, server)
	require.NoError(t, err)

	// Register cleanup for the response body
	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Warning: failed to close response body: %v", err)
		}
	})

	// Verify status code
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Read and parse response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var errResp shared.ErrorResponse
	err = json.Unmarshal(body, &errResp)
	require.NoError(t, err)

	// Check that the expected safe message is returned
	assert.Equal(t, "Failed to get next review card", errResp.Error)

	// Check that none of the internal error details are leaked
	assert.NotContains(t, errResp.Error, "database connection string")
	assert.NotContains(t, errResp.Error, "postgres://")
	assert.NotContains(t, errResp.Error, "repository error")
	assert.NotContains(t, errResp.Error, "service error")
	assert.NotContains(t, errResp.Error, "controller error")
}

// TestErrorDetailsDontLeakFromCustomErrors tests that even custom error types
// don't leak their internal details through API responses
func TestErrorDetailsDontLeakFromCustomErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	// Create a custom error instance with sensitive details
	customErr := fmt.Errorf("error at %s:%d: %s (credentials: %s)",
		"/var/www/app/internal/database/queries.go",
		123,
		"failed to process request",
		"user=admin&password=secret123")

	// Use a custom ValidateTokenFn to simulate a custom error from authentication
	validateTokenFn := func(ctx context.Context, token string) (*auth.Claims, error) {
		// Return our custom error with sensitive details
		return nil, customErr
	}

	// Setup test server with the custom error
	server := testutils.SetupCardReviewTestServer(t, testutils.CardReviewServerOptions{
		UserID:          userID,
		ValidateTokenFn: validateTokenFn,
	})

	// Create a request
	req, err := http.NewRequest("GET", server.URL+"/api/cards/next", nil)
	require.NoError(t, err)

	// Add a clearly invalid auth header
	req.Header.Set("Authorization", "Bearer invalid-token")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)

	// Register cleanup for the response body
	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Warning: failed to close response body: %v", err)
		}
	})

	// Verify status code is either 401 (Unauthorized) or 500 (Internal Server Error)
	// The exact status depends on how the error is handled in the middleware chain
	assert.True(t, resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusInternalServerError,
		"Expected status code to be either 401 or 500, got %d", resp.StatusCode)

	// Read and parse response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var errResp shared.ErrorResponse
	err = json.Unmarshal(body, &errResp)
	require.NoError(t, err)

	// Check that the sensitive details are not leaked in the error message
	assert.NotContains(t, errResp.Error, "/var/www/app/internal/database/queries.go")
	assert.NotContains(t, errResp.Error, "user=admin&password=secret123")
	assert.NotContains(t, errResp.Error, "password=secret123")
	assert.NotContains(t, errResp.Error, "queries.go")
	assert.NotContains(t, errResp.Error, "123")
}
