//go:build test_without_external_deps
// +build test_without_external_deps

package gemini_test

import (
	"context"
	"html/template"
	"log/slog"
	"strings"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/gemini"
)

// TestableGeminiGenerator is a generator for testing
// that doesn't require the actual Google dependencies
type TestableGeminiGenerator struct {
	Logger         *slog.Logger
	Config         config.LLMConfig
	PromptTemplate *template.Template
}

// NewTestableGenerator creates a new GeminiGenerator for testing
func NewTestableGenerator(
	logger *slog.Logger,
	config config.LLMConfig,
	tmpl *template.Template,
) *TestableGeminiGenerator {
	return &TestableGeminiGenerator{
		Logger:         logger,
		Config:         config,
		PromptTemplate: tmpl,
	}
}

// CreatePromptForTest is a helper to test the createPrompt functionality
func CreatePromptForTest(g *TestableGeminiGenerator, ctx context.Context, memoText string) (string, error) {
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
