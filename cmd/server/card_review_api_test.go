//go:build test_without_external_deps
// +build test_without_external_deps

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	authmiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetNextReviewCardAPI tests the GET /cards/next endpoint with various scenarios
func TestGetNextReviewCardAPI(t *testing.T) {
	// Test user
	userID := uuid.New()

	// Create sample card for testing
	memoID := uuid.New()
	cardID := uuid.New()
	now := time.Now().UTC()

	// Create sample content for the test card
	cardContent := map[string]interface{}{
		"front": "What is the capital of France?",
		"back":  "Paris",
	}
	contentBytes, err := json.Marshal(cardContent)
	require.NoError(t, err)

	card := &domain.Card{
		ID:        cardID,
		UserID:    userID,
		MemoID:    memoID,
		Content:   contentBytes,
		CreatedAt: now.Add(-24 * time.Hour),
		UpdatedAt: now.Add(-24 * time.Hour),
	}

	// Test cases
	tests := []struct {
		name           string
		mockSetup      func() (*mocks.MockCardReviewService, *mocks.MockJWTService)
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name: "Success - Card Found",
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Set up card review service mock
				cardReviewMock := mocks.NewMockCardReviewService(
					mocks.WithNextCard(card),
				)

				// Set up JWT service mock with valid claims
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return &auth.Claims{
							UserID: userID,
						}, nil
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response api.CardResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				// Check card fields
				assert.Equal(t, cardID.String(), response.ID)
				assert.Equal(t, userID.String(), response.UserID)
				assert.Equal(t, memoID.String(), response.MemoID)

				// Check content
				content, ok := response.Content.(map[string]interface{})
				assert.True(t, ok, "Content should be a map")
				assert.Equal(t, "What is the capital of France?", content["front"])
				assert.Equal(t, "Paris", content["back"])
			},
		},
		{
			name: "No Cards Due",
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Set up card review service mock to return no cards
				cardReviewMock := mocks.NewMockCardReviewServiceWithNoCardsDue()

				// Set up JWT service mock with valid claims
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return &auth.Claims{
							UserID: userID,
						}, nil
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusNoContent,
			validateBody: func(t *testing.T, body []byte) {
				assert.Empty(t, body, "Response body should be empty for 204 No Content")
			},
		},
		{
			name: "Unauthorized - No Valid JWT",
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Card service won't be called, so just create a default mock
				cardReviewMock := mocks.NewMockCardReviewService()

				// Set up JWT service to simulate auth failure
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return nil, auth.ErrInvalidToken
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, body []byte) {
				var errResp shared.ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)

				assert.Contains(t, errResp.Error, "Invalid token")
			},
		},
		{
			name: "Server Error",
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Set up card review service to return a server error
				cardReviewMock := mocks.NewMockCardReviewService(
					mocks.WithError(assert.AnError), // Use a generic error
				)

				// Set up JWT service mock with valid claims
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return &auth.Claims{
							UserID: userID,
						}, nil
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body []byte) {
				var errResp shared.ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)

				assert.Contains(t, errResp.Error, "Failed to get next review card")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up mocks for the test case
			cardReviewMock, jwtMock := tc.mockSetup()

			// Set up router with auth middleware
			router := chi.NewRouter()
			router.Use(chimiddleware.RequestID)
			router.Use(chimiddleware.RealIP)
			router.Use(chimiddleware.Recoverer)

			// Create auth middleware
			authMiddleware := authmiddleware.NewAuthMiddleware(jwtMock)

			// Create card handler
			cardHandler := api.NewCardHandler(cardReviewMock, nil) // nil logger will use default

			// Set up routes
			router.Route("/api", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(authMiddleware.Authenticate)
					r.Get("/cards/next", cardHandler.GetNextReviewCard)
				})
			})

			// Create a test server
			server := httptest.NewServer(router)
			defer server.Close()

			// Create client and request
			client := &http.Client{}
			req, err := http.NewRequest("GET", server.URL+"/api/cards/next", nil)
			require.NoError(t, err)

			// Add auth header with a fake token (the mock doesn't check the actual token value)
			req.Header.Set("Authorization", "Bearer fake-token")

			// Execute request
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Errorf("Failed to close response body: %v", err)
				}
			}()

			// Check status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			// Read and validate response body if needed
			if tc.validateBody != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				tc.validateBody(t, body)
			}

			// Check call counts
			if tc.expectedStatus != http.StatusUnauthorized {
				// If auth passed, the service should have been called
				assert.Equal(t, 1, cardReviewMock.GetNextCardCalls.Count,
					"GetNextCard should have been called exactly once")
			} else {
				// If auth failed, the service should NOT have been called
				assert.Equal(t, 0, cardReviewMock.GetNextCardCalls.Count,
					"GetNextCard should not have been called on auth failure")
			}
		})
	}
}

