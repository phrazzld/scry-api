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

	// client is the mock client for the Gemini API
	client *MockGenAIClient

	// modelName is the name of the Gemini model to use
	modelName string
}

// Client returns the mock client for testing
func (g *GeminiGenerator) Client() *MockGenAIClient {
	return g.client
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

	// In test mode, we still validate the prompt template path
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

	mockClient := NewMockGenAIClient()

	generator := &GeminiGenerator{
		logger:         logger,
		config:         config,
		promptTemplate: promptTemplate,
		client:         mockClient,
		modelName:      config.ModelName,
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

// parseResponse converts a ResponseSchema from the mock API into domain.Card objects.
//
// It validates each card in the response and creates domain.Card objects with
// properly formatted content. If any card in the response fails validation, the
// method returns an error and no cards are returned.
//
// Parameters:
//   - ctx: Context for the operation, which can be used for logging
//   - response: The structured response from the mock API
//   - userID: The UUID of the user who owns the memo
//   - memoID: The UUID of the memo from which the cards are generated
//
// Returns:
//   - A slice of domain.Card pointers
//   - An error if the response is invalid or card creation fails
func (g *GeminiGenerator) parseResponse(
	ctx context.Context,
	response *ResponseSchema,
	userID uuid.UUID,
	memoID uuid.UUID,
) ([]*domain.Card, error) {
	// Validate input
	if response == nil {
		return nil, fmt.Errorf("%w: response is nil", generation.ErrInvalidResponse)
	}

	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be empty")
	}

	if memoID == uuid.Nil {
		return nil, errors.New("memo ID cannot be empty")
	}

	// Check if we have any cards
	if len(response.Cards) == 0 {
		return nil, fmt.Errorf("%w: no cards in response", generation.ErrInvalidResponse)
	}

	g.logger.InfoContext(ctx, "Parsing mock API response",
		"card_count", len(response.Cards),
		"user_id", userID.String(),
		"memo_id", memoID.String())

	// Create domain cards from response
	cards := make([]*domain.Card, 0, len(response.Cards))
	for i, cardSchema := range response.Cards {
		// Validate required fields
		if cardSchema.Front == "" {
			return nil, fmt.Errorf("%w: card %d missing front side", generation.ErrInvalidResponse, i)
		}

		if cardSchema.Back == "" {
			return nil, fmt.Errorf("%w: card %d missing back side", generation.ErrInvalidResponse, i)
		}

		// Create domain.CardContent structure
		cardContent := domain.CardContent{
			Front: cardSchema.Front,
			Back:  cardSchema.Back,
			Hint:  cardSchema.Hint,
			Tags:  cardSchema.Tags,
		}

		// Convert to JSON
		contentJSON, err := json.Marshal(cardContent)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal card content to JSON: %w", err)
		}

		// Create domain.Card
		card, err := domain.NewCard(userID, memoID, contentJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to create card: %w", err)
		}

		cards = append(cards, card)
		g.logger.DebugContext(ctx, "Created card from mock API response",
			"card_id", card.ID.String(),
			"front_length", len(cardSchema.Front),
			"back_length", len(cardSchema.Back))
	}

	g.logger.InfoContext(ctx, "Successfully parsed mock API response",
		"created_cards", len(cards))

	return cards, nil
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

	// Step 1: Create prompt from memo text
	prompt, err := g.createPrompt(ctx, memoText)
	if err != nil {
		g.logger.ErrorContext(ctx, "Failed to create prompt",
			"error", err)
		return nil, fmt.Errorf("%w: %v", generation.ErrGenerationFailed, err)
	}

	// Step 2: Use our simplified mock client to simulate API call
	response, err := g.client.MockGenerateContent(ctx, prompt)
	if err != nil {
		g.logger.ErrorContext(ctx, "Mock API call failed",
			"error", err)
		return nil, err
	}

	// In a production environment, the memoID would typically be provided by the caller
	// since it would be stored in the database. For this implementation, we'll
	// generate a new ID since we're focused on the generation logic.
	memoID := uuid.New()

	// Step 3: Parse response into domain.Card objects
	cards, err := g.parseResponse(ctx, response, userID, memoID)
	if err != nil {
		g.logger.ErrorContext(ctx, "Failed to parse mock API response",
			"error", err)
		return nil, fmt.Errorf("%w: %v", generation.ErrGenerationFailed, err)
	}

	g.logger.InfoContext(ctx, "Successfully generated mock flashcards",
		"card_count", len(cards),
		"user_id", userID.String())

	return cards, nil
}
