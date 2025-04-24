package testutils

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCardReviewTestHelpers(t *testing.T) {
	// Test CreateCardForAPITest with default values
	t.Run("CreateCardForAPITest defaults", func(t *testing.T) {
		card := CreateCardForAPITest(t)
		assert.NotEqual(t, uuid.Nil, card.ID)
		assert.NotEqual(t, uuid.Nil, card.UserID)
		assert.NotEqual(t, uuid.Nil, card.MemoID)
		assert.NotEmpty(t, card.Content)
		assert.False(t, card.CreatedAt.IsZero())
		assert.False(t, card.UpdatedAt.IsZero())
	})

	// Test CreateCardForAPITest with options
	t.Run("CreateCardForAPITest with options", func(t *testing.T) {
		userID := uuid.New()
		cardID := uuid.New()
		memoID := uuid.New()
		content := map[string]interface{}{
			"front": "Custom question",
			"back":  "Custom answer",
		}

		card := CreateCardForAPITest(t,
			WithCardID(cardID),
			WithCardUserID(userID),
			WithCardMemoID(memoID),
			WithCardContent(content),
		)

		assert.Equal(t, cardID, card.ID)
		assert.Equal(t, userID, card.UserID)
		assert.Equal(t, memoID, card.MemoID)

		// Verify content
		var decodedContent map[string]interface{}
		err := json.Unmarshal(card.Content, &decodedContent)
		require.NoError(t, err)
		assert.Equal(t, "Custom question", decodedContent["front"])
		assert.Equal(t, "Custom answer", decodedContent["back"])
	})

	// Test CreateStatsForAPITest with default values
	t.Run("CreateStatsForAPITest defaults", func(t *testing.T) {
		stats := CreateStatsForAPITest(t)
		assert.NotEqual(t, uuid.Nil, stats.UserID)
		assert.NotEqual(t, uuid.Nil, stats.CardID)
		assert.Equal(t, 1, stats.Interval)
		assert.Equal(t, 2.5, stats.EaseFactor)
		assert.Equal(t, 1, stats.ConsecutiveCorrect)
		assert.Equal(t, 1, stats.ReviewCount)
		assert.False(t, stats.LastReviewedAt.IsZero())
		assert.False(t, stats.NextReviewAt.IsZero())
		assert.False(t, stats.CreatedAt.IsZero())
		assert.False(t, stats.UpdatedAt.IsZero())
	})

	// Test CreateStatsForAPITest with options
	t.Run("CreateStatsForAPITest with options", func(t *testing.T) {
		userID := uuid.New()
		cardID := uuid.New()

		stats := CreateStatsForAPITest(t,
			WithStatsUserID(userID),
			WithStatsCardID(cardID),
			WithStatsInterval(2),
			WithStatsEaseFactor(2.1),
			WithStatsConsecutiveCorrect(3),
			WithStatsReviewCount(5),
		)

		assert.Equal(t, userID, stats.UserID)
		assert.Equal(t, cardID, stats.CardID)
		assert.Equal(t, 2, stats.Interval)
		assert.Equal(t, 2.1, stats.EaseFactor)
		assert.Equal(t, 3, stats.ConsecutiveCorrect)
		assert.Equal(t, 5, stats.ReviewCount)
	})

	// Test server setup and request helpers
	t.Run("Basic API workflow", func(t *testing.T) {
		// Setup test data
		userID := uuid.New()
		cardID := uuid.New()

		// Create test card and stats
		card := CreateCardForAPITest(t,
			WithCardID(cardID),
			WithCardUserID(userID),
		)

		stats := CreateStatsForAPITest(t,
			WithStatsUserID(userID),
			WithStatsCardID(cardID),
		)

		// Setup test server for GetNextCard
		server := SetupCardReviewTestServer(t, CardReviewServerOptions{
			UserID:   userID,
			NextCard: card,
		})
		defer server.Close()

		// Test GetNextCard
		resp, err := ExecuteGetNextCardRequest(t, server, userID)
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		}()

		// Verify response
		AssertCardResponse(t, resp, card)

		// Create new server for SubmitAnswer
		submitServer := SetupCardReviewTestServer(t, CardReviewServerOptions{
			UserID:       userID,
			UpdatedStats: stats,
		})
		defer submitServer.Close()

		// Test SubmitAnswer
		resp2, err := ExecuteSubmitAnswerRequest(
			t,
			submitServer,
			userID,
			cardID,
			domain.ReviewOutcomeGood,
		)
		require.NoError(t, err)
		defer func() {
			if err := resp2.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		}()

		// Verify response
		AssertStatsResponse(t, resp2, stats)
	})

	// Test error response helper
	t.Run("Error response handling", func(t *testing.T) {
		userID := uuid.New()

		// Setup test server with error
		testError := errors.New("test error")
		server := SetupCardReviewTestServer(t, CardReviewServerOptions{
			UserID: userID,
			Error:  testError,
		})
		defer server.Close()

		// Test request
		resp, err := ExecuteGetNextCardRequest(t, server, userID)
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		}()

		// Verify error response
		AssertErrorResponse(
			t,
			resp,
			http.StatusInternalServerError,
			"Failed to get next review card",
		)
	})
}

