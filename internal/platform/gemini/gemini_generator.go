//go:build !test_without_external_deps
// +build !test_without_external_deps

package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/generation"
	generativelanguage "google.golang.org/api/ai/generativelanguage/v1beta"
	"google.golang.org/api/option"
)

// GeminiGenerator implements the generation.Generator interface using
// Google's Gemini API to generate flashcards from memo text.
type GeminiGenerator struct {
	// logger is used for structured logging
	logger *slog.Logger

	// config contains LLM-specific configuration
	config config.LLMConfig

	// promptTemplate is the parsed template for creating prompts
	promptTemplate *template.Template

	// client is the Gemini API client for making requests
	client *generativelanguage.Service

	// model is the name of the Gemini model to use
	model string
}

// We use ErrEmptyMemoText from errors.go

// NewGeminiGenerator creates a new instance of GeminiGenerator with the provided dependencies.
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

	// Validate configuration
	if config.GeminiAPIKey == "" {
		return nil, fmt.Errorf("%w: gemini API key cannot be empty", generation.ErrInvalidConfig)
	}

	if config.ModelName == "" {
		return nil, fmt.Errorf("%w: model name cannot be empty", generation.ErrInvalidConfig)
	}

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

	// Initialize the Gemini client
	client, err := generativelanguage.NewService(ctx, option.WithAPIKey(config.GeminiAPIKey))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create Gemini client: %v",
			generation.ErrInvalidConfig, err)
	}

	generator := &GeminiGenerator{
		logger:         logger,
		config:         config,
		promptTemplate: promptTemplate,
		client:         client,
		model:          config.ModelName,
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

// callGeminiWithRetry makes a call to the Gemini API with exponential backoff retry logic.
//
// It attempts to call the API up to config.MaxRetries times, using exponential backoff
// with jitter between retries for transient errors. Permanent errors (like content being
// blocked by safety filters) are returned immediately without retrying.
//
// Parameters:
//   - ctx: Context for the operation, which can be used for cancellation and logging
//   - prompt: The prompt string to send to the Gemini API
//
// Returns:
//   - The response from the Gemini API, mapped to the ResponseSchema structure
//   - An error if all retries fail or if a permanent error occurs
func (g *GeminiGenerator) callGeminiWithRetry(ctx context.Context, prompt string) (*ResponseSchema, error) {
	if prompt == "" {
		return nil, ErrEmptyMemoText
	}

	// Initialize retry variables
	maxRetries := g.config.MaxRetries
	baseDelaySeconds := g.config.RetryDelaySeconds
	attempt := 0
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Validate retry configuration
	if maxRetries < 0 {
		g.logger.WarnContext(ctx, "Invalid max retries value, using default", "max_retries", 3)
		maxRetries = 3
	}

	if baseDelaySeconds < 1 {
		g.logger.WarnContext(ctx, "Invalid retry delay value, using default", "base_delay_seconds", 2)
		baseDelaySeconds = 2
	}

	for attempt <= maxRetries {
		attemptNum := attempt + 1 // For logging (1-based)
		g.logger.InfoContext(ctx, "Making Gemini API call",
			"attempt", attemptNum,
			"max_attempts", maxRetries+1)

		// Prepare the API request
		req := &generativelanguage.GenerateContentRequest{
			Contents: []*generativelanguage.Content{
				{
					Parts: []*generativelanguage.Part{
						{
							Text: prompt,
						},
					},
				},
			},
		}

		// Call the Gemini API
		var response *ResponseSchema
		var err error
		var isTransientError bool

		// This is a production implementation
		resp, err := g.client.Models.GenerateContent(g.model, req).Context(ctx).Do()
		if err != nil {
			// Handle API errors
			isTransientError = true // Assume transient error by default
			g.logger.ErrorContext(ctx, "Gemini API call error",
				"error", err,
				"attempt", attemptNum)
		} else if resp.Candidates == nil || len(resp.Candidates) == 0 {
			// No candidates in response
			err = fmt.Errorf("%w: no content generated", generation.ErrInvalidResponse)
			isTransientError = false
		} else if resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
			// No content in candidate
			err = fmt.Errorf("%w: empty content in response", generation.ErrInvalidResponse)
			isTransientError = false
		} else if resp.Candidates[0].FinishReason == "SAFETY" {
			// Content blocked by safety filters
			err = fmt.Errorf("%w: content blocked by safety filters", generation.ErrContentBlocked)
			isTransientError = false
		} else {
			// Extract the response text
			text := ""
			for _, part := range resp.Candidates[0].Content.Parts {
				if part.Text != "" {
					text += part.Text
				}
			}

			// Parse the JSON response
			var parsedResponse ResponseSchema
			if err = json.Unmarshal([]byte(text), &parsedResponse); err != nil {
				err = fmt.Errorf("%w: failed to parse JSON response: %v", generation.ErrInvalidResponse, err)
				isTransientError = false
			} else {
				response = &parsedResponse
			}
		}

		// If successful, return the response
		if err == nil {
			g.logger.InfoContext(ctx, "Gemini API call successful",
				"attempt", attemptNum)
			return response, nil
		}

		// Log the error
		g.logger.ErrorContext(ctx, "Gemini API call failed",
			"attempt", attemptNum,
			"error", err)

		// Determine if the error is transient or permanent
		if errors.Is(err, generation.ErrContentBlocked) || errors.Is(err, generation.ErrInvalidResponse) {
			// Permanent error, return immediately
			g.logger.WarnContext(ctx, "Permanent error occurred, not retrying",
				"error_type", err)
			return nil, err
		}

		// Check if we've reached the max retries
		if attempt >= maxRetries {
			g.logger.WarnContext(ctx, "Maximum retry attempts reached",
				"max_retries", maxRetries)
			return nil, fmt.Errorf("%w: exceeded maximum retry attempts (%d)",
				generation.ErrTransientFailure, maxRetries)
		}

		// Only retry for transient errors
		if !isTransientError {
			g.logger.WarnContext(ctx, "Non-transient error occurred, not retrying")
			return nil, err
		}

		// Calculate exponential backoff with jitter
		// delay = baseDelay * (2^attempt) * (0.5 + rand(0, 0.5))
		backoffSeconds := float64(baseDelaySeconds) * math.Pow(2, float64(attempt))
		jitterFactor := 0.5 + rng.Float64()*0.5 // Between 0.5 and 1.0
		delaySeconds := backoffSeconds * jitterFactor
		delay := time.Duration(delaySeconds * float64(time.Second))

		g.logger.InfoContext(ctx, "Retrying after delay",
			"attempt", attemptNum,
			"delay_seconds", delaySeconds)

		// Wait for the delay or context cancellation
		select {
		case <-time.After(delay):
			// Continue to next retry
		case <-ctx.Done():
			// Context was cancelled
			g.logger.WarnContext(ctx, "API call cancelled during retry delay",
				"attempt", attemptNum,
				"ctx_err", ctx.Err())
			return nil, fmt.Errorf("%w: %v", generation.ErrTransientFailure, ctx.Err())
		}

		attempt++
	}

	// This should not be reached due to the check inside the loop,
	// but return an error just in case
	return nil, fmt.Errorf("%w: failed after %d attempts",
		generation.ErrTransientFailure, attempt)
}

