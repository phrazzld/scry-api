//go:build integration

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteCardEndpoint tests the DELETE /cards/{id} endpoint
func TestDeleteCardEndpoint(t *testing.T) {
	// Initialize test database connection
	db := testutils.GetTestDBWithT(t)

	// Run tests in transaction for isolation and automatic cleanup
	testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// tx is already *sql.Tx, no conversion needed

		// Create a test user
		userID := createTestUser(t, tx)

		// Create a test user that doesn't own the card (for forbidden test)
		otherUserID := createTestUser(t, tx)

		// Create a test card owned by userID
		card := createTestCard(t, tx, userID)

		// Get token for authentication
		authToken := getAuthToken(t, userID)
		otherAuthToken := getAuthToken(t, otherUserID)

		// Test cases
		tests := []struct {
			name            string
			cardID          string
			authToken       string
			expectedStatus  int
			expectedMessage string
			verify          func(t *testing.T, tx *sql.Tx, cardID uuid.UUID)
		}{
			{
				name:            "Success",
				cardID:          card.ID.String(),
				authToken:       authToken,
				expectedStatus:  http.StatusNoContent,
				expectedMessage: "",
				verify: func(t *testing.T, tx *sql.Tx, cardID uuid.UUID) {
					// Verify the card was deleted

					// Card should not exist
					_, err := getCardByID(tx, cardID)
					assert.Equal(t, store.ErrCardNotFound, err, "Expected card to be deleted")

					// User card stats should also be deleted (cascade delete)
					stats := getUserCardStats(t, tx, userID, cardID)
					assert.Nil(t, stats, "Expected user card stats to be cascade deleted")
				},
			},
			{
				name:            "Unauthorized - No Token",
				cardID:          card.ID.String(),
				authToken:       "", // No token
				expectedStatus:  http.StatusUnauthorized,
				expectedMessage: "Unauthorized",
			},
			{
				name:            "Forbidden - Not Owner",
				cardID:          card.ID.String(),
				authToken:       otherAuthToken, // Token from different user
				expectedStatus:  http.StatusForbidden,
				expectedMessage: "not owned",
			},
			{
				name:            "Not Found - Card ID Doesn't Exist",
				cardID:          uuid.New().String(), // Random non-existent ID
				authToken:       authToken,
				expectedStatus:  http.StatusNotFound,
				expectedMessage: "not found",
			},
			{
				name:            "Bad Request - Invalid UUID",
				cardID:          "not-a-uuid",
				authToken:       authToken,
				expectedStatus:  http.StatusBadRequest,
				expectedMessage: "Invalid ID",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// tx is already *sql.Tx for WithTx, no type assertion needed

				server := setupCardManagementTestServer(t, tx)
				defer server.Close()

				// Create HTTP request
				url := fmt.Sprintf("%s/api/cards/%s", server.URL, tc.cardID)
				req, err := http.NewRequest(http.MethodDelete, url, nil)
				require.NoError(t, err)

				// Set headers
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
