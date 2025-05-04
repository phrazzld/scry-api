//go:build integration_skip

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/phrazzld/scry-api/internal/testutils/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEditCardEndpoint tests the PUT /cards/{id} endpoint
func TestEditCardEndpoint(t *testing.T) {
	// Initialize test database connection
	db := testdb.GetTestDBWithT(t)

	// Run tests in transaction for isolation and automatic cleanup
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// tx is already *sql.Tx for WithTx, no type assertion needed

		// Create a test user
		userID := api.CreateTestUser(t, tx)

		// Create a test user that doesn't own the card (for forbidden test)
		otherUserID := api.CreateTestUser(t, tx)

		// Create a test card owned by userID
		card := api.CreateTestCard(t, tx, userID)

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
			verify          func(t *testing.T, tx *sql.Tx, cardID uuid.UUID)
		}{
			{
				name:      "Success",
				cardID:    card.ID.String(),
				authToken: authToken,
				requestBody: map[string]interface{}{
					"content": map[string]interface{}{
						"front": "Updated question",
						"back":  "Updated answer",
					},
				},
				expectedStatus:  http.StatusNoContent,
				expectedMessage: "",
				verify: func(t *testing.T, tx *sql.Tx, cardID uuid.UUID) {
					// Verify the card content and updated_at timestamp were updated

					updatedCard, err := api.GetCardByID(tx, cardID)
					require.NoError(t, err)

					// Decode content to verify it was updated
					var content map[string]interface{}
					err = json.Unmarshal(updatedCard.Content, &content)
					require.NoError(t, err)

					assert.Equal(t, "Updated question", content["front"])
					assert.Equal(t, "Updated answer", content["back"])

					// Verify updated_at is later than the original value
					assert.True(t, updatedCard.UpdatedAt.After(card.UpdatedAt),
						"Expected updated_at to be later than original, but got %v vs %v",
						updatedCard.UpdatedAt, card.UpdatedAt)
				},
			},
			{
				name:      "Unauthorized - No Token",
				cardID:    card.ID.String(),
				authToken: "", // No token
				requestBody: map[string]interface{}{
					"content": map[string]interface{}{
						"front": "Unauthorized update attempt",
						"back":  "Should fail",
					},
				},
				expectedStatus:  http.StatusUnauthorized,
				expectedMessage: "Unauthorized",
			},
			{
				name:      "Forbidden - Not Owner",
				cardID:    card.ID.String(),
				authToken: otherAuthToken, // Token from different user
				requestBody: map[string]interface{}{
					"content": map[string]interface{}{
						"front": "Forbidden update attempt",
						"back":  "Should fail",
					},
				},
				expectedStatus:  http.StatusForbidden,
				expectedMessage: "not owned",
			},
			{
				name:      "Not Found - Card ID Doesn't Exist",
				cardID:    uuid.New().String(), // Random non-existent ID
				authToken: authToken,
				requestBody: map[string]interface{}{
					"content": map[string]interface{}{
						"front": "Non-existent card",
						"back":  "Should fail",
					},
				},
				expectedStatus:  http.StatusNotFound,
				expectedMessage: "not found",
			},
			{
				name:        "Bad Request - Invalid JSON",
				cardID:      card.ID.String(),
				authToken:   authToken,
				requestBody: map[string]interface{}{
					// Missing required "content" field
				},
				expectedStatus:  http.StatusBadRequest,
				expectedMessage: "Validation",
			},
			{
				name:            "Bad Request - Invalid UUID",
				cardID:          "not-a-uuid",
				authToken:       authToken,
				requestBody:     map[string]interface{}{},
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
				var requestBody []byte
				var err error

				// Create proper request body for valid tests
				if len(tc.requestBody) > 0 {
					requestBody, err = json.Marshal(tc.requestBody)
					require.NoError(t, err)
				} else {
					// For invalid JSON test
					requestBody = []byte("{invalid-json")
				}

				// Create HTTP request
				url := fmt.Sprintf("%s/api/cards/%s", server.URL, tc.cardID)
				req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(requestBody))
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

				// For successful requests (204 No Content)
				if tc.expectedStatus == http.StatusNoContent {
					// No content to verify in response body
					assert.Equal(t, 0, resp.ContentLength)

					// Run additional verification if provided
					if tc.verify != nil {
						cardID, err := uuid.Parse(tc.cardID)
						require.NoError(t, err)
						tc.verify(t, tx, cardID)
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
