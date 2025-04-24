package testutils

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
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
				t.Errorf("Failed to close response body: %v", err)
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
		resp2, err := ExecuteSubmitAnswerRequest(t, submitServer, userID, cardID, domain.ReviewOutcomeGood)
		require.NoError(t, err)
		defer func() {
			if err := resp2.Body.Close(); err != nil {
				t.Errorf("Failed to close response body: %v", err)
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
				t.Errorf("Failed to close response body: %v", err)
			}
		}()

		// Verify error response
		AssertErrorResponse(t, resp, http.StatusInternalServerError, "Failed to get next review card")
	})
}
