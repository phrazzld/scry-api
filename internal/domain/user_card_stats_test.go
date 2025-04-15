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

// Note: The TestUpdateReview and TestPostponeReview tests have been removed
// since the mutable methods they were testing have been removed.
// The functionality is now tested in the srs/service_test.go file
// which tests the immutable srs.Service methods that replace them.
