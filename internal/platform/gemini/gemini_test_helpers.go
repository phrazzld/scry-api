//go:build test_without_external_deps

package gemini

import (
	"context"
	"html/template"
	"log/slog"

	"github.com/phrazzld/scry-api/internal/config"
)

// NewTestableGenerator builds a mock GeminiGenerator
// by injecting an already-parsed template for testing.
func NewTestableGenerator(
	logger *slog.Logger,
	cfg config.LLMConfig,
	tmpl *template.Template,
) *GeminiGenerator {
	// Create a mock client
	mockClient := NewMockGenAIClient()

	return &GeminiGenerator{
		logger:         logger,
		config:         cfg,
		promptTemplate: tmpl,
		client:         mockClient,
		modelName:      cfg.ModelName,
	}
}

// CreatePromptForTest exposes createPrompt() for unit tests.
func CreatePromptForTest(
	g *GeminiGenerator,
	ctx context.Context,
	memoText string,
) (string, error) {
	return g.createPrompt(ctx, memoText)
}

// The test helper methods are intentionally limited to createPrompt
// as testing of callGeminiWithRetry and parseResponse happens through
// the GenerateCards method, which is the public API of the GeminiGenerator