// TestCardReviewServerOptions tests the simplified CardReviewServerOptions
// structure and its usage in SetupCardReviewTestServer.
func TestCardReviewServerOptions(t *testing.T) {
	t.Run("Priority of options", func(t *testing.T) {
		userID := uuid.New()
		cardID := uuid.New()

		// Create test data
		card := CreateCardForAPITest(t, WithCardUserID(userID), WithCardID(cardID))
		stats := CreateStatsForAPITest(t, WithStatsUserID(userID), WithStatsCardID(cardID))
		testError := errors.New("test error")

		// Define custom functions that will take precedence
		customGetNextCardFn := func(ctx context.Context, uid uuid.UUID) (*domain.Card, error) {
			// This should override the NextCard field
			return nil, testError
		}

		customSubmitAnswerFn := func(ctx context.Context, uid uuid.UUID, cid uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error) {
			// This should override the UpdatedStats field
			return nil, testError
		}

		// Test server with both data fields and custom functions
		server := SetupCardReviewTestServer(t, CardReviewServerOptions{
			UserID:         userID,
			NextCard:       card,                 // This would normally return a success
			UpdatedStats:   stats,                // This would normally return a success
			GetNextCardFn:  customGetNextCardFn,  // This should override and return error
			SubmitAnswerFn: customSubmitAnswerFn, // This should override and return error
		})
		defer server.Close()

		// Test GetNextCard - should use the custom function
		resp, err := ExecuteGetNextCardRequest(t, server, userID)
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		}()

		// Should get error from custom function
		AssertErrorResponse(
			t,
			resp,
			http.StatusInternalServerError,
			"Failed to get next review card",
		)

		// Test SubmitAnswer - should use the custom function
		resp2, err := ExecuteSubmitAnswerRequest(
			t,
			server,
			userID,
			cardID,
			domain.ReviewOutcomeGood,
		)
		require.NoError(t, err)
		defer func() {
			if err := resp2.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		}()

		// Should get error from custom function
		AssertErrorResponse(t, resp2, http.StatusInternalServerError, "Failed to submit answer")
	})

	t.Run("Convenience constructors", func(t *testing.T) {
		userID := uuid.New()
		cardID := uuid.New()

		// Create test data
		card := CreateCardForAPITest(t, WithCardUserID(userID), WithCardID(cardID))
		stats := CreateStatsForAPITest(t, WithStatsUserID(userID), WithStatsCardID(cardID))

		// Test the convenience constructor for NextCard
		t.Run("SetupCardReviewTestServerWithNextCard", func(t *testing.T) {
			server := SetupCardReviewTestServerWithNextCard(t, userID, card)
			defer server.Close()

			resp, err := ExecuteGetNextCardRequest(t, server, userID)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Failed to close response body: %v", err)
				}
			}()

			AssertCardResponse(t, resp, card)
		})

		// Test the convenience constructor for UpdatedStats
		t.Run("SetupCardReviewTestServerWithUpdatedStats", func(t *testing.T) {
			server := SetupCardReviewTestServerWithUpdatedStats(t, userID, stats)
			defer server.Close()

			resp, err := ExecuteSubmitAnswerRequest(
				t,
				server,
				userID,
				cardID,
				domain.ReviewOutcomeGood,
			)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Failed to close response body: %v", err)
				}
			}()

			AssertStatsResponse(t, resp, stats)
		})

		// Test the convenience constructor for Error
		t.Run("SetupCardReviewTestServerWithError", func(t *testing.T) {
			testError := card_review.ErrCardNotFound
			server := SetupCardReviewTestServerWithError(t, userID, testError)
			defer server.Close()

			resp, err := ExecuteGetNextCardRequest(t, server, userID)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Failed to close response body: %v", err)
				}
			}()

			AssertErrorResponse(t, resp, http.StatusNotFound, "Card not found")
		})

		// Test the convenience constructor for AuthError
		t.Run("SetupCardReviewTestServerWithAuthError", func(t *testing.T) {
			server := SetupCardReviewTestServerWithAuthError(t, userID, auth.ErrInvalidToken)
			defer server.Close()

			resp, err := ExecuteGetNextCardRequest(t, server, userID)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Failed to close response body: %v", err)
				}
			}()

			AssertErrorResponse(t, resp, http.StatusUnauthorized, "Invalid token")
		})
	})
}
