# PLAN.MD - Generation Service Implementation

## Task Description

Implement the Generation Service that defines the `Generator` interface and implements a Gemini-based generator to create flashcards from user memos. The implementation should adhere to the architecture guidelines with a focus on separation of concerns, testability, and proper error handling.

This task involves:
1. Defining the `generation.Generator` interface in the core application layer
2. Implementing a `geminiGenerator` struct in the platform layer
3. Loading prompt templates from external configuration
4. Implementing logic to call the Gemini API
5. Implementing error handling and retry logic for transient errors
6. Securely managing API keys via configuration
7. Ensuring the service is swappable via dependency inversion

## Recommended Approach: Interface-First with Platform Abstraction

This approach prioritizes **Simplicity** while strictly adhering to the project's **Separation of Concerns** and **Testability** principles. We'll define the `Generator` interface in the core application layer (`internal/generation`) and implement a concrete `geminiGenerator` in the platform layer (`internal/platform/gemini`). This ensures the core application logic depends only on the abstraction, making it easily testable and allowing for future swapping of LLM providers.

### Alignment with Project Standards

1. **Simplicity/Clarity:** This approach introduces the minimum necessary components (an interface and an implementation), avoiding unnecessary abstraction layers.
2. **Separation of Concerns:** The interface (`generation.Generator`) is correctly placed in the core layer and the concrete implementation (`geminiGenerator`) in the platform layer, ensuring the core depends only on the abstraction.
3. **Testability (Minimal Mocking):** The `generation.Generator` interface serves as the mockable boundary for tests, aligning with the "Mock ONLY True External System Boundaries" principle.
4. **Coding Conventions:** The plan follows Go conventions and the established project structure.
5. **Documentability:** The clear separation makes documenting the purpose of each package straightforward.

## Detailed Implementation Steps

### 1. Define `generation.Generator` Interface (`internal/generation/generator.go`)

```go
package generation

import (
    "context"
    "github.com/google/uuid"
    "github.com/phrazzld/scry-api/internal/domain"
)

// Generator defines the interface for generating flashcards from text.
type Generator interface {
    // GenerateCards creates flashcards based on the provided memo text and user ID.
    // It returns a slice of Card domain objects or an error if generation fails.
    GenerateCards(ctx context.Context, memoText string, userID uuid.UUID) ([]*domain.Card, error)
}
```

Also create `errors.go` to define specific error types:

```go
package generation

import "errors"

// Common errors returned by the generation package
var (
    ErrGenerationFailed   = errors.New("failed to generate cards from text")
    ErrInvalidResponse    = errors.New("invalid response from language model")
    ErrContentBlocked     = errors.New("content blocked by language model safety filters")
    ErrTransientFailure   = errors.New("transient error during card generation")
    ErrInvalidConfig      = errors.New("invalid generator configuration")
)
```

### 2. Update Configuration (`internal/config/config.go`)

Add or update the LLM configuration section:

```go
// LLMConfig defines settings for Language Model integration.
type LLMConfig struct {
    // GeminiAPIKey is the API key for accessing Google's Gemini AI service.
    GeminiAPIKey string `mapstructure:"gemini_api_key" validate:"required"`
    // ModelName specifies the Gemini model to use (e.g., "gemini-1.5-flash").
    ModelName string `mapstructure:"model_name" validate:"required"`
    // PromptTemplatePath is the path to the prompt template file.
    PromptTemplatePath string `mapstructure:"prompt_template_path" validate:"required"`
    // MaxRetries specifies the maximum number of retries for transient API errors.
    MaxRetries int `mapstructure:"max_retries" validate:"omitempty,gte=0,lte=5"`
    // RetryDelaySeconds specifies the base delay between retries in seconds.
    RetryDelaySeconds int `mapstructure:"retry_delay_seconds" validate:"omitempty,gte=1,lte=60"`
}
```

Update `internal/config/load.go` to set defaults (e.g., `MaxRetries: 3`, `RetryDelaySeconds: 2`).

Update `config.yaml.example` with the new configuration entries.

### 3. Create Prompt Template

Create a sample flashcard prompt template file at a location like `prompts/flashcard_template.txt`:

```
You are an expert flashcard creator. Your task is to create effective, concise flashcards from the provided text.

Text: {{.MemoText}}

Create flashcards in JSON format. Each card should have a "front" (question) and "back" (answer) field.
Example:
[
  {"front": "Question 1?", "back": "Answer 1"},
  {"front": "Question 2?", "back": "Answer 2"}
]

Guidelines:
- Create 3-5 flashcards covering key concepts
- Make questions specific and clear
- Ensure answers are concise and complete
- Include only information present in the text
- Output in valid JSON format only
```

