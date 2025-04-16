package srs

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

func TestCalculateNewInterval(t *testing.T) {
	t.Parallel() // Enable parallel execution
	params := NewDefaultParams()

	testCases := []struct {
		name     string
		current  int
		consec   int
		ef       float64
		outcome  domain.ReviewOutcome
		expected int
	}{
		{
			name:     "Again outcome should reset interval",
			current:  10,
			consec:   2,
			ef:       2.5,
			outcome:  domain.ReviewOutcomeAgain,
			expected: 0,
		},
		{
			name:     "Hard outcome for first review",
			current:  0,
			consec:   0,
			ef:       2.5,
			outcome:  domain.ReviewOutcomeHard,
			expected: params.FirstReviewIntervals[domain.ReviewOutcomeHard],
		},
		{
			name:     "Good outcome for first review",
			current:  0,
			consec:   0,
			ef:       2.5,
			outcome:  domain.ReviewOutcomeGood,
			expected: params.FirstReviewIntervals[domain.ReviewOutcomeGood],
		},
		{
			name:     "Easy outcome for first review",
			current:  0,
			consec:   0,
			ef:       2.5,
			outcome:  domain.ReviewOutcomeEasy,
			expected: params.FirstReviewIntervals[domain.ReviewOutcomeEasy],
		},
		{
			name:     "Hard outcome should slightly increase interval",
			current:  10,
			consec:   2,
			ef:       2.5,
			outcome:  domain.ReviewOutcomeHard,
			expected: 12, // 10 * 1.2 = 12
		},
		{
			name:     "Good outcome should increase interval by ease factor",
			current:  10,
			consec:   2,
			ef:       2.5,
			outcome:  domain.ReviewOutcomeGood,
			expected: 25, // 10 * 2.5 = 25
		},
		{
			name:     "Good outcome after lapse",
			current:  10,
			consec:   0, // Just lapsed
			ef:       2.5,
			outcome:  domain.ReviewOutcomeGood,
			expected: 15, // 10 * 1.5 = 15
		},
		{
			name:     "Easy outcome should significantly increase interval",
			current:  10,
			consec:   2,
			ef:       2.5,
			outcome:  domain.ReviewOutcomeEasy,
			expected: 32, // 10 * 2.5 * 1.3 = 32.5 → 32
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newInterval := calculateNewInterval(tc.current, tc.consec, tc.ef, tc.outcome, params)

			if newInterval != tc.expected {
				t.Errorf("Expected interval %d, got %d", tc.expected, newInterval)
			}
		})
	}
}

func TestCalculateNewEaseFactor(t *testing.T) {
	t.Parallel() // Enable parallel execution
	params := NewDefaultParams()

	testCases := []struct {
		name     string
		current  float64
		outcome  domain.ReviewOutcome
		expected float64
	}{
		{
			name:     "Again outcome should decrease ease factor",
			current:  2.5,
			outcome:  domain.ReviewOutcomeAgain,
			expected: 2.3, // 2.5 - 0.2 = 2.3
		},
		{
			name:     "Hard outcome should slightly decrease ease factor",
			current:  2.5,
			outcome:  domain.ReviewOutcomeHard,
			expected: 2.35, // 2.5 - 0.15 = 2.35
		},
		{
			name:     "Good outcome should not change ease factor",
			current:  2.5,
			outcome:  domain.ReviewOutcomeGood,
			expected: 2.5,
		},
		{
			name:     "Easy outcome should increase ease factor",
			current:  2.3,
			outcome:  domain.ReviewOutcomeEasy,
			expected: 2.45, // 2.3 + 0.15 = 2.45
		},
		{
			name:     "Minimum ease factor should be enforced",
			current:  1.35,
			outcome:  domain.ReviewOutcomeAgain,
			expected: 1.3, // 1.35 - 0.2 = 1.15, but min is 1.3
		},
		{
			name:     "Maximum ease factor should be enforced",
			current:  2.45,
			outcome:  domain.ReviewOutcomeEasy,
			expected: 2.5, // 2.45 + 0.15 = 2.6, but max is 2.5
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newEF := calculateNewEaseFactor(tc.current, tc.outcome, params)

			// Use a small epsilon for float comparison
			epsilon := 0.001
			if newEF < tc.expected-epsilon || newEF > tc.expected+epsilon {
				t.Errorf("Expected ease factor %f, got %f", tc.expected, newEF)
			}
		})
	}
}

