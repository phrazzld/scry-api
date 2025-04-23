package domain

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Card-specific validation errors
var (
	// ErrCardIDEmpty is returned when a card ID is empty or nil.
	ErrCardIDEmpty = errors.New("card ID cannot be empty")

	// ErrCardUserIDEmpty is returned when a card's user ID is empty or nil.
	ErrCardUserIDEmpty = errors.New("card user ID cannot be empty")

	// ErrCardMemoIDEmpty is returned when a card's memo ID is empty or nil.
	ErrCardMemoIDEmpty = errors.New("card memo ID cannot be empty")

	// ErrCardContentEmpty is returned when a card's content is empty.
	ErrCardContentEmpty = errors.New("card content cannot be empty")

	// ErrCardContentInvalid is returned when a card's content is not valid JSON.
	ErrCardContentInvalid = errors.New("card content must be valid JSON")
)

// Card represents a flashcard generated from a user's memo.
// The content is stored as a JSONB structure, allowing for flexible
// card formats and future extensibility.
type Card struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	MemoID    uuid.UUID       `json:"memo_id"`
	Content   json.RawMessage `json:"content"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// CardContent represents the structure of the content field in a Card.
// This is provided as a sample structure but cards can have flexible content
// as it's stored as a JSONB field.
type CardContent struct {
	Front    string   `json:"front"`
	Back     string   `json:"back"`
	Hint     string   `json:"hint,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	ImageURL string   `json:"image_url,omitempty"`
}

// NewCard creates a new Card with the given user ID, memo ID, and content.
// It generates a new UUID for the card ID and sets the creation/update timestamps.
// Returns an error if validation fails.
func NewCard(userID, memoID uuid.UUID, content json.RawMessage) (*Card, error) {
	card := &Card{
		ID:        uuid.New(),
		UserID:    userID,
		MemoID:    memoID,
		Content:   content,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := card.Validate(); err != nil {
		return nil, err
	}

	return card, nil
}

// Validate checks if the Card has valid data.
// Returns an error if any field fails validation.
func (c *Card) Validate() error {
	if c.ID == uuid.Nil {
		return ErrCardIDEmpty
	}

	if c.UserID == uuid.Nil {
		return ErrCardUserIDEmpty
	}

	if c.MemoID == uuid.Nil {
		return ErrCardMemoIDEmpty
	}

	if len(c.Content) == 0 {
		return ErrCardContentEmpty
	}

	// Check if content is valid JSON
	var js json.RawMessage
	if err := json.Unmarshal(c.Content, &js); err != nil {
		return ErrCardContentInvalid
	}

	return nil
}

// UpdateContent updates the card's content and updates the UpdatedAt timestamp.
// Returns an error if the new content is invalid.
func (c *Card) UpdateContent(content json.RawMessage) error {
	// Temporarily update content to validate
	origContent := c.Content
	c.Content = content

	// Check validity
	if err := c.Validate(); err != nil {
		// Restore original content if invalid
		c.Content = origContent
		return err
	}

	// Update timestamp
	c.UpdatedAt = time.Now().UTC()
	return nil
}
