package srs

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultService(t *testing.T) {
	t.Parallel() // Enable parallel execution
	service, err := NewDefaultService()
	require.NoError(t, err, "Failed to create SRS service")
	if service == nil {
		t.Fatal("Expected non-nil service")
	}

	// Check if default params are present
	defaultService, ok := service.(*defaultService)
	if !ok {
		t.Fatal("Expected *defaultService type")
	}

	if defaultService.params == nil {
		t.Fatal("Expected non-nil params")
	}
}

func TestCalculateNextReview(t *testing.T) {
	t.Parallel() // Enable parallel execution
	service, err := NewDefaultService()
	require.NoError(t, err, "Failed to create SRS service")
	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now().UTC()

	// Create initial stats
	initialStats, err := domain.NewUserCardStats(userID, cardID)
	if err != nil {
		t.Fatalf("Failed to create initial stats: %v", err)
	}

	testCases := []struct {
		name                     string
		initialStats             *domain.UserCardStats
		outcome                  domain.ReviewOutcome
		expectInterval           func(int) bool
		expectEaseFactor         func(float64) bool
		expectConsecutiveCorrect func(int) bool
	}{
		{
			name:                     "Again outcome should reset interval",
			initialStats:             initialStats,
			outcome:                  domain.ReviewOutcomeAgain,
			expectInterval:           func(i int) bool { return i == 0 },
			expectEaseFactor:         func(ef float64) bool { return ef < 2.5 && ef >= 1.3 },
			expectConsecutiveCorrect: func(cc int) bool { return cc == 0 },
		},
		{
			name:                     "Hard outcome should slightly increase interval",
			initialStats:             initialStats,
			outcome:                  domain.ReviewOutcomeHard,
			expectInterval:           func(i int) bool { return i == 1 }, // First review
			expectEaseFactor:         func(ef float64) bool { return ef < 2.5 && ef >= 1.3 },
			expectConsecutiveCorrect: func(cc int) bool { return cc == 1 },
		},
		{
			name:                     "Good outcome should normally increase interval",
			initialStats:             initialStats,
			outcome:                  domain.ReviewOutcomeGood,
			expectInterval:           func(i int) bool { return i == 1 }, // First review
			expectEaseFactor:         func(ef float64) bool { return ef == 2.5 },
			expectConsecutiveCorrect: func(cc int) bool { return cc == 1 },
		},
		{
			name:                     "Easy outcome should significantly increase interval",
			initialStats:             initialStats,
			outcome:                  domain.ReviewOutcomeEasy,
			expectInterval:           func(i int) bool { return i == 2 }, // First review
			expectEaseFactor:         func(ef float64) bool { return ef >= 2.5 },
			expectConsecutiveCorrect: func(cc int) bool { return cc == 1 },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updatedStats, err := service.CalculateNextReview(tc.initialStats, tc.outcome, now)
			if err != nil {
				t.Fatalf("CalculateNextReview returned error: %v", err)
			}

			if updatedStats == nil {
				t.Fatal("CalculateNextReview returned nil stats")
			}

			// Check that the updated stats has expected values
			if !tc.expectInterval(updatedStats.Interval) {
				t.Errorf("Unexpected interval: got %d", updatedStats.Interval)
			}

			if !tc.expectEaseFactor(updatedStats.EaseFactor) {
				t.Errorf("Unexpected ease factor: got %f", updatedStats.EaseFactor)
			}

			if !tc.expectConsecutiveCorrect(updatedStats.ConsecutiveCorrect) {
				t.Errorf("Unexpected consecutive correct: got %d", updatedStats.ConsecutiveCorrect)
			}

			// Check review count increment
			if updatedStats.ReviewCount != tc.initialStats.ReviewCount+1 {
				t.Errorf("Expected review count to increment by 1, got %d (from %d)",
					updatedStats.ReviewCount, tc.initialStats.ReviewCount)
			}

			// Check last reviewed time updated
			if !updatedStats.LastReviewedAt.Equal(now) {
				t.Errorf(
					"Expected LastReviewedAt to be %v, got %v",
					now,
					updatedStats.LastReviewedAt,
				)
			}

			// Check that next review time is set and in the future
			if updatedStats.NextReviewAt.Before(now) && tc.outcome != domain.ReviewOutcomeAgain {
				t.Errorf(
					"Expected NextReviewAt to be in the future for non-Again outcomes, got %v",
					updatedStats.NextReviewAt,
				)
			}

			// Check UpdatedAt is set to current time
			if !updatedStats.UpdatedAt.Equal(now) {
				t.Errorf("Expected UpdatedAt to be %v, got %v", now, updatedStats.UpdatedAt)
			}

			// Check that the original stats weren't modified (immutability)
			if tc.initialStats.Interval != initialStats.Interval ||
				tc.initialStats.EaseFactor != initialStats.EaseFactor ||
				tc.initialStats.ConsecutiveCorrect != initialStats.ConsecutiveCorrect ||
				tc.initialStats.ReviewCount != initialStats.ReviewCount ||
				!tc.initialStats.LastReviewedAt.Equal(initialStats.LastReviewedAt) ||
				!tc.initialStats.NextReviewAt.Equal(initialStats.NextReviewAt) ||
				!tc.initialStats.UpdatedAt.Equal(initialStats.UpdatedAt) {
				t.Error("Original stats object was modified")
			}
		})
	}

	// Test invalid outcome
	_, err = service.CalculateNextReview(initialStats, "invalid_outcome", now)
	if err == nil {
		t.Error("Expected error for invalid outcome, got nil")
	}

	// Test nil stats
	_, err = service.CalculateNextReview(nil, domain.ReviewOutcomeGood, now)
	if err == nil {
		t.Error("Expected error for nil stats, got nil")
	}
}

func TestPostponeReview(t *testing.T) {
	t.Parallel() // Enable parallel execution
	service, err := NewDefaultService()
	require.NoError(t, err, "Failed to create SRS service")
	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now().UTC()

	// Create initial stats
	initialStats, err := domain.NewUserCardStats(userID, cardID)
	if err != nil {
		t.Fatalf("Failed to create initial stats: %v", err)
	}

	// Set a specific NextReviewAt time for predictable testing
	initialStats.NextReviewAt = now

	// Test valid postponement
	updatedStats, err := service.PostponeReview(initialStats, 7, now)
	if err != nil {
		t.Fatalf("PostponeReview returned error: %v", err)
	}

	expectedNextReview := now.AddDate(0, 0, 7)
	if !updatedStats.NextReviewAt.Equal(expectedNextReview) {
		t.Errorf(
			"Expected NextReviewAt to be %v, got %v",
			expectedNextReview,
			updatedStats.NextReviewAt,
		)
	}

	if !updatedStats.UpdatedAt.Equal(now) {
		t.Errorf("Expected UpdatedAt to be %v, got %v", now, updatedStats.UpdatedAt)
	}

	// Test that original stats weren't modified
	if !initialStats.NextReviewAt.Equal(now) {
		t.Error("Original stats object was modified")
	}

	// Test invalid days
	_, err = service.PostponeReview(initialStats, 0, now)
	if err == nil {
		t.Error("Expected error for 0 days, got nil")
	}

	_, err = service.PostponeReview(initialStats, -1, now)
	if err == nil {
		t.Error("Expected error for negative days, got nil")
	}

	// Test nil stats
	_, err = service.PostponeReview(nil, 7, now)
	if err == nil {
		t.Error("Expected error for nil stats, got nil")
	}
}
