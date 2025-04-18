# GeminiGenerator Tests

This directory contains tests for the `GeminiGenerator` implementation. The tests are designed to verify the functionality of the various components of the generator, including:

1. Constructor (`NewGeminiGenerator`)
2. Prompt template parsing and generation (`createPrompt` method)
3. API call with retry logic (`callGeminiWithRetry` method)
4. Response parsing logic (`parseResponse` method)
5. The main `GenerateCards` method implementing the Generator interface

## Dual Implementation Approach

This package contains two implementations of the `GeminiGenerator`:

1. **Real Implementation** (default): Uses the actual Gemini API
2. **Mock Implementation** (with build tag): Uses a mock implementation for testing without external dependencies

The package is structured to use build tags to select the appropriate implementation:

- `gemini_generator.go`: Contains the real implementation (active when `test_without_external_deps` tag is NOT set)
- `gemini_generator_mock.go`: Contains the mock implementation (active when `test_without_external_deps` tag IS set)
- `types.go`: Contains shared type definitions used by both implementations

### Shared Types

To prevent type declaration conflicts between implementations, common types are defined in `types.go`:

- `promptData`: The data structure passed to the prompt template
- `ResponseSchema`: The JSON response structure from the Gemini API
- `CardSchema`: Represents a single flashcard in the API response

## Test Dependencies

The real implementation tests require the following dependencies:
- `github.com/google/generative-ai-go/genai`
- `google.golang.org/api/option`

The mock implementation has minimal dependencies and can be tested without these external libraries.

## Test Structure

The tests are organized into the following files:

- `gemini_generator_test.go`: Contains the main test functions
- `gemini_generator_test_helpers.go`: Contains helper functions that expose unexported methods for testing

Some test helper files use build tags to ensure they're only included in appropriate test builds.

## Running the Tests

### With Mock Implementation (no external dependencies)

```bash
go test -v -tags=test_without_external_deps ./internal/platform/gemini
```

### With Real Implementation (requires API access)

```bash
go test -v ./internal/platform/gemini
```

## Usage in Project

By default, the real implementation is used. For testing environments or CI/CD pipelines where external dependencies should be avoided, you can build with the `test_without_external_deps` tag:

```bash
go build -tags=test_without_external_deps ./...
```

## Future Improvements

For better testability, it would be beneficial to modify the `GeminiGenerator` to allow dependency injection of the API client. This would enable more comprehensive testing without relying on the actual API client.

A suggested approach is to:

1. Define an interface for the API client operations used in `GeminiGenerator`
2. Modify `NewGeminiGenerator` to accept an optional implementation of this interface
3. Create a mock implementation of this interface for testing

This would make the tests more reliable and eliminate the dependency on external services during testing.
