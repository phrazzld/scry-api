package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"log/slog"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file contains unit tests for the CardService implementation
// focusing on the methods added in T014, T015, and T016:
// - UpdateCardContent
// - DeleteCard
// - PostponeCard

// TestCardService_UnitTests_UpdateCardContent tests the UpdateCardContent method
func TestCardService_UnitTests_UpdateCardContent(t *testing.T) {
	// Setup shared test values
	ctx := context.Background()
	userID := uuid.New()
	cardID := uuid.New()
	content := json.RawMessage(`{"front": "Test front", "back": "Test back"}`)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create mocks
	cardRepo := new(MockCardRepository)
	statsRepo := new(MockStatsRepository)
	srsService := new(MockSRSService)

	// Create the service
	service, err := NewCardService(cardRepo, statsRepo, srsService, logger)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		// Mock card with matching owner
		card := &domain.Card{
			ID:      cardID,
			UserID:  userID,
			Content: json.RawMessage(`{"front": "Old front", "back": "Old back"}`),
		}

		// Setup expectations
		cardRepo.On("GetByID", ctx, cardID).Return(card, nil).Once()
		cardRepo.On("UpdateContent", ctx, cardID, content).Return(nil).Once()

		// Execute the method
		err := service.UpdateCardContent(ctx, userID, cardID, content)

		// Assertions
		assert.NoError(t, err)
		cardRepo.AssertExpectations(t)
	})

	t.Run("card not found", func(t *testing.T) {
		// Setup expectations for card not found
		cardRepo.On("GetByID", ctx, cardID).Return(nil, store.ErrCardNotFound).Once()

		// Execute the method
		err := service.UpdateCardContent(ctx, userID, cardID, content)

		// Assertions
		assert.Error(t, err)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), store.ErrCardNotFound)
		cardRepo.AssertExpectations(t)
	})

	t.Run("card fetch error", func(t *testing.T) {
		// Setup expectations for database error
		dbErr := errors.New("database error")
		cardRepo.On("GetByID", ctx, cardID).Return(nil, dbErr).Once()

		// Execute the method
		err := service.UpdateCardContent(ctx, userID, cardID, content)

		// Assertions
		assert.Error(t, err)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), dbErr)
		cardRepo.AssertExpectations(t)
	})

	t.Run("not card owner", func(t *testing.T) {
		// Mock card with different owner
		differentUserID := uuid.New()
		card := &domain.Card{
			ID:      cardID,
			UserID:  differentUserID, // Different user ID
			Content: json.RawMessage(`{"front": "Old front", "back": "Old back"}`),
		}

		// Setup expectations
		cardRepo.On("GetByID", ctx, cardID).Return(card, nil).Once()

		// Execute the method
		err := service.UpdateCardContent(ctx, userID, cardID, content)

		// Assertions
		assert.Error(t, err)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), ErrNotOwned)
		cardRepo.AssertExpectations(t)
	})

	t.Run("update error", func(t *testing.T) {
		// Mock card with matching owner
		card := &domain.Card{
			ID:      cardID,
			UserID:  userID,
			Content: json.RawMessage(`{"front": "Old front", "back": "Old back"}`),
		}

		// Setup expectations with update error
		updateErr := errors.New("update error")
		cardRepo.On("GetByID", ctx, cardID).Return(card, nil).Once()
		cardRepo.On("UpdateContent", ctx, cardID, content).Return(updateErr).Once()

		// Execute the method
		err := service.UpdateCardContent(ctx, userID, cardID, content)

		// Assertions
		assert.Error(t, err)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), updateErr)
		cardRepo.AssertExpectations(t)
	})
}