// TestSubmitAnswerAPI tests the POST /cards/{id}/answer endpoint with various scenarios
func TestSubmitAnswerAPI(t *testing.T) {
	// Test user and card
	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now().UTC()

	// Create sample stats for testing
	sampleStats := &domain.UserCardStats{
		UserID:             userID,
		CardID:             cardID,
		Interval:           1,
		EaseFactor:         2.5,
		ConsecutiveCorrect: 1,
		LastReviewedAt:     now,
		NextReviewAt:       now.Add(24 * time.Hour),
		ReviewCount:        1,
		CreatedAt:          now.Add(-24 * time.Hour),
		UpdatedAt:          now,
	}

	// Note: These are the valid outcome values that we use in our tests
	// domain.ReviewOutcomeAgain
	// domain.ReviewOutcomeHard
	// domain.ReviewOutcomeGood
	// domain.ReviewOutcomeEasy

	// Test cases
	tests := []struct {
		name           string
		cardIDInPath   string
		requestBody    map[string]string
		mockSetup      func() (*mocks.MockCardReviewService, *mocks.MockJWTService)
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:         "Success",
			cardIDInPath: cardID.String(),
			requestBody:  map[string]string{"outcome": string(domain.ReviewOutcomeGood)},
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Set up card review service mock
				cardReviewMock := mocks.NewMockCardReviewService(
					mocks.WithUpdatedStats(sampleStats),
				)

				// Set up JWT service mock with valid claims
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return &auth.Claims{
							UserID: userID,
						}, nil
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response api.UserCardStatsResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				// Check stats fields
				assert.Equal(t, userID.String(), response.UserID)
				assert.Equal(t, cardID.String(), response.CardID)
				assert.Equal(t, 1, response.Interval)
				assert.Equal(t, 2.5, response.EaseFactor)
				assert.Equal(t, 1, response.ConsecutiveCorrect)
				assert.Equal(t, 1, response.ReviewCount)
				assert.Equal(t, now, response.LastReviewedAt)
				assert.Equal(t, now.Add(24*time.Hour), response.NextReviewAt)
			},
		},
		{
			name:         "Card Not Found",
			cardIDInPath: uuid.New().String(), // Different card ID
			requestBody:  map[string]string{"outcome": string(domain.ReviewOutcomeGood)},
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Set up card review service to return card not found error
				cardReviewMock := mocks.NewMockCardReviewServiceWithCardNotFound()

				// Set up JWT service mock with valid claims
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return &auth.Claims{
							UserID: userID,
						}, nil
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body []byte) {
				var errResp shared.ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)

				assert.Contains(t, errResp.Error, "Card not found")
			},
		},
		{
			name:         "Card Not Owned",
			cardIDInPath: cardID.String(),
			requestBody:  map[string]string{"outcome": string(domain.ReviewOutcomeGood)},
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Set up card review service to return card not owned error
				cardReviewMock := mocks.NewMockCardReviewServiceWithCardNotOwned()

				// Set up JWT service mock with valid claims
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return &auth.Claims{
							UserID: userID,
						}, nil
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusForbidden,
			validateBody: func(t *testing.T, body []byte) {
				var errResp shared.ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)

				assert.Contains(t, errResp.Error, "You do not own this card")
			},
		},
		{
			name:         "Invalid Answer",
			cardIDInPath: cardID.String(),
			requestBody:  map[string]string{"outcome": string(domain.ReviewOutcomeGood)},
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Set up card review service to return invalid answer error
				cardReviewMock := mocks.NewMockCardReviewServiceWithInvalidAnswer()

				// Set up JWT service mock with valid claims
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return &auth.Claims{
							UserID: userID,
						}, nil
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var errResp shared.ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)

				assert.Contains(t, errResp.Error, "Invalid answer")
			},
		},
		{
			name:         "Invalid Card ID Format",
			cardIDInPath: "not-a-uuid",
			requestBody:  map[string]string{"outcome": string(domain.ReviewOutcomeGood)},
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Service won't be called, so default mock is fine
				cardReviewMock := mocks.NewMockCardReviewService()

				// Set up JWT service mock with valid claims
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return &auth.Claims{
							UserID: userID,
						}, nil
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var errResp shared.ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)

				assert.Contains(t, errResp.Error, "Invalid card ID format")
			},
		},
		{
			name:         "Missing Outcome Field",
			cardIDInPath: cardID.String(),
			requestBody:  map[string]string{}, // Empty request
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Service won't be called, so default mock is fine
				cardReviewMock := mocks.NewMockCardReviewService()

				// Set up JWT service mock with valid claims
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return &auth.Claims{
							UserID: userID,
						}, nil
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var errResp shared.ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)

				assert.Contains(t, errResp.Error, "Validation error")
			},
		},
		{
			name:         "Invalid Outcome Value",
			cardIDInPath: cardID.String(),
			requestBody:  map[string]string{"outcome": "invalid-outcome"},
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Service won't be called, so default mock is fine
				cardReviewMock := mocks.NewMockCardReviewService()

				// Set up JWT service mock with valid claims
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return &auth.Claims{
							UserID: userID,
						}, nil
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var errResp shared.ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)

				assert.Contains(t, errResp.Error, "Validation error")
			},
		},
		{
			name:         "Unauthorized - No Valid JWT",
			cardIDInPath: cardID.String(),
			requestBody:  map[string]string{"outcome": string(domain.ReviewOutcomeGood)},
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Service won't be called, so default mock is fine
				cardReviewMock := mocks.NewMockCardReviewService()

				// Set up JWT service to simulate auth failure
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return nil, auth.ErrInvalidToken
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, body []byte) {
				var errResp shared.ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)

				assert.Contains(t, errResp.Error, "Invalid token")
			},
		},
		{
			name:         "Server Error",
			cardIDInPath: cardID.String(),
			requestBody:  map[string]string{"outcome": string(domain.ReviewOutcomeGood)},
			mockSetup: func() (*mocks.MockCardReviewService, *mocks.MockJWTService) {
				// Set up card review service to return a server error
				cardReviewMock := mocks.NewMockCardReviewService(
					mocks.WithError(errors.New("database error")),
				)

				// Set up JWT service mock with valid claims
				jwtMock := &mocks.MockJWTService{
					ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
						return &auth.Claims{
							UserID: userID,
						}, nil
					},
				}

				return cardReviewMock, jwtMock
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body []byte) {
				var errResp shared.ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)

				assert.Contains(t, errResp.Error, "Failed to submit answer")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up mocks for the test case
			cardReviewMock, jwtMock := tc.mockSetup()

			// Set up router with auth middleware
			router := chi.NewRouter()
			router.Use(chimiddleware.RequestID)
			router.Use(chimiddleware.RealIP)
			router.Use(chimiddleware.Recoverer)

			// Create auth middleware
			authMiddleware := authmiddleware.NewAuthMiddleware(jwtMock)

			// Create card handler
			cardHandler := api.NewCardHandler(cardReviewMock, nil) // nil logger will use default

			// Set up routes
			router.Route("/api", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(authMiddleware.Authenticate)
					r.Post("/cards/{id}/answer", cardHandler.SubmitAnswer)
				})
			})

			// Create a test server
			server := httptest.NewServer(router)
			defer server.Close()

			// Create request body
			var requestBody []byte
			if tc.requestBody != nil {
				var err error
				requestBody, err = json.Marshal(tc.requestBody)
				require.NoError(t, err)
			}

			// Create client and request
			client := &http.Client{}
			req, err := http.NewRequest(
				"POST",
				server.URL+"/api/cards/"+tc.cardIDInPath+"/answer",
				bytes.NewBuffer(requestBody),
			)
			require.NoError(t, err)

			// Set headers
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer fake-token")

			// Execute request
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Errorf("Failed to close response body: %v", err)
				}
			}()

			// Check status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			// Read and validate response body if needed
			if tc.validateBody != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				tc.validateBody(t, body)
			}

			// Check call counts only for cases where we expect the service to be called
			// The service should not be called for:
			// - Authorization failures
			// - Invalid card ID format
			// - Invalid/missing request body or invalid outcome
			shouldCallService := tc.expectedStatus != http.StatusUnauthorized &&
				tc.cardIDInPath != "not-a-uuid" &&
				len(tc.requestBody) > 0 &&
				(tc.name != "Invalid Outcome Value")

			if shouldCallService {
				// If all preconditions pass, the service should have been called
				assert.Equal(t, 1, cardReviewMock.SubmitAnswerCalls.Count,
					"SubmitAnswer should have been called exactly once")
			} else {
				// Otherwise, the service should NOT have been called
				assert.Equal(t, 0, cardReviewMock.SubmitAnswerCalls.Count,
					"SubmitAnswer should not have been called")
			}
		})
	}
}
