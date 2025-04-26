package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestNewCard(t *testing.T) {
	t.Parallel() // Enable parallel execution
	// Test valid card creation
	userID := uuid.New()
	memoID := uuid.New()
	content := json.RawMessage(`{"front": "What is Go?", "back": "A programming language"}`)

	card, err := NewCard(userID, memoID, content)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if card.ID == uuid.Nil {
		t.Error("Expected non-nil UUID, got nil UUID")
	}

	if card.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, card.UserID)
	}

	if card.MemoID != memoID {
		t.Errorf("Expected memo ID %s, got %s", memoID, card.MemoID)
	}

	if string(card.Content) != string(content) {
		t.Errorf("Expected content %s, got %s", string(content), string(card.Content))
	}

	if card.CreatedAt.IsZero() {
		t.Error("Expected non-zero CreatedAt time")
	}

	if card.UpdatedAt.IsZero() {
		t.Error("Expected non-zero UpdatedAt time")
	}

	// Test invalid userID
	_, err = NewCard(uuid.Nil, memoID, content)
	if err != ErrCardUserIDEmpty {
		t.Errorf("Expected error %v, got %v", ErrCardUserIDEmpty, err)
	}

	// Test invalid memoID
	_, err = NewCard(userID, uuid.Nil, content)
	if err != ErrCardMemoIDEmpty {
		t.Errorf("Expected error %v, got %v", ErrCardMemoIDEmpty, err)
	}

	// Test invalid content
	_, err = NewCard(userID, memoID, nil)
	if err != ErrCardContentEmpty {
		t.Errorf("Expected error %v, got %v", ErrCardContentEmpty, err)
	}

	// Test invalid JSON content
	invalidJSON := json.RawMessage(`{"front": "broken JSON`)
	_, err = NewCard(userID, memoID, invalidJSON)
	if err != ErrCardContentInvalid {
		t.Errorf("Expected error %v, got %v", ErrCardContentInvalid, err)
	}
}

func TestCardValidate(t *testing.T) {
	t.Parallel() // Enable parallel execution
	validCard := Card{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		MemoID:  uuid.New(),
		Content: json.RawMessage(`{"front": "What is Go?", "back": "A programming language"}`),
	}

	// Test valid card
	if err := validCard.Validate(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test invalid ID
	invalidCard := validCard
	invalidCard.ID = uuid.Nil
	if err := invalidCard.Validate(); err != ErrCardIDEmpty {
		t.Errorf("Expected error %v, got %v", ErrCardIDEmpty, err)
	}

	// Test invalid UserID
	invalidCard = validCard
	invalidCard.UserID = uuid.Nil
	if err := invalidCard.Validate(); err != ErrCardUserIDEmpty {
		t.Errorf("Expected error %v, got %v", ErrCardUserIDEmpty, err)
	}

	// Test invalid MemoID
	invalidCard = validCard
	invalidCard.MemoID = uuid.Nil
	if err := invalidCard.Validate(); err != ErrCardMemoIDEmpty {
		t.Errorf("Expected error %v, got %v", ErrCardMemoIDEmpty, err)
	}

	// Test empty Content
	invalidCard = validCard
	invalidCard.Content = nil
	if err := invalidCard.Validate(); err != ErrCardContentEmpty {
		t.Errorf("Expected error %v, got %v", ErrCardContentEmpty, err)
	}

	// Test invalid JSON Content
	invalidCard = validCard
	invalidCard.Content = json.RawMessage(`{"front": "broken JSON`)
	if err := invalidCard.Validate(); err != ErrCardContentInvalid {
		t.Errorf("Expected error %v, got %v", ErrCardContentInvalid, err)
	}
}

func TestUpdateContent(t *testing.T) {
	t.Parallel() // Enable parallel execution
	card := Card{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		MemoID:  uuid.New(),
		Content: json.RawMessage(`{"front": "What is Go?", "back": "A programming language"}`),
	}

	// Test valid content update
	newContent := json.RawMessage(
		`{"front": "What is Python?", "back": "Another programming language"}`,
	)
	origUpdatedAt := card.UpdatedAt

	err := card.UpdateContent(newContent)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if string(card.Content) != string(newContent) {
		t.Errorf("Expected content %s, got %s", string(newContent), string(card.Content))
	}

	if !card.UpdatedAt.After(origUpdatedAt) && !card.UpdatedAt.Equal(origUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated")
	}

	// Test invalid content update
	invalidContent := json.RawMessage(`{"front": "broken JSON`)
	originalContent := card.Content

	err = card.UpdateContent(invalidContent)

	if err != ErrCardContentInvalid {
		t.Errorf("Expected error %v, got %v", ErrCardContentInvalid, err)
	}

	if string(card.Content) != string(originalContent) {
		t.Errorf("Expected content to remain unchanged at %s, got %s",
			string(originalContent), string(card.Content))
	}
}