// TestCardService_UnitTests_DeleteCard tests the DeleteCard method
func TestCardService_UnitTests_DeleteCard(t *testing.T) {
	// Setup shared test values
	ctx := context.Background()
	userID := uuid.New()
	cardID := uuid.New()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create mocks
	cardRepo := new(MockCardRepository)
	statsRepo := new(MockStatsRepository)
	srsService := new(MockSRSService)

	// Create the service
	service, err := NewCardService(cardRepo, statsRepo, srsService, logger)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		// Mock card with matching owner
		card := &domain.Card{
			ID:     cardID,
			UserID: userID,
		}

		// Setup expectations
		cardRepo.On("GetByID", ctx, cardID).Return(card, nil).Once()
		cardRepo.On("Delete", ctx, cardID).Return(nil).Once()

		// Execute the method
		err := service.DeleteCard(ctx, userID, cardID)

		// Assertions
		assert.NoError(t, err)
		cardRepo.AssertExpectations(t)
	})

	t.Run("card not found", func(t *testing.T) {
		// Setup expectations for card not found
		cardRepo.On("GetByID", ctx, cardID).Return(nil, store.ErrCardNotFound).Once()

		// Execute the method
		err := service.DeleteCard(ctx, userID, cardID)

		// Assertions
		assert.Error(t, err)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), store.ErrCardNotFound)
		cardRepo.AssertExpectations(t)
	})

	t.Run("card fetch error", func(t *testing.T) {
		// Setup expectations for database error
		dbErr := errors.New("database error")
		cardRepo.On("GetByID", ctx, cardID).Return(nil, dbErr).Once()

		// Execute the method
		err := service.DeleteCard(ctx, userID, cardID)

		// Assertions
		assert.Error(t, err)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), dbErr)
		cardRepo.AssertExpectations(t)
	})

	t.Run("not card owner", func(t *testing.T) {
		// Mock card with different owner
		differentUserID := uuid.New()
		card := &domain.Card{
			ID:     cardID,
			UserID: differentUserID, // Different user ID
		}

		// Setup expectations
		cardRepo.On("GetByID", ctx, cardID).Return(card, nil).Once()

		// Execute the method
		err := service.DeleteCard(ctx, userID, cardID)

		// Assertions
		assert.Error(t, err)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), ErrNotOwned)
		cardRepo.AssertExpectations(t)
	})

	t.Run("delete error", func(t *testing.T) {
		// Mock card with matching owner
		card := &domain.Card{
			ID:     cardID,
			UserID: userID,
		}

		// Setup expectations with delete error
		deleteErr := errors.New("delete error")
		cardRepo.On("GetByID", ctx, cardID).Return(card, nil).Once()
		cardRepo.On("Delete", ctx, cardID).Return(deleteErr).Once()

		// Execute the method
		err := service.DeleteCard(ctx, userID, cardID)

		// Assertions
		assert.Error(t, err)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), deleteErr)
		cardRepo.AssertExpectations(t)
	})
}

// TestCardService_UnitTests_PostponeCard_NoTransaction tests the non-transaction parts of PostponeCard
// (the validation and card ownership checks that happen before the transaction)
func TestCardService_UnitTests_PostponeCard_NoTransaction(t *testing.T) {
	// Setup shared test values
	ctx := context.Background()
	userID := uuid.New()
	cardID := uuid.New()
	days := 7
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create mocks
	cardRepo := new(MockCardRepository)
	statsRepo := new(MockStatsRepository)
	srsService := new(MockSRSService)

	// Create the service
	service, err := NewCardService(cardRepo, statsRepo, srsService, logger)
	require.NoError(t, err)

	t.Run("invalid days", func(t *testing.T) {
		// Execute with invalid days
		result, err := service.PostponeCard(ctx, userID, cardID, 0)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), srs.ErrInvalidDays)
	})

	t.Run("card not found", func(t *testing.T) {
		// Setup expectations for card not found
		cardRepo.On("GetByID", ctx, cardID).Return(nil, store.ErrCardNotFound).Once()

		// Execute the method
		result, err := service.PostponeCard(ctx, userID, cardID, days)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), store.ErrCardNotFound)
		cardRepo.AssertExpectations(t)
	})

	t.Run("card fetch error", func(t *testing.T) {
		// Setup expectations for database error
		dbErr := errors.New("database error")
		cardRepo.On("GetByID", ctx, cardID).Return(nil, dbErr).Once()

		// Execute the method
		result, err := service.PostponeCard(ctx, userID, cardID, days)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), dbErr)
		cardRepo.AssertExpectations(t)
	})

	t.Run("not card owner", func(t *testing.T) {
		// Mock card with different owner
		differentUserID := uuid.New()
		card := &domain.Card{
			ID:     cardID,
			UserID: differentUserID, // Different user ID
		}

		// Setup expectations
		cardRepo.On("GetByID", ctx, cardID).Return(card, nil).Once()

		// Execute the method
		result, err := service.PostponeCard(ctx, userID, cardID, days)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		var cardSvcErr *CardServiceError
		assert.True(t, errors.As(err, &cardSvcErr))
		assert.ErrorIs(t, errors.Unwrap(err), ErrNotOwned)
		cardRepo.AssertExpectations(t)
	})
}
