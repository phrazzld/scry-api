//go:build test_without_external_deps

package gemini_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/generation"
	"github.com/phrazzld/scry-api/internal/platform/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test utilities

// newTestLogger creates a logger for testing
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// newTestTemplate creates a template for testing
func newTestTemplate() *template.Template {
	tmpl, _ := template.New("test").Parse("Generate flashcards for: {{.MemoText}}")
	return tmpl
}

// newTestConfig creates a configuration for testing
func newTestConfig() config.LLMConfig {
	return config.LLMConfig{
		GeminiAPIKey:       "test-api-key",
		ModelName:          "test-model",
		PromptTemplatePath: "test-prompt-template.txt",
		MaxRetries:         3,
		RetryDelaySeconds:  1,
	}
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

// Test createPrompt functionality
func TestCreatePrompt(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	templateContent := "Generate flashcards for: {{.MemoText}}"
	tmpl, err := template.New("test").Parse(templateContent)
	require.NoError(t, err, "Failed to parse template")

	testConfig := newTestConfig()

	generator := gemini.NewTestableGenerator(logger, testConfig, tmpl)

	// Test valid memo text
	memoText := "This is a test memo."
	prompt, err := gemini.CreatePromptForTest(generator, ctx, memoText)
	require.NoError(t, err, "Failed to create prompt")
	assert.Equal(
		t,
		"Generate flashcards for: This is a test memo.",
		prompt,
		"Prompt should match expected output",
	)

	// Test empty memo text
	prompt, err = gemini.CreatePromptForTest(generator, ctx, "")
	assert.Error(t, err, "Should error with empty memo text")
	assert.Equal(t, gemini.ErrEmptyMemoText, err, "Should return ErrEmptyMemoText for empty memo")
	assert.Empty(t, prompt, "Prompt should be empty on error")
}

// Test GenerateCards functionality with the mock implementation
func TestGenerateCards_HappyPath(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()
	tmpl := newTestTemplate()
	testConfig := newTestConfig()

	generator := gemini.NewTestableGenerator(logger, testConfig, tmpl)

	// Test with valid inputs
	memoText := "This is a test memo about Go programming language."
	userID := uuid.New()

	cards, err := generator.GenerateCards(ctx, memoText, userID)
	require.NoError(t, err, "GenerateCards should not fail with valid inputs")
	require.NotNil(t, cards, "Cards should not be nil")
	require.NotEmpty(t, cards, "Cards should not be empty")

	// Verify card structure
	for _, card := range cards {
		assert.NotEqual(t, uuid.Nil, card.ID, "Card ID should not be nil")
		assert.Equal(t, userID, card.UserID, "Card should have correct user ID")
		assert.NotEqual(t, uuid.Nil, card.MemoID, "Card memo ID should not be nil")

		// Check card content
		var content domain.CardContent
		err := json.Unmarshal(card.Content, &content)
		require.NoError(t, err, "Should be able to unmarshal card content")

		assert.NotEmpty(t, content.Front, "Card front should not be empty")
		assert.NotEmpty(t, content.Back, "Card back should not be empty")
	}
}

// Test GenerateCards with empty memo text
func TestGenerateCards_EmptyMemoText(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()
	tmpl := newTestTemplate()
	testConfig := newTestConfig()

	generator := gemini.NewTestableGenerator(logger, testConfig, tmpl)

	// Test with empty memo text
	userID := uuid.New()

	cards, err := generator.GenerateCards(ctx, "", userID)
	assert.Error(t, err, "GenerateCards should fail with empty memo text")
	assert.Equal(t, gemini.ErrEmptyMemoText, err, "Error should be ErrEmptyMemoText")
	assert.Nil(t, cards, "Cards should be nil")
}

// Test GenerateCards with nil user ID
func TestGenerateCards_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()
	tmpl := newTestTemplate()
	testConfig := newTestConfig()

	generator := gemini.NewTestableGenerator(logger, testConfig, tmpl)

	// Test with nil user ID
	memoText := "This is a test memo."

	cards, err := generator.GenerateCards(ctx, memoText, uuid.Nil)
	assert.Error(t, err, "GenerateCards should fail with nil user ID")
	assert.Contains(t, err.Error(), "user ID cannot be empty")
	assert.Nil(t, cards, "Cards should be nil")
}

// Test GenerateCards error propagation for specific error types
func TestGenerateCards_ErrorPropagation(t *testing.T) {
	// Define test cases for each error type
	testCases := []struct {
		name          string
		mockError     error
		expectedError error
	}{
		{
			name:          "content blocked error",
			mockError:     generation.ErrContentBlocked,
			expectedError: generation.ErrContentBlocked,
		},
		{
			name:          "invalid response error",
			mockError:     generation.ErrInvalidResponse,
			expectedError: generation.ErrInvalidResponse,
		},
		{
			name:          "transient failure error",
			mockError:     generation.ErrTransientFailure,
			expectedError: generation.ErrTransientFailure,
		},
		{
			name: "wrapped content blocked error",
			mockError: fmt.Errorf(
				"%w: content blocked by safety filters",
				generation.ErrContentBlocked,
			),
			expectedError: generation.ErrContentBlocked,
		},
		{
			name:          "wrapped invalid response error",
			mockError:     fmt.Errorf("%w: malformed response", generation.ErrInvalidResponse),
			expectedError: generation.ErrInvalidResponse,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test environment
			ctx := context.Background()
			logger := newTestLogger()
			tmpl := newTestTemplate()
			testConfig := newTestConfig()

			generator := gemini.NewTestableGenerator(logger, testConfig, tmpl)

			// Configure the mock client to return the specific error
			mockClient := generator.Client()
			mockClient.SetErrorToReturn(tc.mockError)

			// Call GenerateCards
			userID := uuid.New()
			memoText := "This is a test memo for error propagation."
			cards, err := generator.GenerateCards(ctx, memoText, userID)

			// Verify the error is of the expected type
			assert.Error(t, err, "GenerateCards should fail with the configured error")
			assert.True(t, errors.Is(err, tc.expectedError),
				"Error should be or wrap %v, got %v", tc.expectedError, err)
			assert.Nil(t, cards, "Cards should be nil when an error occurs")
		})
	}
}

