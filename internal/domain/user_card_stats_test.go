package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewUserCardStats(t *testing.T) {
	// Test valid user card stats creation
	userID := uuid.New()
	cardID := uuid.New()

	stats, err := NewUserCardStats(userID, cardID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, stats.UserID)
	}

	if stats.CardID != cardID {
		t.Errorf("Expected card ID %s, got %s", cardID, stats.CardID)
	}

	if stats.Interval != 0 {
		t.Errorf("Expected interval 0, got %d", stats.Interval)
	}

	if stats.EaseFactor != 2.5 {
		t.Errorf("Expected ease factor 2.5, got %f", stats.EaseFactor)
	}

	if stats.ConsecutiveCorrect != 0 {
		t.Errorf("Expected consecutive correct 0, got %d", stats.ConsecutiveCorrect)
	}

	if !stats.LastReviewedAt.IsZero() {
		t.Errorf("Expected zero LastReviewedAt, got %v", stats.LastReviewedAt)
	}

	now := time.Now().UTC()
	maxDiff := 2 * time.Second

	if stats.NextReviewAt.Sub(now) > maxDiff || now.Sub(stats.NextReviewAt) > maxDiff {
		t.Errorf("Expected NextReviewAt to be close to now, got %v", stats.NextReviewAt)
	}

	if stats.ReviewCount != 0 {
		t.Errorf("Expected review count 0, got %d", stats.ReviewCount)
	}

	if stats.CreatedAt.IsZero() {
		t.Error("Expected non-zero CreatedAt time")
	}

	if stats.UpdatedAt.IsZero() {
		t.Error("Expected non-zero UpdatedAt time")
	}

	// Test invalid userID
	_, err = NewUserCardStats(uuid.Nil, cardID)
	if err != ErrEmptyStatsUserID {
		t.Errorf("Expected error %v, got %v", ErrEmptyStatsUserID, err)
	}

	// Test invalid cardID
	_, err = NewUserCardStats(userID, uuid.Nil)
	if err != ErrEmptyStatsCardID {
		t.Errorf("Expected error %v, got %v", ErrEmptyStatsCardID, err)
	}
}

