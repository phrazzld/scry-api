package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// MemoStatus represents the processing state of a memo
type MemoStatus string

// Possible memo status values
const (
	MemoStatusPending             MemoStatus = "pending"
	MemoStatusProcessing          MemoStatus = "processing"
	MemoStatusCompleted           MemoStatus = "completed"
	MemoStatusCompletedWithErrors MemoStatus = "completed_with_errors"
	MemoStatusFailed              MemoStatus = "failed"
)

// Memo-specific validation errors
var (
	// ErrMemoIDEmpty is returned when a memo ID is empty or nil.
	ErrMemoIDEmpty = errors.New("memo ID cannot be empty")

	// ErrMemoUserIDEmpty is returned when a memo's user ID is empty or nil.
	ErrMemoUserIDEmpty = errors.New("memo user ID cannot be empty")

	// ErrMemoTextEmpty is returned when a memo's text is empty.
	ErrMemoTextEmpty = errors.New("memo text cannot be empty")

	// ErrMemoStatusInvalid is returned when a memo status is not valid.
	ErrMemoStatusInvalid = errors.New("invalid memo status")
)

// Memo represents a text-based entry submitted by a user
// to generate flashcards. It tracks both the original content
// and the processing state.
type Memo struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Text      string     `json:"text"`
	Status    MemoStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// NewMemo creates a new Memo with the given user ID and text.
// It generates a new UUID for the memo ID, sets the status to pending,
// and sets the creation/update timestamps.
// Returns an error if validation fails.
func NewMemo(userID uuid.UUID, text string) (*Memo, error) {
	memo := &Memo{
		ID:        uuid.New(),
		UserID:    userID,
		Text:      text,
		Status:    MemoStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := memo.Validate(); err != nil {
		return nil, err
	}

	return memo, nil
}

// Validate checks if the Memo has valid data.
// Returns an error if any field fails validation.
func (m *Memo) Validate() error {
	if m.ID == uuid.Nil {
		return ErrMemoIDEmpty
	}

	if m.UserID == uuid.Nil {
		return ErrMemoUserIDEmpty
	}

	if m.Text == "" {
		return ErrMemoTextEmpty
	}

	if !isValidMemoStatus(m.Status) {
		return ErrMemoStatusInvalid
	}

	return nil
}

// UpdateStatus updates the memo's status and updates the UpdatedAt timestamp.
// Returns an error if the new status is invalid.
func (m *Memo) UpdateStatus(status MemoStatus) error {
	if !isValidMemoStatus(status) {
		return ErrMemoStatusInvalid
	}

	m.Status = status
	m.UpdatedAt = time.Now().UTC()
	return nil
}

// isValidMemoStatus checks if the given status is a valid MemoStatus.
func isValidMemoStatus(status MemoStatus) bool {
	switch status {
	case MemoStatusPending, MemoStatusProcessing, MemoStatusCompleted,
		MemoStatusCompletedWithErrors, MemoStatusFailed:
		return true
	default:
		return false
	}
}
