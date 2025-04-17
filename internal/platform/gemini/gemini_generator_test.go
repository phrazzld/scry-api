//go:build test_without_external_deps
// +build test_without_external_deps

package gemini_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/generation"
	"github.com/phrazzld/scry-api/internal/platform/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test utilities

// createTempTemplateFile creates a temporary template file for testing
func createTempTemplateFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test-prompt.tmpl")
	err := os.WriteFile(path, []byte(content), 0600)
	require.NoError(t, err, "Failed to create temp template file")
	return path
}

// newTestLogger creates a logger for testing
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// Basic test to verify minimal functionality
func TestMinimal(t *testing.T) {
	t.Log("Minimal test running successfully")
}

// Test ResponseSchema parsing
func TestResponseSchema(t *testing.T) {
	// Test JSON marshaling/unmarshaling of ResponseSchema
	expected := &gemini.ResponseSchema{
		Cards: []gemini.CardSchema{
			{
				Front: "What is Go?",
				Back:  "A programming language",
				Hint:  "Created by Google",
				Tags:  []string{"programming", "languages"},
			},
		},
	}

	jsonData, err := json.Marshal(expected)
	require.NoError(t, err, "Failed to marshal ResponseSchema")

	var actual gemini.ResponseSchema
	err = json.Unmarshal(jsonData, &actual)
	require.NoError(t, err, "Failed to unmarshal ResponseSchema")

	assert.Equal(t, 1, len(actual.Cards), "Should have 1 card")
	assert.Equal(t, "What is Go?", actual.Cards[0].Front, "Front should match")
	assert.Equal(t, "A programming language", actual.Cards[0].Back, "Back should match")
	assert.Equal(t, "Created by Google", actual.Cards[0].Hint, "Hint should match")
	assert.Equal(t, []string{"programming", "languages"}, actual.Cards[0].Tags, "Tags should match")
}

// Test Card creation from ResponseSchema
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

// Test template execution (similar to createPrompt functionality)
func TestCreatePrompt(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	templateContent := "Generate flashcards for: {{.MemoText}}"
	tmpl, err := template.New("test").Parse(templateContent)
	require.NoError(t, err, "Failed to parse template")

	testConfig := config.LLMConfig{
		GeminiAPIKey:       "test-api-key",
		ModelName:          "test-model",
		PromptTemplatePath: "test-template.txt",
		MaxRetries:         3,
		RetryDelaySeconds:  2,
	}

	generator := NewTestableGenerator(logger, testConfig, tmpl)

	// Test valid memo text
	memoText := "This is a test memo."
	prompt, err := CreatePromptForTest(generator, ctx, memoText)
	require.NoError(t, err, "Failed to create prompt")
	assert.Equal(t, "Generate flashcards for: This is a test memo.", prompt, "Prompt should match expected output")

	// Test empty memo text
	prompt, err = CreatePromptForTest(generator, ctx, "")
	assert.Error(t, err, "Should error with empty memo text")
	assert.Equal(t, gemini.ErrEmptyMemoText, err, "Should return ErrEmptyMemoText for empty memo")
	assert.Empty(t, prompt, "Prompt should be empty on error")
}

// Test error wrapping in the generation package
func TestErrorWrapping(t *testing.T) {
	// Test wrapping ErrGenerationFailed
	origErr := errors.New("some underlying error")
	wrappedErr := fmt.Errorf("%w: %v", generation.ErrGenerationFailed, origErr)

	assert.True(t, errors.Is(wrappedErr, generation.ErrGenerationFailed), "Wrapped error should be ErrGenerationFailed")
	assert.Contains(t, wrappedErr.Error(), origErr.Error(), "Wrapped error should contain the original error")
}

// Note: The remaining tests that require the actual Gemini API client
// will be implemented once all dependencies are properly resolved
// These include:
// - TestNewGeminiGenerator
// - TestCallGeminiWithRetry
// - TestParseResponse
// - TestGenerateCards