// parseResponse converts a ResponseSchema from the Gemini API into domain.Card objects.
//
// It validates each card in the response and creates domain.Card objects with
// properly formatted content. If any card in the response fails validation, the
// method returns an error and no cards are returned.
//
// Parameters:
//   - ctx: Context for the operation, which can be used for logging
//   - response: The structured response from the Gemini API
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

	g.logger.InfoContext(ctx, "Parsing Gemini API response",
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
		g.logger.DebugContext(ctx, "Created card from API response",
			"card_id", card.ID.String(),
			"front_length", len(cardSchema.Front),
			"back_length", len(cardSchema.Back))
	}

	g.logger.InfoContext(ctx, "Successfully parsed API response",
		"created_cards", len(cards))

	return cards, nil
}

// GenerateCards creates flashcards based on the provided memo text and user ID.
// It fulfills the generation.Generator interface by:
// 1. Creating a prompt from the memo text
// 2. Calling the Gemini API with retry logic
// 3. Parsing the API response into domain.Card objects
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

	g.logger.InfoContext(ctx, "Starting flashcard generation",
		"memo_length", len(memoText),
		"user_id", userID.String())

	// Step 1: Create prompt from memo text
	prompt, err := g.createPrompt(ctx, memoText)
	if err != nil {
		g.logger.ErrorContext(ctx, "Failed to create prompt",
			"error", err)
		return nil, fmt.Errorf("%w: %v", generation.ErrGenerationFailed, err)
	}

	// Step 2: Call Gemini API with retry logic
	response, err := g.callGeminiWithRetry(ctx, prompt)
	if err != nil {
		// The underlying error is already appropriately typed in callGeminiWithRetry
		g.logger.ErrorContext(ctx, "Gemini API call failed",
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
		g.logger.ErrorContext(ctx, "Failed to parse API response",
			"error", err)
		return nil, fmt.Errorf("%w: %v", generation.ErrGenerationFailed, err)
	}

	g.logger.InfoContext(ctx, "Successfully generated flashcards",
		"card_count", len(cards),
		"user_id", userID.String())

	return cards, nil
}
