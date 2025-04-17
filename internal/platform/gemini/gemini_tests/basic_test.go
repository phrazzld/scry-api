// Package gemini_tests contains tests for gemini package functionality
// that do not require external dependencies
package gemini_tests

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test minimal test to verify the package builds and tests can run
func TestMinimal(t *testing.T) {
	t.Log("Minimal test running successfully")
}

// Test basic UUID functionality used by the package
func TestUUIDGeneration(t *testing.T) {
	id := uuid.New()
	assert.NotEqual(t, uuid.Nil, id, "UUID should not be nil")
}

// Test JSON marshaling for card content
func TestCardContentJSON(t *testing.T) {
	content := domain.CardContent{
		Front: "What is Go?",
		Back:  "A programming language",
		Hint:  "Created by Google",
		Tags:  []string{"programming", "languages"},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(content)
	require.NoError(t, err, "Marshal should not error")
	assert.NotEmpty(t, jsonData, "JSON data should not be empty")

	// Unmarshal back
	var decoded domain.CardContent
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err, "Unmarshal should not error")

	// Verify all fields
	assert.Equal(t, content.Front, decoded.Front, "Front field should match")
	assert.Equal(t, content.Back, decoded.Back, "Back field should match")
	assert.Equal(t, content.Hint, decoded.Hint, "Hint field should match")
	assert.Equal(t, content.Tags, decoded.Tags, "Tags field should match")
}

// Test card creation
func TestCardCreation(t *testing.T) {
	userID := uuid.New()
	memoID := uuid.New()

	cardContent := domain.CardContent{
		Front: "What is Go?",
		Back:  "A programming language",
	}

	contentJSON, err := json.Marshal(cardContent)
	require.NoError(t, err, "Failed to marshal card content")

	card, err := domain.NewCard(userID, memoID, contentJSON)
	require.NoError(t, err, "Failed to create card")

	assert.Equal(t, userID, card.UserID, "UserID should match")
	assert.Equal(t, memoID, card.MemoID, "MemoID should match")

	var parsedContent domain.CardContent
	err = json.Unmarshal(card.Content, &parsedContent)
	require.NoError(t, err, "Failed to unmarshal card content")

	assert.Equal(t, cardContent.Front, parsedContent.Front, "Front should match")
	assert.Equal(t, cardContent.Back, parsedContent.Back, "Back should match")
}
