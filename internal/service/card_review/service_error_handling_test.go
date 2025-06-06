package card_review_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestServiceErrors tests the service error types and their behavior
func TestServiceErrors(t *testing.T) {
	t.Run("service_error_creation", func(t *testing.T) {
		underlyingErr := errors.New("database connection failed")
		message := "failed to process request"

		// Test NewSubmitAnswerError
		submitErr := card_review.NewSubmitAnswerError(message, underlyingErr)
		assert.NotNil(t, submitErr)
		assert.Equal(t, "submit_answer", submitErr.Operation)
		assert.Equal(t, message, submitErr.Message)
		assert.Equal(t, underlyingErr, submitErr.Err)

		// Test error message formatting
		expectedMsg := "submit_answer operation failed: failed to process request: database connection failed"
		assert.Equal(t, expectedMsg, submitErr.Error())

		// Test unwrapping
		assert.Equal(t, underlyingErr, submitErr.Unwrap())

		// Test NewGetNextCardError
		getNextErr := card_review.NewGetNextCardError(message, underlyingErr)
		assert.NotNil(t, getNextErr)
		assert.Equal(t, "get_next_card", getNextErr.Operation)
		assert.Equal(t, message, getNextErr.Message)
		assert.Equal(t, underlyingErr, getNextErr.Err)
	})

	t.Run("service_error_without_underlying", func(t *testing.T) {
		message := "operation failed"

		// Test error without underlying error
		serviceErr := card_review.NewSubmitAnswerError(message, nil)
		assert.NotNil(t, serviceErr)

		expectedMsg := "submit_answer operation failed: operation failed"
		assert.Equal(t, expectedMsg, serviceErr.Error())

		assert.Nil(t, serviceErr.Unwrap())
	})

	t.Run("error_type_checking", func(t *testing.T) {
		underlyingErr := errors.New("database error")
		serviceErr := card_review.NewSubmitAnswerError("test", underlyingErr)

		// Test errors.Is with underlying error
		assert.True(t, errors.Is(serviceErr, underlyingErr))

		// Test errors.As with service error
		var targetErr *card_review.ServiceError
		assert.True(t, errors.As(serviceErr, &targetErr))
		assert.Equal(t, serviceErr, targetErr)

		// Test with unrelated error
		otherErr := errors.New("other error")
		assert.False(t, errors.Is(serviceErr, otherErr))
	})
}

// TestGetNextCard_ErrorHandling tests error handling scenarios for GetNextCard
func TestGetNextCard_ErrorHandling(t *testing.T) {
	userID := uuid.New()

	t.Run("store_not_found_error", func(t *testing.T) {
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		// Set up mock to return store.ErrNotFound
		mockCardStore.On("GetNextReviewCard", mock.Anything, userID).
			Return(nil, store.ErrNotFound)

		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		require.NoError(t, err)

		_, err = service.GetNextCard(context.Background(), userID)
		assert.ErrorIs(t, err, card_review.ErrNoCardsDue)

		mockCardStore.AssertExpectations(t)
	})

	t.Run("store_card_not_found_error", func(t *testing.T) {
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		// Set up mock to return store.ErrCardNotFound
		mockCardStore.On("GetNextReviewCard", mock.Anything, userID).
			Return(nil, store.ErrCardNotFound)

		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		require.NoError(t, err)

		_, err = service.GetNextCard(context.Background(), userID)
		assert.ErrorIs(t, err, card_review.ErrNoCardsDue)

		mockCardStore.AssertExpectations(t)
	})

	t.Run("generic_database_error", func(t *testing.T) {
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		// Set up mock to return generic database error
		dbErr := errors.New("connection timeout")
		mockCardStore.On("GetNextReviewCard", mock.Anything, userID).
			Return(nil, dbErr)

		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		require.NoError(t, err)

		_, err = service.GetNextCard(context.Background(), userID)

		// Should be wrapped in ServiceError
		var serviceErr *card_review.ServiceError
		assert.ErrorAs(t, err, &serviceErr)
		assert.Equal(t, "get_next_card", serviceErr.Operation)
		assert.Equal(t, "database error", serviceErr.Message)
		assert.True(t, errors.Is(err, dbErr))

		mockCardStore.AssertExpectations(t)
	})

	t.Run("successful_card_retrieval", func(t *testing.T) {
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		// Create test card
		card := createTestCard(userID)
		mockCardStore.On("GetNextReviewCard", mock.Anything, userID).
			Return(card, nil)

		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		require.NoError(t, err)

		retrievedCard, err := service.GetNextCard(context.Background(), userID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedCard)
		assert.Equal(t, card.ID, retrievedCard.ID)
		assert.Equal(t, card.UserID, retrievedCard.UserID)

		mockCardStore.AssertExpectations(t)
	})
}

