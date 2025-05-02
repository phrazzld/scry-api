//go:build integration && compatibility

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/testutils/api"
	"github.com/phrazzld/scry-api/internal/testutils/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostponeCardEndpoint tests the POST /cards/{id}/postpone endpoint
func TestPostponeCardEndpoint(t *testing.T) {
	// Initialize test database connection
	dbConn := db.GetTestDBWithT(t)

	// Run tests in transaction for isolation and automatic cleanup
	db.WithTx(t, dbConn, func(t *testing.T, tx *sql.Tx) {
		// tx is already *sql.Tx for WithTx, no type assertion needed

		// Create a test user
		userID := api.CreateTestUser(t, tx)

		// Create a test user that doesn't own the card (for forbidden test)
		otherUserID := api.CreateTestUser(t, tx)

		// Create a test card owned by userID
		card := api.CreateTestCard(t, tx, userID)

		// Get original stats for the card
		originalStats := api.GetUserCardStats(t, tx, userID, card.ID)
		require.NotNil(t, originalStats, "Expected user card stats to exist")

		// Get token for authentication
		authToken := api.GetAuthToken(t, userID)
		otherAuthToken := api.GetAuthToken(t, otherUserID)

		// Test cases
		tests := []struct {
			name            string
			cardID          string
			authToken       string
			requestBody     map[string]interface{}
			expectedStatus  int
			expectedMessage string
			verify          func(t *testing.T, tx *sql.Tx, response map[string]interface{}, originalStats *domain.UserCardStats)
		}{
			{
				name:      "Success",
				cardID:    card.ID.String(),
				authToken: authToken,
				requestBody: map[string]interface{}{
					"days": 7,
				},
				expectedStatus:  http.StatusOK,
				expectedMessage: "",
				verify: func(t *testing.T, tx *sql.Tx, response map[string]interface{}, originalStats *domain.UserCardStats) {
					// Verify response contains expected fields
					assert.Equal(t, userID.String(), response["user_id"])
					assert.Equal(t, card.ID.String(), response["card_id"])

					// Get the updated stats from the database
					updatedStats := api.GetUserCardStats(t, tx, userID, card.ID)
					require.NotNil(t, updatedStats, "Expected user card stats to exist")

					// Calculate expected next review date (original + 7 days)
					expectedNextReviewAt := originalStats.NextReviewAt.AddDate(0, 0, 7)

					// Parse response next_review_at as RFC3339 string
					nextReviewAtStr, ok := response["next_review_at"].(string)
					require.True(t, ok, "Expected next_review_at to be a string")

					nextReviewAt, err := time.Parse(time.RFC3339, nextReviewAtStr)
					require.NoError(t, err, "Failed to parse next_review_at")

					// Verify next_review_at in response matches expected
					assert.WithinDuration(
						t,
						expectedNextReviewAt,
						nextReviewAt,
						time.Second,
						"Expected next_review_at to be %v, but got %v",
						expectedNextReviewAt,
						nextReviewAt,
					)

					// Verify next_review_at in database matches expected
					assert.WithinDuration(
						t,
						expectedNextReviewAt,
						updatedStats.NextReviewAt,
						time.Second,
						"Expected next_review_at to be %v, but got %v",
						expectedNextReviewAt,
						updatedStats.NextReviewAt,
					)

					// Verify updated_at is later than the original value
					assert.True(t, updatedStats.UpdatedAt.After(originalStats.UpdatedAt),
						"Expected updated_at to be later than original, but got %v vs %v",
						updatedStats.UpdatedAt, originalStats.UpdatedAt)
				},
			},
			{
				name:      "Bad Request - Invalid Days Parameter",
				cardID:    card.ID.String(),
				authToken: authToken,
				requestBody: map[string]interface{}{
					"days": 0, // Should be >= 1
				},
				expectedStatus:  http.StatusBadRequest,
				expectedMessage: "Validation",
			},
			{
				name:        "Bad Request - Missing Days Parameter",
				cardID:      card.ID.String(),
				authToken:   authToken,
				requestBody: map[string]interface{}{
					// Missing required "days" field
				},
				expectedStatus:  http.StatusBadRequest,
				expectedMessage: "Validation",
			},
			{
				name:      "Unauthorized - No Token",
				cardID:    card.ID.String(),
				authToken: "", // No token
				requestBody: map[string]interface{}{
					"days": 7,
				},
				expectedStatus:  http.StatusUnauthorized,
				expectedMessage: "Unauthorized",
			},
			{
				name:      "Forbidden - Not Owner",
				cardID:    card.ID.String(),
				authToken: otherAuthToken, // Token from different user
				requestBody: map[string]interface{}{
					"days": 7,
				},
				expectedStatus:  http.StatusForbidden,
				expectedMessage: "not owned",
			},
			{
				name:      "Not Found - Card ID Doesn't Exist",
				cardID:    uuid.New().String(), // Random non-existent ID
				authToken: authToken,
				requestBody: map[string]interface{}{
					"days": 7,
				},
				expectedStatus:  http.StatusNotFound,
				expectedMessage: "not found",
			},
			{
				name:      "Bad Request - Invalid UUID",
				cardID:    "not-a-uuid",
				authToken: authToken,
				requestBody: map[string]interface{}{
					"days": 7,
				},
				expectedStatus:  http.StatusBadRequest,
				expectedMessage: "Invalid ID",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// tx is already *sql.Tx for WithTx, no type assertion needed

				server := api.SetupCardManagementTestServer(t, tx)
				defer server.Close()

				// Create request body
				requestBody, err := json.Marshal(tc.requestBody)
				require.NoError(t, err)

				// Create HTTP request
				url := fmt.Sprintf("%s/api/cards/%s/postpone", server.URL, tc.cardID)
				req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBody))
				require.NoError(t, err)

				// Set headers
				req.Header.Set("Content-Type", "application/json")
				if tc.authToken != "" {
					req.Header.Set("Authorization", "Bearer "+tc.authToken)
				}

				// Send request
				client := &http.Client{}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer func() {
					err := resp.Body.Close()
					if err != nil {
						t.Logf("Warning: failed to close response body: %v", err)
					}
				}()

				// Check status code
				assert.Equal(t, tc.expectedStatus, resp.StatusCode)

				// For successful requests (200 OK)
				if tc.expectedStatus == http.StatusOK {
					// Parse response body
					var response map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&response)
					require.NoError(t, err)

					// Run additional verification if provided
					if tc.verify != nil {
						tc.verify(t, tx, response, originalStats)
					}
					return
				}

				// For error responses, verify error message
				var errorResp map[string]string
				err = json.NewDecoder(resp.Body).Decode(&errorResp)
				require.NoError(t, err)

				assert.Contains(t, errorResp["error"], tc.expectedMessage)
			})
		}
	})
}