### 4. Implement `geminiGenerator` (`internal/platform/gemini/gemini_generator.go`)

```go
package gemini

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "os"
    "text/template"
    "time"

    "github.com/google/generative-ai-go/genai"
    "github.com/google/uuid"
    "github.com/phrazzld/scry-api/internal/config"
    "github.com/phrazzld/scry-api/internal/domain"
    "github.com/phrazzld/scry-api/internal/generation"
    "google.golang.org/api/option"
)

type promptData struct {
    MemoText string
}

type cardData struct {
    Front string `json:"front"`
    Back  string `json:"back"`
}

type geminiGenerator struct {
    apiKey       string
    modelName    string
    promptTmpl   *template.Template
    maxRetries   int
    initialDelay time.Duration
    logger       *slog.Logger
}

// NewGeminiGenerator creates a new instance of a Generator using the Gemini API.
func NewGeminiGenerator(cfg config.LLMConfig, logger *slog.Logger) (generation.Generator, error) {
    // Load and parse prompt template
    tmplContent, err := os.ReadFile(cfg.PromptTemplatePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read prompt template: %w", err)
    }

    tmpl, err := template.New("flashcard").Parse(string(tmplContent))
    if err != nil {
        return nil, fmt.Errorf("failed to parse prompt template: %w", err)
    }

    return &geminiGenerator{
        apiKey:       cfg.GeminiAPIKey,
        modelName:    cfg.ModelName,
        promptTmpl:   tmpl,
        maxRetries:   cfg.MaxRetries,
        initialDelay: time.Duration(cfg.RetryDelaySeconds) * time.Second,
        logger:       logger.With("component", "gemini_generator"),
    }, nil
}

// GenerateCards implements the generation.Generator interface.
func (g *geminiGenerator) GenerateCards(ctx context.Context, memoText string, userID uuid.UUID) ([]*domain.Card, error) {
    // Create prompt from template
    prompt, err := g.createPrompt(memoText)
    if err != nil {
        g.logger.Error("Failed to create prompt", "error", err)
        return nil, fmt.Errorf("failed to create prompt: %w", err)
    }

    // Call Gemini API with retry logic
    result, err := g.callGeminiWithRetry(ctx, prompt)
    if err != nil {
        g.logger.Error("Failed to generate cards", "error", err)
        return nil, err
    }

    // Parse response
    cards, err := g.parseResponse(result, userID)
    if err != nil {
        g.logger.Error("Failed to parse response", "error", err)
        return nil, err
    }

    g.logger.Info("Successfully generated cards", "count", len(cards))
    return cards, nil
}

// createPrompt generates the prompt string from the template
func (g *geminiGenerator) createPrompt(memoText string) (string, error) {
    var buf bytes.Buffer
    data := promptData{MemoText: memoText}

    if err := g.promptTmpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to execute prompt template: %w", err)
    }

    return buf.String(), nil
}

// callGeminiWithRetry calls the Gemini API with exponential backoff retry
func (g *geminiGenerator) callGeminiWithRetry(ctx context.Context, prompt string) (string, error) {
    var lastErr error
    var result string

    for attempt := 0; attempt <= g.maxRetries; attempt++ {
        // Check if context is cancelled
        if ctx.Err() != nil {
            return "", ctx.Err()
        }

        // If not the first attempt, wait with exponential backoff
        if attempt > 0 {
            delay := g.initialDelay * (1 << (attempt - 1))
            g.logger.Warn("Retrying API call after delay",
                "attempt", attempt,
                "delay", delay.String(),
                "error", lastErr)

            select {
            case <-time.After(delay):
                // Wait completed
            case <-ctx.Done():
                return "", ctx.Err()
            }
        }

        // Create new client for each attempt
        client, err := genai.NewClient(ctx, option.WithAPIKey(g.apiKey))
        if err != nil {
            lastErr = fmt.Errorf("failed to create Gemini client: %w", err)
            continue
        }
        defer client.Close()

        // Get model and configure it for JSON output
        model := client.GenerativeModel(g.modelName)
        model.ResponseMIMEType = "application/json"

        // Define expected JSON schema for structured output
        schema := &genai.Schema{
            Type: genai.TypeArray,
            Items: &genai.Schema{
                Type: genai.TypeObject,
                Properties: map[string]*genai.Schema{
                    "front": {Type: genai.TypeString},
                    "back":  {Type: genai.TypeString},
                },
                Required: []string{"front", "back"},
            },
        }
        model.ResponseSchema = schema

        // Call API
        resp, err := model.GenerateContent(ctx, genai.Text(prompt))

        // Handle different error types
        if err != nil {
            statusErr, ok := err.(*genai.StatusError)
            if ok {
                // Handle specific error types based on status code
                switch statusErr.Status.Code {
                case 400: // Invalid request
                    return "", generation.ErrInvalidConfig
                case 403: // Content blocked
                    return "", generation.ErrContentBlocked
                }
            }

            // Store error for potential retry
            lastErr = err
            continue
        }

        // Check for empty response
        if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
            lastErr = generation.ErrInvalidResponse
            continue
        }

        // Extract text from response
        text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
        if !ok {
            lastErr = generation.ErrInvalidResponse
            continue
        }

        // Success - return the result
        result = string(text)
        return result, nil
    }

    // If we get here, all attempts failed
    return "", fmt.Errorf("%w: %v", generation.ErrTransientFailure, lastErr)
}

// parseResponse converts the API response into domain.Card objects
func (g *geminiGenerator) parseResponse(jsonStr string, userID uuid.UUID) ([]*domain.Card, error) {
    var cardDataList []cardData
    if err := json.Unmarshal([]byte(jsonStr), &cardDataList); err != nil {
        return nil, fmt.Errorf("%w: failed to parse JSON: %v",
            generation.ErrInvalidResponse, err)
    }

    // Validate response
    if len(cardDataList) == 0 {
        return nil, generation.ErrInvalidResponse
    }

    // Convert to domain.Card objects
    cards := make([]*domain.Card, 0, len(cardDataList))
    for _, data := range cardDataList {
        // Validate card data
        if data.Front == "" || data.Back == "" {
            g.logger.Warn("Skipping invalid card with empty fields",
                "front_empty", data.Front == "",
                "back_empty", data.Back == "")
            continue
        }

        // Create card content
        content := domain.CardContent{
            Front: data.Front,
            Back:  data.Back,
        }

        // Marshal content to JSON
        contentBytes, err := json.Marshal(content)
        if err != nil {
            g.logger.Error("Failed to marshal card content", "error", err)
            continue
        }

        // Create new card (memo ID will be set by the caller)
        card, err := domain.NewCard(userID, uuid.Nil, contentBytes)
        if err != nil {
            g.logger.Error("Failed to create card", "error", err)
            continue
        }

        cards = append(cards, card)
    }

    if len(cards) == 0 {
        return nil, generation.ErrGenerationFailed
    }

    return cards, nil
}
```