func TestCalculateNextReviewDate(t *testing.T) {
	t.Parallel() // Enable parallel execution
	params := NewDefaultParams()
	now := time.Now().UTC()

	testCases := []struct {
		name     string
		interval int
		outcome  domain.ReviewOutcome
		checkFn  func(time.Time) bool
	}{
		{
			name:     "Again outcome should set next review in minutes",
			interval: 0,
			outcome:  domain.ReviewOutcomeAgain,
			checkFn: func(t time.Time) bool {
				// Check if time is approximately 10 minutes in the future (within a few seconds)
				minTime := now.Add(9*time.Minute + 55*time.Second)
				maxTime := now.Add(10*time.Minute + 5*time.Second)
				return !t.Before(minTime) && !t.After(maxTime)
			},
		},
		{
			name:     "Non-Again outcome with interval 0 should set next review today",
			interval: 0,
			outcome:  domain.ReviewOutcomeGood,
			checkFn: func(t time.Time) bool {
				// Should be today
				return t.Day() == now.Day() && t.Month() == now.Month() && t.Year() == now.Year()
			},
		},
		{
			name:     "Interval of 1 should set next review tomorrow",
			interval: 1,
			outcome:  domain.ReviewOutcomeGood,
			checkFn: func(t time.Time) bool {
				expected := now.AddDate(0, 0, 1)
				return t.Day() == expected.Day() && t.Month() == expected.Month() &&
					t.Year() == expected.Year()
			},
		},
		{
			name:     "Interval of 30 should set next review in 30 days",
			interval: 30,
			outcome:  domain.ReviewOutcomeGood,
			checkFn: func(t time.Time) bool {
				expected := now.AddDate(0, 0, 30)
				return t.Day() == expected.Day() && t.Month() == expected.Month() &&
					t.Year() == expected.Year()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nextDate := calculateNextReviewDate(tc.interval, tc.outcome, now, params)

			if !tc.checkFn(nextDate) {
				t.Errorf("Next review date does not match expectations: %v", nextDate)
			}
		})
	}
}

func TestCalculateNextStats(t *testing.T) {
	t.Parallel() // Enable parallel execution
	params := NewDefaultParams()
	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now().UTC()

	// Create initial stats
	stats, err := domain.NewUserCardStats(userID, cardID)
	if err != nil {
		t.Fatalf("Failed to create stats: %v", err)
	}

	// Test that we get a new object, not a modified original
	updated := calculateNextStats(stats, domain.ReviewOutcomeGood, now, params)

	// Check that updated is not nil and is a different object
	if updated == nil {
		t.Fatal("calculateNextStats returned nil")
	}

	if updated == stats {
		t.Fatal("calculateNextStats returned the same object, not a new one")
	}

	// Check the basic outcome properties were updated
	if updated.Interval <= stats.Interval && stats.Interval == 0 {
		t.Errorf("Expected interval to increase from 0, got %d", updated.Interval)
	}

	if updated.ReviewCount != stats.ReviewCount+1 {
		t.Errorf("Expected ReviewCount to increment by 1, got %d (from %d)",
			updated.ReviewCount, stats.ReviewCount)
	}

	if updated.ConsecutiveCorrect != stats.ConsecutiveCorrect+1 {
		t.Errorf("Expected ConsecutiveCorrect to increment by 1, got %d (from %d)",
			updated.ConsecutiveCorrect, stats.ConsecutiveCorrect)
	}

	if !updated.LastReviewedAt.Equal(now) {
		t.Errorf("Expected LastReviewedAt to be %v, got %v", now, updated.LastReviewedAt)
	}

	if updated.NextReviewAt.Before(now) {
		t.Errorf("Expected NextReviewAt to be in the future, got %v", updated.NextReviewAt)
	}

	if !updated.UpdatedAt.Equal(now) {
		t.Errorf("Expected UpdatedAt to be %v, got %v", now, updated.UpdatedAt)
	}

	// Test "Again" outcome resets consecutive correct
	stats.ConsecutiveCorrect = 5
	updated = calculateNextStats(stats, domain.ReviewOutcomeAgain, now, params)

	if updated.ConsecutiveCorrect != 0 {
		t.Errorf("Expected ConsecutiveCorrect to reset to 0 for Again outcome, got %d",
			updated.ConsecutiveCorrect)
	}
}
