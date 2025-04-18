//go:build test_without_external_deps
// +build test_without_external_deps

package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"math/rand"
	"os"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/generation"
)

// GeminiGenerator implements the generation.Generator interface using
// an in-memory implementation for testing purposes.
type GeminiGenerator struct {
	// logger is used for structured logging
	logger *slog.Logger

	// config contains LLM-specific configuration
	config config.LLMConfig

	// promptTemplate is the parsed template for creating prompts
	promptTemplate *template.Template
}

// NewGeminiGenerator creates a new instance of GeminiGenerator with the provided dependencies.
// This is a mock implementation for testing purposes that doesn't require external API access.
//
// Parameters:
//   - ctx: Context for the operation, which can be used for cancellation
//   - logger: A structured logger for operation logging
//   - config: LLM configuration containing API key, model name, and other settings
//
// Returns:
//   - A properly initialized GeminiGenerator or an error if initialization fails
func NewGeminiGenerator(ctx context.Context, logger *slog.Logger, config config.LLMConfig) (*GeminiGenerator, error) {
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	// In test mode, we don't validate API key requirements
	// Instead we just check that the prompt template path is valid

	if config.PromptTemplatePath == "" {
		return nil, fmt.Errorf("%w: prompt template path cannot be empty", generation.ErrInvalidConfig)
	}

	// Load and parse prompt template
	templateContent, err := os.ReadFile(config.PromptTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read prompt template from %s: %v",
			generation.ErrInvalidConfig, config.PromptTemplatePath, err)
	}

	promptTemplate, err := template.New("flashcard").Parse(string(templateContent))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse prompt template: %v",
			generation.ErrInvalidConfig, err)
	}

	generator := &GeminiGenerator{
		logger:         logger,
		config:         config,
		promptTemplate: promptTemplate,
	}

	return generator, nil
}

// createPrompt generates a prompt string from the template with the provided memo text.
//
// It executes the template with the memo text and returns the resulting string.
// If the memo text is empty or the template execution fails, it returns an error.
//
// Parameters:
//   - ctx: Context for the operation, which can be used for logging
//   - memoText: The text of the memo to include in the prompt
//
// Returns:
//   - The generated prompt string
//   - An error if the memo text is empty or the template execution fails
func (g *GeminiGenerator) createPrompt(ctx context.Context, memoText string) (string, error) {
	// Validate input
	if memoText == "" {
		return "", ErrEmptyMemoText
	}

	// Create data for template
	data := promptData{
		MemoText: memoText,
	}

	g.logger.DebugContext(ctx, "Generating prompt from template",
		"memo_length", len(memoText),
		"template_name", g.promptTemplate.Name())

	// Execute template
	var promptBuffer bytes.Buffer
	if err := g.promptTemplate.Execute(&promptBuffer, data); err != nil {
		return "", fmt.Errorf("failed to execute prompt template: %w", err)
	}

	prompt := promptBuffer.String()
	g.logger.DebugContext(ctx, "Prompt generated successfully",
		"prompt_length", len(prompt))

	return prompt, nil
}

// GenerateCards creates mock flashcards based on the provided memo text and user ID.
// This is a test implementation that doesn't require external API access.
//
// Parameters:
//   - ctx: Context for the operation, which can be used for cancellation
//   - memoText: The content of the memo to generate cards from
//   - userID: The UUID of the user who owns the memo
//
// Returns:
//   - A slice of domain.Card pointers representing the generated flashcards
//   - An error if the generation fails for any reason
func (g *GeminiGenerator) GenerateCards(
	ctx context.Context,
	memoText string,
	userID uuid.UUID,
) ([]*domain.Card, error) {
	// Validate inputs
	if memoText == "" {
		return nil, ErrEmptyMemoText
	}

	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be empty")
	}

	g.logger.InfoContext(ctx, "Starting mock flashcard generation",
		"memo_length", len(memoText),
		"user_id", userID.String())

	// Create a prompt just for logging purposes
	_, err := g.createPrompt(ctx, memoText)
	if err != nil {
		g.logger.ErrorContext(ctx, "Failed to create prompt", "error", err)
		return nil, fmt.Errorf("%w: %v", generation.ErrGenerationFailed, err)
	}

	// Generate deterministic but random-looking cards
	// This uses the content of the memo to seed the random generator
	// to ensure consistent outputs for the same input
	seed := int64(len(memoText))
	for i := 0; i < len(memoText) && i < 20; i++ {
		seed += int64(memoText[i])
	}
	rng := rand.New(rand.NewSource(seed))

	// Extract words from the memo to use in the cards
	words := extractKeywords(memoText)

	// Create mock card data
	var cards []*domain.Card
	cardCount := 1 + rng.Intn(3) // Generate 1-3 cards

	for i := 0; i < cardCount && i < len(words); i++ {
		// Create a mock card with content using words from the memo
		cardContent := domain.CardContent{
			Front: fmt.Sprintf("What is the definition of '%s'?", words[i]),
			Back:  fmt.Sprintf("This is a mock definition of '%s' generated for testing purposes.", words[i]),
		}

		// Add optional fields occasionally
		if rng.Intn(2) == 0 {
			cardContent.Hint = "This is a mock hint."
		}

		if rng.Intn(2) == 0 {
			cardContent.Tags = []string{"mock", "test", "flashcard"}
		}

		// Convert to JSON
		contentJSON, err := json.Marshal(cardContent)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal card content to JSON: %w", err)
		}

		// Create mock memo ID (in real implementation this would be provided)
		memoID := uuid.New()

		// Create domain.Card
		card, err := domain.NewCard(userID, memoID, contentJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to create card: %w", err)
		}

		cards = append(cards, card)
	}

	g.logger.InfoContext(ctx, "Successfully generated mock flashcards",
		"card_count", len(cards),
		"user_id", userID.String())

	return cards, nil
}

// extractKeywords is a helper function that extracts words from the memo text
// for use in mock flashcards.
func extractKeywords(text string) []string {
	// This is a simple implementation for testing - in a real application
	// you might want to implement actual keyword extraction
	words := []string{"test", "mock", "implementation", "flashcard", "gemini"}

	// If we have actual text, use some words from it
	if len(text) > 10 {
		// Pick a few words from the text as "keywords"
		// This is extremely simplified and just for testing
		customWords := []string{}
		start := 0

		for i := 0; i < len(text); i++ {
			if text[i] == ' ' || text[i] == '.' || text[i] == ',' || text[i] == '\n' {
				if i-start > 4 { // Only use words longer than 4 characters
					word := text[start:i]
					customWords = append(customWords, word)
				}
				start = i + 1
			}

			if len(customWords) >= 5 {
				break
			}
		}

		if len(customWords) > 0 {
			return customWords
		}
	}

	return words
}