Add package documentation in `internal/platform/gemini/doc.go`:

```go
// Package gemini provides an implementation of the generation.Generator interface
// using Google's Gemini API. It handles API key management, prompt templating,
// retries for transient errors, and response parsing to convert API results into
// domain objects.
package gemini
```

### 5. Update Dependencies and Integration

1. Add required dependencies to `go.mod`:
   - `github.com/google/generative-ai-go/genai`
   - `google.golang.org/api`

2. Update `cmd/server/main.go` to initialize the generator and inject it:

```go
// In the setupDependencies function or similar
geminiGenerator, err := gemini.NewGeminiGenerator(config.LLM, logger)
if err != nil {
    return nil, fmt.Errorf("failed to create Gemini generator: %w", err)
}

// Store in dependencies struct
deps.Generator = geminiGenerator

// Pass to the memo generation task factory
memoTaskFactory := task.NewMemoGenerationTaskFactory(
    memoStore,
    deps.Generator, // Instead of mockGenerator
    cardStore,
    logger,
)
```

### 6. Implement Tests

1. **Unit Tests for Generator Interface** (`internal/generation/generation_test.go`):
   - Test that documentation contains key terms
   - Add tests for custom error types

2. **Unit Tests for Gemini Implementation** (`internal/platform/gemini/gemini_generator_test.go`):
   - Test prompt template parsing
   - Test error handling and retry logic
   - Test response parsing
   - Use mocks to simulate the Gemini API responses

3. **Integration Tests**:
   - Modify existing integration tests to use a mock implementation of `generation.Generator`
   - Create a mock generator in `internal/mocks/generator.go`

### 7. Testing Strategy

- **Unit Tests**: Test the `geminiGenerator` implementation thoroughly, mocking only the external Gemini API
- **Integration Tests**: Test the components using the `generation.Generator` interface via mocks
- **Manual Testing**: Test the full flow with a valid Gemini API key (optional)

## Validation Criteria

Before marking this task as complete:

1. All unit tests for `generation.Generator` interface and `geminiGenerator` implementation pass
2. All integration tests for components using the generator pass
3. The code adheres to the project's development philosophy
4. The API key is securely loaded via configuration
5. Error handling is robust, including retries for transient errors
6. Implementation is properly documented
