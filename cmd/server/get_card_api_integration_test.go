//go:build integration

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetNextReviewCardIntegration tests the GET /cards/next endpoint with real dependencies
func TestGetNextReviewCardIntegration(t *testing.T) {
	// Get a test database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create stores with the transaction
		userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for test speed
		logger := slog.Default()
		cardStore := postgres.NewPostgresCardStore(tx, logger)
		memoStore := postgres.NewPostgresMemoStore(tx, logger)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, logger)

		// Create test user
		testUser, err := domain.NewUser("get-next-review-card-integration-test@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(t, userStore.Create(ctx, testUser), "Failed to save test user")

		// Create test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Get Next Review Card Integration Test")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(t, memoStore.Create(ctx, testMemo), "Failed to save test memo")

		// Create test card with content
		cardContent := map[string]interface{}{
			"front": "What is the capital of France?",
			"back":  "Paris",
		}
		contentBytes, err := json.Marshal(cardContent)
		require.NoError(t, err, "Failed to marshal card content")

		card, err := domain.NewCard(testUser.ID, testMemo.ID, contentBytes)
		require.NoError(t, err, "Failed to create test card")
		require.NoError(t, cardStore.CreateMultiple(ctx, []*domain.Card{card}), "Failed to save test card")

		// Create user card stats to make the card eligible for review
		pastTime := time.Now().UTC().Add(-24 * time.Hour) // Set review time in the past
		stats, err := domain.NewUserCardStats(testUser.ID, card.ID)
		require.NoError(t, err, "Failed to create test stats")
		stats.NextReviewAt = pastTime // Make card due for review
		require.NoError(t, statsStore.Create(ctx, stats), "Failed to save test stats")

		// Create SRS service
		srsService, err := srs.NewDefaultService()
		require.NoError(t, err, "Failed to create SRS service")

		// Create repository adapters for service layer
		dbConn := cardStore.DB()
		cardRepo := service.NewCardRepositoryAdapter(cardStore, dbConn)
		statsRepo := service.NewStatsRepositoryAdapter(statsStore)

		// Create services using the adapters
		cardService, err := service.NewCardService(cardRepo, statsRepo, srsService, logger)
		require.NoError(t, err, "Failed to create card service")

		cardReviewService, err := card_review.NewCardReviewService(cardStore, statsStore, srsService, logger)
		require.NoError(t, err, "Failed to create card review service")

		// Create JWT service for auth
		jwtService, err := testutils.CreateTestJWTService()
		require.NoError(t, err, "Failed to create JWT service")

		// Set up tests with different scenarios
		tests := []struct {
			name           string
			setupAuth      func(*http.Request)
			expectedStatus int
			verifyResponse func(*testing.T, *http.Response)
		}{
			{
				name: "Success",
				setupAuth: func(req *http.Request) {
					// Add valid auth token for test user
					authHeader, err := testutils.GenerateAuthHeader(testUser.ID)
					require.NoError(t, err, "Failed to generate auth header")
					req.Header.Set("Authorization", authHeader)
				},
				expectedStatus: http.StatusOK,
				verifyResponse: func(t *testing.T, resp *http.Response) {
					// Verify response contains the expected card
					assert.Equal(t, http.StatusOK, resp.StatusCode)

					// Read and parse response body
					body, err := io.ReadAll(resp.Body)
					require.NoError(t, err, "Failed to read response body")

					var cardResp api.CardResponse
					err = json.Unmarshal(body, &cardResp)
					require.NoError(t, err, "Failed to unmarshal response")

					// Verify fields
					assert.Equal(t, card.ID.String(), cardResp.ID, "Card ID should match")
					assert.Equal(t, card.UserID.String(), cardResp.UserID, "User ID should match")
					assert.Equal(t, card.MemoID.String(), cardResp.MemoID, "Memo ID should match")

					// Verify content
					content, ok := cardResp.Content.(map[string]interface{})
					assert.True(t, ok, "Content should be a map")
					assert.Equal(t, "What is the capital of France?", content["front"], "Front content should match")
					assert.Equal(t, "Paris", content["back"], "Back content should match")
				},
			},
			{
				name:           "Unauthorized - No Token",
				setupAuth:      func(req *http.Request) {}, // No auth token
				expectedStatus: http.StatusUnauthorized,
				verifyResponse: func(t *testing.T, resp *http.Response) {
					assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				},
			},
			{
				name: "Unauthorized - Invalid Token",
				setupAuth: func(req *http.Request) {
					req.Header.Set("Authorization", "Bearer invalid-token")
				},
				expectedStatus: http.StatusUnauthorized,
				verifyResponse: func(t *testing.T, resp *http.Response) {
					assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Create router with standard middleware
				router := chi.NewRouter()
				router.Use(chimiddleware.RequestID)
				router.Use(chimiddleware.RealIP)
				router.Use(chimiddleware.Recoverer)

				// Create auth middleware
				authMiddleware := middleware.NewAuthMiddleware(jwtService)

				// Create logger
				logger := slog.Default()

				// Create card handler with real services
				cardHandler := api.NewCardHandler(cardReviewService, cardService, logger)

				// Set up API routes
				router.Route("/api", func(r chi.Router) {
					r.Group(func(r chi.Router) {
						r.Use(authMiddleware.Authenticate)
						r.Get("/cards/next", cardHandler.GetNextReviewCard)
					})
				})

				// Create server
				server := httptest.NewServer(router)
				defer server.Close()

				// Create request
				req, err := http.NewRequest("GET", server.URL+"/api/cards/next", nil)
				require.NoError(t, err, "Failed to create request")

				// Set up auth
				tc.setupAuth(req)

				// Execute request
				client := &http.Client{}
				resp, err := client.Do(req)
				require.NoError(t, err, "Failed to execute request")
				defer resp.Body.Close()

				// Verify response
				assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code should match expected")
				tc.verifyResponse(t, resp)
			})
		}

		// Test case: No cards due
		t.Run("No Cards Due", func(t *testing.T) {
			// Create a separate user with no due cards
			userWithNoDueCards, err := domain.NewUser("no-due-cards@example.com", "password123")
			require.NoError(t, err, "Failed to create test user")
			require.NoError(t, userStore.Create(ctx, userWithNoDueCards), "Failed to save test user")

			// Create router with standard middleware
			router := chi.NewRouter()
			router.Use(chimiddleware.RequestID)
			router.Use(chimiddleware.RealIP)
			router.Use(chimiddleware.Recoverer)

			// Create auth middleware
			authMiddleware := middleware.NewAuthMiddleware(jwtService)

			// Create logger
			logger := slog.Default()

			// Create card handler with real services
			cardHandler := api.NewCardHandler(cardReviewService, cardService, logger)

			// Set up API routes
			router.Route("/api", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(authMiddleware.Authenticate)
					r.Get("/cards/next", cardHandler.GetNextReviewCard)
				})
			})

			// Create server
			server := httptest.NewServer(router)
			defer server.Close()

			// Create request
			req, err := http.NewRequest("GET", server.URL+"/api/cards/next", nil)
			require.NoError(t, err, "Failed to create request")

			// Add auth header for user with no due cards
			authHeader, err := testutils.GenerateAuthHeader(userWithNoDueCards.ID)
			require.NoError(t, err, "Failed to generate auth header")
			req.Header.Set("Authorization", authHeader)

			// Execute request
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err, "Failed to execute request")
			defer resp.Body.Close()

			// Verify response
			assert.Equal(t, http.StatusNoContent, resp.StatusCode, "Should return 204 No Content when no cards are due")
		})
	})
}
