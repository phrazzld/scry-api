# Gemini API Generator

This package provides an implementation of the generation.Generator interface using Google's Gemini API for generating flashcards from memo text.

## API Modernization

In Spring 2025, we modernized the Gemini API client implementation to use the latest Google AI client library `google.golang.org/genai` instead of the deprecated `google.golang.org/api/ai/generativelanguage/v1beta` package.

### Key Improvements

1. **Simplified API**: The new implementation uses Google's more developer-friendly client library with better abstractions and reduced boilerplate code.

2. **Enhanced Error Handling**: Comprehensive error classification and handling with proper error wrapping for easier diagnosis of issues.

3. **Better Testing Support**: Improved testing architecture with proper mocking that can run without external dependencies using build tags.

4. **Exponential Backoff Retry**: More sophisticated retry mechanism with jitter to handle transient API errors.

5. **Structured Logging**: Enhanced logging with contextual information throughout the request lifecycle.

## Usage

The primary entry point is the `GenerateCards` method on the `GeminiGenerator` struct:

```go
func (g *GeminiGenerator) GenerateCards(
    ctx context.Context,
    memoText string,
    userID uuid.UUID,
) ([]*domain.Card, error)
```

### Initialization

```go
// Create a new GeminiGenerator with the given dependencies
generator, err := gemini.NewGeminiGenerator(ctx, logger, config)
if err != nil {
    // Handle initialization error
}

// Generate cards for a memo
cards, err := generator.GenerateCards(ctx, memoContent, userID)
if err != nil {
    // Handle generation error
}
```

## Testing

The package supports testing with and without external dependencies using build tags:

### With External Dependencies (Real API)

```bash
go test ./internal/platform/gemini/...
```

### Without External Dependencies (Mock API)

```bash
go test -tags=test_without_external_deps ./internal/platform/gemini/...
```

## Error Types

The package uses error types from the `internal/generation` package, including:

- `ErrGenerationFailed`: General card generation failure
- `ErrInvalidResponse`: Invalid or malformed API response
- `ErrContentBlocked`: Content blocked by safety filters
- `ErrTransientFailure`: Temporary errors that might resolve on retry
- `ErrInvalidConfig`: Invalid generator configuration

## Architecture

The implementation follows clean architecture principles:

1. `GenerateCards`: Main interface method that orchestrates the generation process
2. `createPrompt`: Creates structured prompts from memo text using templates
3. `callGeminiWithRetry`: Makes API calls with retry logic for transient errors
4. `parseResponse`: Converts API responses to domain models