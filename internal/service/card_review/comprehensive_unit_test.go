//go:build integration || test_without_external_deps

package card_review_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/stretchr/testify/assert"
)

// TestComprehensiveUnitCoverage provides additional unit test coverage focusing on validation logic
func TestComprehensiveUnitCoverage(t *testing.T) {
	t.Run("answer_validation_comprehensive", func(t *testing.T) {
		// Setup service with mocks
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		service, err := card_review.NewCardReviewService(
			mockCardStore,
			mockStatsStore,
			mockSrsService,
			logger,
		)
		assert.NoError(t, err)

		userID := uuid.New()
		cardID := uuid.New()

		// Test all invalid outcomes with comprehensive coverage
		invalidOutcomes := []struct {
			name    string
			outcome domain.ReviewOutcome
		}{
			{"empty_string", ""},
			{"random_text", "invalid"},
			{"numeric", "123"},
			{"wrong_case_good", "Good"},
			{"wrong_case_easy", "Easy"},
			{"wrong_case_hard", "Hard"},
			{"wrong_case_again", "Again"},
			{"wrong_case_mixed", "GOOD"},
			{"partial_match", "goo"},
			{"extra_chars", "good!"},
			{"with_spaces", " good "},
			{"special_chars", "good@#$"},
			{"unicode", "good™"},
			{"very_long", "this_is_a_very_long_invalid_outcome_string"},
		}

		for _, tc := range invalidOutcomes {
			t.Run("invalid_outcome_"+tc.name, func(t *testing.T) {
				answer := card_review.ReviewAnswer{Outcome: tc.outcome}
				_, err := service.SubmitAnswer(context.Background(), userID, cardID, answer)
				assert.ErrorIs(t, err, card_review.ErrInvalidAnswer,
					"Outcome '%s' should be invalid", tc.outcome)
			})
		}

		// Note: We don't test valid outcomes here because they would require
		// database setup to proceed past validation. The validation logic
		// for valid outcomes is covered by integration tests.
	})

	t.Run("service_constructor_edge_cases", func(t *testing.T) {
		// Test constructor with various nil combinations
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		// Test nil card store
		service, err := card_review.NewCardReviewService(nil, mockStatsStore, mockSrsService, logger)
		assert.Error(t, err)
		assert.Nil(t, service)
		var validationErr *domain.ValidationError
		assert.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "cardStore", validationErr.Field)

		// Test nil stats store
		service, err = card_review.NewCardReviewService(mockCardStore, nil, mockSrsService, logger)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "statsStore", validationErr.Field)

		// Test nil SRS service
		service, err = card_review.NewCardReviewService(mockCardStore, mockStatsStore, nil, logger)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "srsService", validationErr.Field)

		// Test nil logger (should work, uses default)
		service, err = card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, nil)
		assert.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("error_constructor_functions", func(t *testing.T) {
		// Test NewSubmitAnswerError
		underlyingErr := assert.AnError
		msg := "test message"
		serviceErr := card_review.NewSubmitAnswerError(msg, underlyingErr)

		assert.NotNil(t, serviceErr)
		assert.Equal(t, "submit_answer", serviceErr.Operation)
		assert.Equal(t, msg, serviceErr.Message)
		assert.Equal(t, underlyingErr, serviceErr.Err)

		// Test error interface
		expectedStr := "submit_answer operation failed: test message: assert.AnError general error for testing"
		assert.Equal(t, expectedStr, serviceErr.Error())

		// Test unwrapping
		assert.Equal(t, underlyingErr, serviceErr.Unwrap())

		// Test NewGetNextCardError
		serviceErr2 := card_review.NewGetNextCardError(msg, underlyingErr)
		assert.NotNil(t, serviceErr2)
		assert.Equal(t, "get_next_card", serviceErr2.Operation)
		assert.Equal(t, msg, serviceErr2.Message)
		assert.Equal(t, underlyingErr, serviceErr2.Err)
	})

	t.Run("service_error_edge_cases", func(t *testing.T) {
		// Test ServiceError with nil underlying error
		serviceErr := &card_review.ServiceError{
			Operation: "test_op",
			Message:   "test message",
			Err:       nil,
		}

		assert.Equal(t, "test_op operation failed: test message", serviceErr.Error())
		assert.Nil(t, serviceErr.Unwrap())

		// Test with empty strings
		serviceErr2 := &card_review.ServiceError{
			Operation: "",
			Message:   "",
			Err:       nil,
		}
		assert.Equal(t, " operation failed: ", serviceErr2.Error())
	})

	t.Run("context_handling", func(t *testing.T) {
		// Test that service methods properly handle context
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		service, err := card_review.NewCardReviewService(
			mockCardStore,
			mockStatsStore,
			mockSrsService,
			logger,
		)
		assert.NoError(t, err)

		userID := uuid.New()
		cardID := uuid.New()

		// Test with various context values
		type contextKey string
		const testKey contextKey = "test"
		ctx := context.WithValue(context.Background(), testKey, "test_value")

		// GetNextCard with context
		mockCardStore.On("GetNextReviewCard", ctx, userID).Return(nil, mockCardStore.ErrCardNotFound)
		_, err = service.GetNextCard(ctx, userID)
		assert.ErrorIs(t, err, card_review.ErrNoCardsDue)

		// SubmitAnswer with context (will fail at validation, but context is passed through)
		invalidAnswer := card_review.ReviewAnswer{Outcome: "invalid"}
		_, err = service.SubmitAnswer(ctx, userID, cardID, invalidAnswer)
		assert.ErrorIs(t, err, card_review.ErrInvalidAnswer)

		mockCardStore.AssertExpectations(t)
	})

	t.Run("uuid_edge_cases", func(t *testing.T) {
		// Test with various UUID formats
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		service, err := card_review.NewCardReviewService(
			mockCardStore,
			mockStatsStore,
			mockSrsService,
			logger,
		)
		assert.NoError(t, err)

		// Test with nil UUID
		nilUUID := uuid.Nil
		invalidAnswer := card_review.ReviewAnswer{Outcome: "invalid"}

		// GetNextCard with nil UUID
		mockCardStore.On("GetNextReviewCard", context.Background(), nilUUID).Return(nil, mockCardStore.ErrCardNotFound)
		_, err = service.GetNextCard(context.Background(), nilUUID)
		assert.ErrorIs(t, err, card_review.ErrNoCardsDue)

		// SubmitAnswer with nil UUIDs and invalid answer (should fail at validation)
		_, err = service.SubmitAnswer(context.Background(), nilUUID, nilUUID, invalidAnswer)
		assert.ErrorIs(t, err, card_review.ErrInvalidAnswer)

		// Test with valid UUIDs
		userID := uuid.New()

		mockCardStore.On("GetNextReviewCard", context.Background(), userID).Return(nil, mockCardStore.ErrCardNotFound)
		_, err = service.GetNextCard(context.Background(), userID)
		assert.ErrorIs(t, err, card_review.ErrNoCardsDue)

		mockCardStore.AssertExpectations(t)
	})

	t.Run("logging_coverage", func(t *testing.T) {
		// Test service with different logger configurations
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)

		// Test with different log levels
		debugLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
		service, err := card_review.NewCardReviewService(
			mockCardStore,
			mockStatsStore,
			mockSrsService,
			debugLogger,
		)
		assert.NoError(t, err)

		userID := uuid.New()
		cardID := uuid.New()

		// Test operations that trigger different log messages
		mockCardStore.On("GetNextReviewCard", context.Background(), userID).Return(nil, mockCardStore.ErrCardNotFound)
		_, err = service.GetNextCard(context.Background(), userID)
		assert.ErrorIs(t, err, card_review.ErrNoCardsDue)

		// Test with invalid answer (triggers warning log)
		invalidAnswer := card_review.ReviewAnswer{Outcome: "invalid"}
		_, err = service.SubmitAnswer(context.Background(), userID, cardID, invalidAnswer)
		assert.ErrorIs(t, err, card_review.ErrInvalidAnswer)

		mockCardStore.AssertExpectations(t)
	})
}

// TestValidOutcomeConstants ensures all domain.ReviewOutcome constants are defined correctly
func TestValidOutcomeConstants(t *testing.T) {
	// Test that the domain constants have expected values
	// This ensures the constants haven't changed unexpectedly
	assert.Equal(t, domain.ReviewOutcome("again"), domain.ReviewOutcomeAgain)
	assert.Equal(t, domain.ReviewOutcome("hard"), domain.ReviewOutcomeHard)
	assert.Equal(t, domain.ReviewOutcome("good"), domain.ReviewOutcomeGood)
	assert.Equal(t, domain.ReviewOutcome("easy"), domain.ReviewOutcomeEasy)

	// Test that the constants are not empty
	assert.NotEmpty(t, domain.ReviewOutcomeAgain)
	assert.NotEmpty(t, domain.ReviewOutcomeHard)
	assert.NotEmpty(t, domain.ReviewOutcomeGood)
	assert.NotEmpty(t, domain.ReviewOutcomeEasy)
}
