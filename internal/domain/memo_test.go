package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewMemo(t *testing.T) {
	// Test valid memo creation
	userID := uuid.New()
	text := "This is a test memo for generating flashcards."

	memo, err := NewMemo(userID, text)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if memo.ID == uuid.Nil {
		t.Error("Expected non-nil UUID, got nil UUID")
	}

	if memo.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, memo.UserID)
	}

	if memo.Text != text {
		t.Errorf("Expected text %s, got %s", text, memo.Text)
	}

	if memo.Status != MemoStatusPending {
		t.Errorf("Expected status %s, got %s", MemoStatusPending, memo.Status)
	}

	if memo.CreatedAt.IsZero() {
		t.Error("Expected non-zero CreatedAt time")
	}

	if memo.UpdatedAt.IsZero() {
		t.Error("Expected non-zero UpdatedAt time")
	}

	// Test invalid userID
	_, err = NewMemo(uuid.Nil, text)
	if err != ErrEmptyMemoUserID {
		t.Errorf("Expected error %v, got %v", ErrEmptyMemoUserID, err)
	}

	// Test invalid text
	_, err = NewMemo(userID, "")
	if err != ErrEmptyMemoText {
		t.Errorf("Expected error %v, got %v", ErrEmptyMemoText, err)
	}
}

func TestMemoValidate(t *testing.T) {
	validMemo := Memo{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Text:   "Test memo",
		Status: MemoStatusPending,
	}

	// Test valid memo
	if err := validMemo.Validate(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test invalid ID
	invalidMemo := validMemo
	invalidMemo.ID = uuid.Nil
	if err := invalidMemo.Validate(); err != ErrEmptyMemoID {
		t.Errorf("Expected error %v, got %v", ErrEmptyMemoID, err)
	}

	// Test invalid UserID
	invalidMemo = validMemo
	invalidMemo.UserID = uuid.Nil
	if err := invalidMemo.Validate(); err != ErrEmptyMemoUserID {
		t.Errorf("Expected error %v, got %v", ErrEmptyMemoUserID, err)
	}

	// Test invalid Text
	invalidMemo = validMemo
	invalidMemo.Text = ""
	if err := invalidMemo.Validate(); err != ErrEmptyMemoText {
		t.Errorf("Expected error %v, got %v", ErrEmptyMemoText, err)
	}

	// Test invalid Status
	invalidMemo = validMemo
	invalidMemo.Status = "invalid_status"
	if err := invalidMemo.Validate(); err != ErrInvalidMemoStatus {
		t.Errorf("Expected error %v, got %v", ErrInvalidMemoStatus, err)
	}
}

func TestUpdateStatus(t *testing.T) {
	memo := Memo{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Text:   "Test memo",
		Status: MemoStatusPending,
	}

	// Test valid status update
	origUpdatedAt := memo.UpdatedAt
	err := memo.UpdateStatus(MemoStatusProcessing)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if memo.Status != MemoStatusProcessing {
		t.Errorf("Expected status %s, got %s", MemoStatusProcessing, memo.Status)
	}

	if !memo.UpdatedAt.After(origUpdatedAt) && !memo.UpdatedAt.Equal(origUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated")
	}

	// Test all valid status transitions
	validStatuses := []MemoStatus{
		MemoStatusPending,
		MemoStatusProcessing,
		MemoStatusCompleted,
		MemoStatusCompletedWithErrors,
		MemoStatusFailed,
	}

	for _, status := range validStatuses {
		err := memo.UpdateStatus(status)
		if err != nil {
			t.Errorf("Expected no error for status %s, got %v", status, err)
		}

		if memo.Status != status {
			t.Errorf("Expected status %s, got %s", status, memo.Status)
		}
	}

	// Test invalid status
	err = memo.UpdateStatus("invalid_status")
	if err != ErrInvalidMemoStatus {
		t.Errorf("Expected error %v, got %v", ErrInvalidMemoStatus, err)
	}
}
