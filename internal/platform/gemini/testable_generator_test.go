//go:build test_without_external_deps

package gemini_test

import (
	"context"
	"html/template"
	"log/slog"
	"strings"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/gemini"
)

// TestableGeminiGenerator is a test implementation that mimics GeminiGenerator
// for use in testing without requiring actual Google Gemini API dependencies.
// It uses a mock GenAI client to simulate API responses in a controlled way.
type TestableGeminiGenerator struct {
	Logger         *slog.Logger
	Config         config.LLMConfig
	PromptTemplate *template.Template
	mockClient     *gemini.MockGenAIClient
}

// NewTestableGenerator creates a new TestableGeminiGenerator for testing.
// This generator uses a MockGenAIClient internally to simulate the Gemini API,
// allowing tests to run without external dependencies and with controlled responses.
func NewTestableGenerator(
	logger *slog.Logger,
	config config.LLMConfig,
	tmpl *template.Template,
) *TestableGeminiGenerator {
	mockClient := gemini.NewMockGenAIClient()
	return &TestableGeminiGenerator{
		Logger:         logger,
		Config:         config,
		PromptTemplate: tmpl,
		mockClient:     mockClient,
	}
}

// Client returns the mock GenAIClient used by this generator.
// This provides access to the mock for test setup, like configuring
// expected responses or simulating errors.
func (g *TestableGeminiGenerator) Client() *gemini.MockGenAIClient {
	return g.mockClient
}

// CreatePromptForTest is a test helper function that exposes the prompt creation logic
// normally internal to the GeminiGenerator. It formats the provided memo text
// using the configured template.
//
// Parameters:
//   - g: The TestableGeminiGenerator instance
//   - ctx: Context for the operation
//   - memoText: The memo text to format into a prompt
//
// Returns:
//   - The formatted prompt string
//   - An error if formatting fails or if memoText is empty
func CreatePromptForTest(
	g *TestableGeminiGenerator,
	ctx context.Context,
	memoText string,
) (string, error) {
	if memoText == "" {
		return "", gemini.ErrEmptyMemoText
	}

	// Create data for template
	data := struct{ MemoText string }{MemoText: memoText}

	// Execute template
	var promptBuffer strings.Builder
	if err := g.PromptTemplate.Execute(&promptBuffer, data); err != nil {
		return "", err
	}

	return promptBuffer.String(), nil
}