// Test error wrapping in the generation package
func TestErrorWrapping(t *testing.T) {
	// Test wrapping ErrGenerationFailed
	origErr := errors.New("some underlying error")
	wrappedErr := fmt.Errorf("%w: %v", generation.ErrGenerationFailed, origErr)

	assert.True(
		t,
		errors.Is(wrappedErr, generation.ErrGenerationFailed),
		"Wrapped error should be ErrGenerationFailed",
	)
	assert.Contains(
		t,
		wrappedErr.Error(),
		origErr.Error(),
		"Wrapped error should contain the original error",
	)
}

// Test card content with various fields
func TestCardContent_AllFields(t *testing.T) {
	// Create card content with all fields
	expected := domain.CardContent{
		Front:    "What is Go?",
		Back:     "A programming language",
		Hint:     "Created by Google",
		Tags:     []string{"programming", "languages"},
		ImageURL: "https://example.com/go.png",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(expected)
	require.NoError(t, err, "Failed to marshal card content")

	// Unmarshal back
	var actual domain.CardContent
	err = json.Unmarshal(jsonData, &actual)
	require.NoError(t, err, "Failed to unmarshal card content")

	// Verify fields
	assert.Equal(t, expected.Front, actual.Front, "Front should match")
	assert.Equal(t, expected.Back, actual.Back, "Back should match")
	assert.Equal(t, expected.Hint, actual.Hint, "Hint should match")
	assert.Equal(t, expected.Tags, actual.Tags, "Tags should match")
	assert.Equal(t, expected.ImageURL, actual.ImageURL, "ImageURL should match")
}

// Test card timestamps
func TestCard_Timestamps(t *testing.T) {
	userID := uuid.New()
	memoID := uuid.New()
	content := []byte(`{"front":"test","back":"test"}`)

	before := time.Now().UTC().Add(-time.Second)
	card, err := domain.NewCard(userID, memoID, content)
	require.NoError(t, err, "Failed to create card")
	after := time.Now().UTC().Add(time.Second)

	// Timestamps should be between before and after
	assert.True(t, (card.CreatedAt.After(before) || card.CreatedAt.Equal(before)) &&
		(card.CreatedAt.Before(after) || card.CreatedAt.Equal(after)),
		"CreatedAt should be around now")

	assert.True(t, (card.UpdatedAt.After(before) || card.UpdatedAt.Equal(before)) &&
		(card.UpdatedAt.Before(after) || card.UpdatedAt.Equal(after)),
		"UpdatedAt should be around now")
}

// Test card validation failures
func TestCard_ValidationFailures(t *testing.T) {
	// Test nil user ID
	_, err := domain.NewCard(uuid.Nil, uuid.New(), []byte(`{"front":"test","back":"test"}`))
	assert.Error(t, err, "Should error with nil user ID")
	assert.Equal(t, domain.ErrCardUserIDEmpty, err, "Error should be ErrCardUserIDEmpty")

	// Test nil memo ID
	_, err = domain.NewCard(uuid.New(), uuid.Nil, []byte(`{"front":"test","back":"test"}`))
	assert.Error(t, err, "Should error with nil memo ID")
	assert.Equal(t, domain.ErrCardMemoIDEmpty, err, "Error should be ErrCardMemoIDEmpty")

	// Test empty content
	_, err = domain.NewCard(uuid.New(), uuid.New(), []byte{})
	assert.Error(t, err, "Should error with empty content")
	assert.Equal(t, domain.ErrCardContentEmpty, err, "Error should be ErrCardContentEmpty")

	// Test invalid JSON content
	_, err = domain.NewCard(uuid.New(), uuid.New(), []byte("not json"))
	assert.Error(t, err, "Should error with invalid JSON content")
	assert.Equal(t, domain.ErrCardContentInvalid, err, "Error should be ErrCardContentInvalid")
}

// Test error handling
func TestErrorHandling(t *testing.T) {
	// Test that generation.ErrContentBlocked is properly wrapped
	contentBlockedErr := fmt.Errorf(
		"%w: content blocked by safety filters",
		generation.ErrContentBlocked,
	)
	assert.True(
		t,
		errors.Is(contentBlockedErr, generation.ErrContentBlocked),
		"Error should be ErrContentBlocked",
	)

	// Test that generation.ErrGenerationFailed is properly wrapped
	genFailedErr := fmt.Errorf("%w: some underlying error", generation.ErrGenerationFailed)
	assert.True(
		t,
		errors.Is(genFailedErr, generation.ErrGenerationFailed),
		"Error should be ErrGenerationFailed",
	)
}
