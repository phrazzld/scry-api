// Package gemini provides implementations for the generation interface using Google's Gemini API.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/generation"
)

// NewGenerator creates the appropriate GeminiGenerator implementation based on build tags.
// This factory function allows the application to use the real implementation in production
// and the mock implementation in test environments with the test_without_external_deps build tag.
//
// Parameters:
//   - ctx: Context for initialization, which may include timeouts or cancellation
//   - logger: A logger for recording operations
//   - config: Configuration information including API keys and settings
//
// Returns:
//   - A generation.Generator implementation
//   - An error if initialization fails
func NewGenerator(
	ctx context.Context,
	logger *slog.Logger,
	config config.LLMConfig,
) (generation.Generator, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	// Log the initialization attempt
	logger.InfoContext(ctx, "Initializing Gemini generator")

	// Validate configuration (performed differently based on build tags)
	// General configuration validation that applies to all environments
	if config.PromptTemplatePath == "" {
		return nil, fmt.Errorf(
			"%w: prompt template path cannot be empty",
			generation.ErrInvalidConfig,
		)
	}

	// Additional validation for production environments
	// In test environments with test_without_external_deps tag, these validations are less strict
	// Since we're running with the test_without_external_deps tag, we skip detailed validation
	// Just log the configuration source
	logger.InfoContext(ctx, "Using test configuration for Gemini generator")

	// Call the version-specific implementation
	generator, err := NewGeminiGenerator(ctx, logger, config)
	if err != nil {
		return nil, err
	}

	return generator, nil
}

// createPromptFromTemplate generates a prompt string from the template with the provided memo text.
//
// It executes the template with the memo text and returns the resulting string.
// If the memo text is empty or the template execution fails, it returns an error.
//
// Parameters:
//   - ctx: Context for the operation, which can be used for logging
//   - logger: Structured logger for logging operations
//   - tmpl: The parsed template to execute
//   - memoText: The text of the memo to include in the prompt
//
// Returns:
//   - The generated prompt string
//   - An error if the memo text is empty or the template execution fails
func createPromptFromTemplate(
	ctx context.Context,
	logger *slog.Logger,
	tmpl *template.Template,
	memoText string,
) (string, error) {
	// Validate input
	if memoText == "" {
		return "", ErrEmptyMemoText
	}

	// Create data for template
	data := promptData{
		MemoText: memoText,
	}

	logger.DebugContext(ctx, "Generating prompt from template",
		"memo_length", len(memoText),
		"template_name", tmpl.Name())

	// Execute template
	var promptBuffer bytes.Buffer
	if err := tmpl.Execute(&promptBuffer, data); err != nil {
		return "", fmt.Errorf("failed to execute prompt template: %w", err)
	}

	prompt := promptBuffer.String()
	logger.DebugContext(ctx, "Prompt generated successfully",
		"prompt_length", len(prompt))

	return prompt, nil
}

// parseResponseToCards converts a ResponseSchema from the API into domain.Card objects.
//
// It validates each card in the response and creates domain.Card objects with
// properly formatted content. If any card in the response fails validation, the
// method returns an error and no cards are returned.
//
// Parameters:
//   - ctx: Context for the operation, which can be used for logging
//   - logger: Structured logger for logging operations
//   - response: The structured response from the API
//   - userID: The UUID of the user who owns the memo
//   - memoID: The UUID of the memo from which the cards are generated
//   - isReal: Whether this is a real API response (for logging purposes)
//
// Returns:
//   - A slice of domain.Card pointers
//   - An error if the response is invalid or card creation fails
func parseResponseToCards(
	ctx context.Context,
	logger *slog.Logger,
	response *ResponseSchema,
	userID uuid.UUID,
	memoID uuid.UUID,
	isReal bool,
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

	sourceType := "mock API"
	if isReal {
		sourceType = "Gemini API"
	}

	logger.InfoContext(ctx, "Parsing "+sourceType+" response",
		"card_count", len(response.Cards),
		"user_id", userID.String(),
		"memo_id", memoID.String())

	// Create domain cards from response
	cards := make([]*domain.Card, 0, len(response.Cards))
	for i, cardSchema := range response.Cards {
		// Validate required fields
		if cardSchema.Front == "" {
			return nil, fmt.Errorf(
				"%w: card %d missing front side",
				generation.ErrInvalidResponse,
				i,
			)
		}

		if cardSchema.Back == "" {
			return nil, fmt.Errorf(
				"%w: card %d missing back side",
				generation.ErrInvalidResponse,
				i,
			)
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
		logger.DebugContext(ctx, "Created card from "+sourceType+" response",
			"card_id", card.ID.String(),
			"front_length", len(cardSchema.Front),
			"back_length", len(cardSchema.Back))
	}

	logger.InfoContext(ctx, "Successfully parsed "+sourceType+" response",
		"created_cards", len(cards))

	return cards, nil
}