// TestServiceConstruction tests service constructor edge cases
func TestServiceConstruction(t *testing.T) {
	mockCardStore := NewMockCardStore()
	mockStatsStore := new(MockUserCardStatsStore)
	mockSrsService := new(MockSRSService)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("all_valid_dependencies", func(t *testing.T) {
		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		assert.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("nil_logger_uses_default", func(t *testing.T) {
		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, nil)
		assert.NoError(t, err)
		assert.NotNil(t, service)
		// Logger should be defaulted internally
	})

	t.Run("validation_error_structure", func(t *testing.T) {
		// Test that validation errors have correct structure
		_, err := card_review.NewCardReviewService(nil, mockStatsStore, mockSrsService, logger)
		assert.Error(t, err)

		var validationErr *domain.ValidationError
		assert.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "cardStore", validationErr.Field)
		assert.Equal(t, "cannot be nil", validationErr.Message)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// TestContextHandling tests context propagation through service methods
func TestContextHandling(t *testing.T) {
	userID := uuid.New()

	t.Run("context_with_values", func(t *testing.T) {
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		// Create context with values using custom key type
		testKey := contextKey("test_key")
		ctx := context.WithValue(context.Background(), testKey, "test_value")

		// Set up mock to verify context is passed through
		mockCardStore.On("GetNextReviewCard", mock.MatchedBy(func(passedCtx context.Context) bool {
			// Verify the context value is preserved using the same key type
			return passedCtx.Value(testKey) == "test_value"
		}), userID).Return(nil, store.ErrCardNotFound)

		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		require.NoError(t, err)

		_, err = service.GetNextCard(ctx, userID)
		assert.ErrorIs(t, err, card_review.ErrNoCardsDue)

		mockCardStore.AssertExpectations(t)
	})

	t.Run("cancelled_context", func(t *testing.T) {
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Mock should still be called even with cancelled context
		mockCardStore.On("GetNextReviewCard", mock.Anything, userID).
			Return(nil, store.ErrCardNotFound)

		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		require.NoError(t, err)

		_, err = service.GetNextCard(ctx, userID)
		assert.ErrorIs(t, err, card_review.ErrNoCardsDue)

		mockCardStore.AssertExpectations(t)
	})
}

// TestUUIDHandling tests UUID edge cases
func TestUUIDHandling(t *testing.T) {
	t.Run("nil_uuid_handling", func(t *testing.T) {
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		// Test with nil UUID
		mockCardStore.On("GetNextReviewCard", mock.Anything, uuid.Nil).
			Return(nil, store.ErrCardNotFound)

		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		require.NoError(t, err)

		_, err = service.GetNextCard(context.Background(), uuid.Nil)
		assert.ErrorIs(t, err, card_review.ErrNoCardsDue)

		mockCardStore.AssertExpectations(t)
	})

	t.Run("valid_uuid_handling", func(t *testing.T) {
		validUserID := uuid.New()

		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		card := createTestCard(validUserID)
		mockCardStore.On("GetNextReviewCard", mock.Anything, validUserID).
			Return(card, nil)

		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		require.NoError(t, err)

		retrievedCard, err := service.GetNextCard(context.Background(), validUserID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedCard)
		assert.Equal(t, validUserID, retrievedCard.UserID)

		mockCardStore.AssertExpectations(t)
	})
}
