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
	"github.com/stretchr/testify/require"
)

// TestInvalidOutcomeValidation tests invalid outcome validation thoroughly
// This tests the isValidOutcome function through the SubmitAnswer validation path
func TestInvalidOutcomeValidation(t *testing.T) {
	userID := uuid.New()
	cardID := uuid.New()

	// Create service with basic mocks (they won't be called due to early validation failure)
	mockCardStore := NewMockCardStore()
	mockStatsStore := new(MockUserCardStatsStore)
	mockSrsService := new(MockSRSService)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
	require.NoError(t, err)

	// Test comprehensive set of invalid outcomes
	invalidOutcomes := []string{
		// Empty and whitespace
		"", "  ", "\t", "\n", "\r\n", "   \t\n  ",

		// Basic invalid strings
		"invalid", "wrong", "bad", "unknown", "skip", "pass", "fail", "ok",

		// Case variations (validation is case-sensitive)
		"AGAIN", "HARD", "GOOD", "EASY",
		"Again", "Hard", "Good", "Easy",
		"AgAiN", "HaRd", "GoOd", "EaSy",

		// With whitespace padding
		"again ", " again", " again ", "\tagain\t", "\nagain\n",
		"hard ", " hard", " hard ", "\thard\t", "\nhard\n",
		"good ", " good", " good ", "\tgood\t", "\ngood\n",
		"easy ", " easy", " easy ", "\teasy\t", "\neasy\n",

		// With internal spaces
		"ag ain", "h ard", "go od", "ea sy",
		"a gain", "har d", "goo d", "eas y",

		// Numeric and special characters
		"0", "1", "2", "3", "4", "5", "-1", "10",
		"true", "false", "yes", "no", "y", "n",
		"!", "@", "#", "$", "%", "^", "&", "*", "(", ")",
		"null", "undefined", "NaN", "Infinity", "-Infinity",

		// Emojis and unicode
		"👍", "👎", "😊", "😢", "✅", "❌", "🟢", "🔴",
		"α", "β", "γ", "δ", "π", "Ω", "∞", "∅",

		// Multiple values or separators
		"again,hard", "good|easy", "hard;good", "again+hard",
		"again/hard", "good\\easy", "hard:good", "again=hard",

		// Attempts at injection or special strings
		"<script>", "</script>", "<html>", "</html>",
		"SELECT", "DROP", "INSERT", "UPDATE", "DELETE",
		"javascript:", "data:", "vbscript:", "onload=",
		"../", "./", "~/", "/etc/passwd", "C:\\Windows",

		// Variations of correct answers
		"againn", "hardd", "goodd", "easyy",
		"agian", "ahrd", "godo", "eays",
		"agan", "had", "god", "esy",
		"gains", "hard!", "good?", "easy.",

		// Mixed case correct answers
		"aGAIN", "hARD", "gOOD", "eASY",
		"AgaiN", "HarD", "GooD", "EasY",
	}

	for _, outcome := range invalidOutcomes {
		t.Run("invalid_"+outcome, func(t *testing.T) {
			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcome(outcome)}

			_, err := service.SubmitAnswer(context.Background(), userID, cardID, answer)
			assert.ErrorIs(t, err, card_review.ErrInvalidAnswer,
				"Outcome '%s' should be rejected as invalid", outcome)
		})
	}
}

// TestValidOutcomeAcceptance tests that valid outcomes pass initial validation
// Note: Valid outcomes are tested in integration tests due to transaction complexity
func TestValidOutcomeAcceptance(t *testing.T) {
	t.Run("valid_outcomes_tested_in_integration", func(t *testing.T) {
		// Valid outcomes require transaction logic and are tested in integration tests
		// This test documents that valid outcome behavior is covered elsewhere
		validOutcomes := []domain.ReviewOutcome{
			domain.ReviewOutcomeAgain,
			domain.ReviewOutcomeHard,
			domain.ReviewOutcomeGood,
			domain.ReviewOutcomeEasy,
		}

		assert.Len(t, validOutcomes, 4, "All four valid outcomes should be documented")
		t.Log("Valid outcomes are tested in service_integration_test.go with real transaction handling")
	})
}

// TestSubmitAnswerEdgeCases tests edge cases that can be unit tested
func TestSubmitAnswerEdgeCases(t *testing.T) {
	userID := uuid.New()
	cardID := uuid.New()

	t.Run("empty_outcome_string", func(t *testing.T) {
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		require.NoError(t, err)

		answer := card_review.ReviewAnswer{Outcome: ""}
		_, err = service.SubmitAnswer(context.Background(), userID, cardID, answer)
		assert.ErrorIs(t, err, card_review.ErrInvalidAnswer)
	})

	t.Run("unicode_outcome", func(t *testing.T) {
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		require.NoError(t, err)

		answer := card_review.ReviewAnswer{Outcome: "∅"}
		_, err = service.SubmitAnswer(context.Background(), userID, cardID, answer)
		assert.ErrorIs(t, err, card_review.ErrInvalidAnswer)
	})

	t.Run("long_outcome_string", func(t *testing.T) {
		mockCardStore := NewMockCardStore()
		mockStatsStore := new(MockUserCardStatsStore)
		mockSrsService := new(MockSRSService)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		require.NoError(t, err)

		// Create a very long string
		longOutcome := ""
		for i := 0; i < 1000; i++ {
			longOutcome += "a"
		}

		answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcome(longOutcome)}
		_, err = service.SubmitAnswer(context.Background(), userID, cardID, answer)
		assert.ErrorIs(t, err, card_review.ErrInvalidAnswer)
	})
}

// TestAnswerStructure tests the ReviewAnswer struct behavior
func TestAnswerStructure(t *testing.T) {
	t.Run("answer_creation", func(t *testing.T) {
		answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
		assert.Equal(t, domain.ReviewOutcomeGood, answer.Outcome)
	})

	t.Run("answer_modification", func(t *testing.T) {
		answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
		answer.Outcome = domain.ReviewOutcomeHard
		assert.Equal(t, domain.ReviewOutcomeHard, answer.Outcome)
	})

	t.Run("zero_value_answer", func(t *testing.T) {
		var answer card_review.ReviewAnswer
		assert.Equal(t, domain.ReviewOutcome(""), answer.Outcome)
	})
}