func TestUserCardStatsValidate(t *testing.T) {
	validStats := UserCardStats{
		UserID:     uuid.New(),
		CardID:     uuid.New(),
		Interval:   1,
		EaseFactor: 2.5,
	}

	// Test valid stats
	if err := validStats.Validate(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test invalid UserID
	invalidStats := validStats
	invalidStats.UserID = uuid.Nil
	if err := invalidStats.Validate(); err != ErrEmptyStatsUserID {
		t.Errorf("Expected error %v, got %v", ErrEmptyStatsUserID, err)
	}

	// Test invalid CardID
	invalidStats = validStats
	invalidStats.CardID = uuid.Nil
	if err := invalidStats.Validate(); err != ErrEmptyStatsCardID {
		t.Errorf("Expected error %v, got %v", ErrEmptyStatsCardID, err)
	}

	// Test invalid Interval
	invalidStats = validStats
	invalidStats.Interval = -1
	if err := invalidStats.Validate(); err != ErrInvalidInterval {
		t.Errorf("Expected error %v, got %v", ErrInvalidInterval, err)
	}

	// Test invalid EaseFactor
	invalidStats = validStats
	invalidStats.EaseFactor = 0.5
	if err := invalidStats.Validate(); err != ErrInvalidEaseFactor {
		t.Errorf("Expected error %v, got %v", ErrInvalidEaseFactor, err)
	}
}

func TestUpdateReview(t *testing.T) {
	userID := uuid.New()
	cardID := uuid.New()

	// Test new card with "again" outcome
	stats, _ := NewUserCardStats(userID, cardID)
	now := time.Now().UTC()
	time.Sleep(10 * time.Millisecond) // Ensure updated time is different

	err := stats.UpdateReview(ReviewOutcomeAgain)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if stats.ReviewCount != 1 {
		t.Errorf("Expected review count 1, got %d", stats.ReviewCount)
	}

	if stats.LastReviewedAt.Before(now) {
		t.Errorf("Expected LastReviewedAt to be after %v, got %v", now, stats.LastReviewedAt)
	}

	if stats.Interval != 0 {
		t.Errorf("Expected interval 0, got %d", stats.Interval)
	}

	if stats.EaseFactor != 2.3 { // 2.5 - 0.2
		t.Errorf("Expected ease factor 2.3, got %f", stats.EaseFactor)
	}

	if stats.ConsecutiveCorrect != 0 {
		t.Errorf("Expected consecutive correct 0, got %d", stats.ConsecutiveCorrect)
	}

	// Test intervals for different outcomes
	testCases := []struct {
		name          string
		startInterval int
		startEase     float64
		startConsec   int
		outcome       ReviewOutcome
		wantInterval  func(int) bool
		wantEase      func(float64) bool
		wantConsec    int
	}{
		{
			name:          "Again resets interval",
			startInterval: 10,
			startEase:     2.5,
			startConsec:   2,
			outcome:       ReviewOutcomeAgain,
			wantInterval:  func(i int) bool { return i == 0 },
			wantEase:      func(e float64) bool { return e < 2.5 },
			wantConsec:    0,
		},
		{
			name:          "Hard increases interval slightly",
			startInterval: 10,
			startEase:     2.5,
			startConsec:   2,
			outcome:       ReviewOutcomeHard,
			wantInterval:  func(i int) bool { return i > 10 && i < 20 },
			wantEase:      func(e float64) bool { return e < 2.5 },
			wantConsec:    3,
		},
		{
			name:          "Good increases interval normally",
			startInterval: 10,
			startEase:     2.5,
			startConsec:   2,
			outcome:       ReviewOutcomeGood,
			wantInterval:  func(i int) bool { return i > 20 },
			wantEase:      func(e float64) bool { return e == 2.5 },
			wantConsec:    3,
		},
		{
			name:          "Easy increases interval significantly",
			startInterval: 10,
			startEase:     2.3,
			startConsec:   2,
			outcome:       ReviewOutcomeEasy,
			wantInterval:  func(i int) bool { return i > 25 },
			wantEase:      func(e float64) bool { return e > 2.3 },
			wantConsec:    3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stats := UserCardStats{
				UserID:             userID,
				CardID:             cardID,
				Interval:           tc.startInterval,
				EaseFactor:         tc.startEase,
				ConsecutiveCorrect: tc.startConsec,
				ReviewCount:        5,
			}

			err := stats.UpdateReview(tc.outcome)

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if stats.ReviewCount != 6 {
				t.Errorf("Expected review count 6, got %d", stats.ReviewCount)
			}

			if !tc.wantInterval(stats.Interval) {
				t.Errorf("Interval %d didn't meet expectations for %s", stats.Interval, tc.name)
			}

			if !tc.wantEase(stats.EaseFactor) {
				t.Errorf("Ease %f didn't meet expectations for %s", stats.EaseFactor, tc.name)
			}

			if stats.ConsecutiveCorrect != tc.wantConsec {
				t.Errorf("Expected consecutive correct %d, got %d for %s",
					tc.wantConsec, stats.ConsecutiveCorrect, tc.name)
			}
		})
	}

	// Test invalid outcome
	stats, _ = NewUserCardStats(userID, cardID)
	err = stats.UpdateReview("invalid_outcome")

	if err != ErrInvalidReviewOutcome {
		t.Errorf("Expected error %v, got %v", ErrInvalidReviewOutcome, err)
	}
}

func TestPostponeReview(t *testing.T) {
	userID := uuid.New()
	cardID := uuid.New()

	stats, _ := NewUserCardStats(userID, cardID)
	originalNextReview := stats.NextReviewAt

	// Test valid postpone
	err := stats.PostponeReview(7)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedNext := originalNextReview.AddDate(0, 0, 7)
	if !stats.NextReviewAt.Equal(expectedNext) {
		t.Errorf("Expected next review at %v, got %v", expectedNext, stats.NextReviewAt)
	}

	// Test invalid postpone days
	err = stats.PostponeReview(0)
	if err == nil {
		t.Error("Expected error for zero postpone days, got nil")
	}

	err = stats.PostponeReview(-1)
	if err == nil {
		t.Error("Expected error for negative postpone days, got nil")
	}
}
