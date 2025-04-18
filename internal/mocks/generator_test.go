package mocks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/generation"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestMockGenerator(t *testing.T) {
	t.Parallel()

	// Test with default success case
	t.Run("Default success case", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		memoID := uuid.New()
		mockGen := mocks.NewMockGeneratorWithDefaultCards(memoID, userID)

		// Call the generator
		ctx := context.Background()
		cards, err := mockGen.GenerateCards(ctx, "Test memo content", userID)

		// Verify results
		assert.NoError(t, err, "Should not return an error")
		assert.Len(t, cards, 2, "Should return 2 cards")
		assert.Equal(t, userID, cards[0].UserID, "Card should have correct userID")
		assert.Equal(t, memoID, cards[0].MemoID, "Card should have correct memoID")

		// Verify call tracking
		assert.Equal(t, 1, mockGen.GenerateCardsCalls.Count, "GenerateCards should be called once")
		assert.Equal(t, "Test memo content", mockGen.GenerateCardsCalls.MemoTexts[0], "Should record correct memo text")
		assert.Equal(t, userID, mockGen.GenerateCardsCalls.UserIDs[0], "Should record correct userID")
	})

	// Test error case
	t.Run("Error case", func(t *testing.T) {
		t.Parallel()

		// Create a generator that always fails
		mockGen := mocks.MockGeneratorThatFails()

		// Call the generator
		ctx := context.Background()
		userID := uuid.New()
		cards, err := mockGen.GenerateCards(ctx, "Test memo content", userID)

		// Verify results
		assert.Error(t, err, "Should return an error")
		assert.Equal(t, generation.ErrGenerationFailed, err, "Should return ErrGenerationFailed")
		assert.Empty(t, cards, "Should not return any cards")

		// Verify call tracking
		assert.Equal(t, 1, mockGen.GenerateCardsCalls.Count, "GenerateCards should be called once")
	})

	// Test custom function
	t.Run("Custom function", func(t *testing.T) {
		t.Parallel()

		customErr := errors.New("custom error")
		mockGen := &mocks.MockGenerator{
			GenerateCardsFn: func(ctx context.Context, memoText string, userID uuid.UUID) ([]*domain.Card, error) {
				if memoText == "trigger error" {
					return nil, customErr
				}
				return []*domain.Card{}, nil
			},
		}

		// Test error case
		ctx := context.Background()
		userID := uuid.New()
		cards, err := mockGen.GenerateCards(ctx, "trigger error", userID)
		assert.Error(t, err)
		assert.Equal(t, customErr, err)
		assert.Empty(t, cards)

		// Test success case
		cards, err = mockGen.GenerateCards(ctx, "normal text", userID)
		assert.NoError(t, err)
		assert.Empty(t, cards)

		// Verify call count
		assert.Equal(t, 2, mockGen.GenerateCardsCalls.Count, "GenerateCards should be called twice")
	})

	// Test Reset
	t.Run("Reset", func(t *testing.T) {
		t.Parallel()

		mockGen := &mocks.MockGenerator{}
		ctx := context.Background()
		userID := uuid.New()

		// Make some calls
		_, _ = mockGen.GenerateCards(ctx, "memo1", userID)
		_, _ = mockGen.GenerateCards(ctx, "memo2", userID)
		assert.Equal(t, 2, mockGen.GenerateCardsCalls.Count)

		// Reset and verify
		mockGen.Reset()
		assert.Equal(t, 0, mockGen.GenerateCardsCalls.Count)
		assert.Empty(t, mockGen.GenerateCardsCalls.MemoTexts)
		assert.Empty(t, mockGen.GenerateCardsCalls.UserIDs)
	})
}
