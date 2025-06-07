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

func TestNewServiceWithParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		params      *Params
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_params",
			params:      NewDefaultParams(),
			expectError: false,
		},
		{
			name:        "nil_params",
			params:      nil,
			expectError: true,
			errorMsg:    "params cannot be nil",
		},
		{
			name: "custom_params",
			params: NewParams(ParamsConfig{
				MinEaseFactor: 1.5,
				MaxEaseFactor: 3.0,
			}),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewServiceWithParams(tt.params)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
				require.Nil(t, service)
			} else {
				require.NoError(t, err)
				require.NotNil(t, service)

				// Verify params were set correctly
				defaultService, ok := service.(*defaultService)
				require.True(t, ok, "Expected *defaultService type")
				require.Equal(t, tt.params, defaultService.params)
			}
		})
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

	t.Run("valid 7-day postponement", func(t *testing.T) {
		t.Parallel()
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
	})

	t.Run("large number of days", func(t *testing.T) {
		t.Parallel()
		// Create a new copy of stats for this test to avoid shared state
		localStats, err := domain.NewUserCardStats(userID, cardID)
		if err != nil {
			t.Fatalf("Failed to create stats: %v", err)
		}
		localStats.NextReviewAt = now

		// Test with a very large number of days (over a year)
		days := 500
		updatedStats, err := service.PostponeReview(localStats, days, now)
		if err != nil {
			t.Fatalf("PostponeReview returned error for %d days: %v", days, err)
		}

		expectedNextReview := now.AddDate(0, 0, days)
		if !updatedStats.NextReviewAt.Equal(expectedNextReview) {
			t.Errorf(
				"Expected NextReviewAt to be %v, got %v",
				expectedNextReview,
				updatedStats.NextReviewAt,
			)
		}
	})

	t.Run("DST transition", func(t *testing.T) {
		t.Parallel()
		// Create a new copy of stats for this test
		localStats, err := domain.NewUserCardStats(userID, cardID)
		if err != nil {
			t.Fatalf("Failed to create stats: %v", err)
		}

		// Create a date in February
		feb := time.Date(2025, time.February, 15, 12, 0, 0, 0, time.UTC)
		localStats.NextReviewAt = feb

		// Postpone past a DST transition (30 days should cross into March/April when many regions change DST)
		updatedStats, err := service.PostponeReview(localStats, 30, feb)
		if err != nil {
			t.Fatalf("PostponeReview returned error: %v", err)
		}

		expectedNextReview := feb.AddDate(0, 0, 30)
		if !updatedStats.NextReviewAt.Equal(expectedNextReview) {
			t.Errorf(
				"Expected NextReviewAt to be %v, got %v",
				expectedNextReview,
				updatedStats.NextReviewAt,
			)
		}

		// The hour should remain the same in UTC regardless of DST
		if updatedStats.NextReviewAt.Hour() != feb.Hour() {
			t.Errorf(
				"Hour changed after DST transition: expected %d, got %d",
				feb.Hour(),
				updatedStats.NextReviewAt.Hour(),
			)
		}
	})

	t.Run("leap year", func(t *testing.T) {
		t.Parallel()
		// Create a new copy of stats for this test
		localStats, err := domain.NewUserCardStats(userID, cardID)
		if err != nil {
			t.Fatalf("Failed to create stats: %v", err)
		}

		// Create a date before Feb 29 in a leap year
		leapYearDate := time.Date(2024, time.February, 15, 12, 0, 0, 0, time.UTC) // 2024 is a leap year
		localStats.NextReviewAt = leapYearDate

		// Postpone past Feb 29
		updatedStats, err := service.PostponeReview(localStats, 20, leapYearDate)
		if err != nil {
			t.Fatalf("PostponeReview returned error: %v", err)
		}

		expectedNextReview := leapYearDate.AddDate(0, 0, 20)
		if !updatedStats.NextReviewAt.Equal(expectedNextReview) {
			t.Errorf(
				"Expected NextReviewAt to be %v, got %v",
				expectedNextReview,
				updatedStats.NextReviewAt,
			)
		}

		// March 6, 2024 (after Feb 29)
		if updatedStats.NextReviewAt.Day() != 6 || updatedStats.NextReviewAt.Month() != time.March {
			t.Errorf(
				"Incorrect date after leap year: expected Mar 6, got %s %d",
				updatedStats.NextReviewAt.Month(),
				updatedStats.NextReviewAt.Day(),
			)
		}
	})

	t.Run("invalid days", func(t *testing.T) {
		t.Parallel()
		// Create a new copy of stats for this test
		localStats, err := domain.NewUserCardStats(userID, cardID)
		if err != nil {
			t.Fatalf("Failed to create stats: %v", err)
		}

		// Test invalid days
		_, err = service.PostponeReview(localStats, 0, now)
		if err == nil {
			t.Error("Expected error for 0 days, got nil")
		}

		_, err = service.PostponeReview(localStats, -1, now)
		if err == nil {
			t.Error("Expected error for negative days, got nil")
		}
	})

	t.Run("nil stats", func(t *testing.T) {
		t.Parallel()
		// Test nil stats
		_, err = service.PostponeReview(nil, 7, now)
		if err == nil {
			t.Error("Expected error for nil stats, got nil")
		}
	})
}
