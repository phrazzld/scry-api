# GeminiGenerator Tests

This directory contains tests for the `GeminiGenerator` implementation. The tests are designed to verify the functionality of the various components of the generator, including:

1. Constructor (`NewGeminiGenerator`)
2. Prompt template parsing and generation (`createPrompt` method)
3. API call with retry logic (`callGeminiWithRetry` method)
4. Response parsing logic (`parseResponse` method)
5. The main `GenerateCards` method implementing the Generator interface

## Important Note

Due to dependency issues with the Google Gemini API client libraries, the tests are currently skipped in the test file. The tests are fully implemented and ready to run once the dependency issues are resolved.

## Test Dependencies

The tests require the following dependencies:
- `github.com/google/generative-ai-go/genai`
- `google.golang.org/api/option`

## Test Structure

The tests are organized into the following files:

- `gemini_generator_test.go`: Contains the main test functions
- `gemini_generator_test_helpers.go`: Contains helper functions that expose unexported methods for testing

The test helper file uses a build tag `//go:build testing` to ensure it's only included in test builds.

## Testing Approach

1. **Constructor Tests**: Verify that the constructor properly validates configuration and initializes the generator.
2. **Prompt Creation Tests**: Verify that the prompt template is correctly rendered with the provided memo text.
3. **API Call Tests**: Verify that the retry logic works correctly with exponential backoff and properly handles errors.
4. **Response Parsing Tests**: Verify that API responses are correctly parsed into domain cards and that validation is properly enforced.
5. **Generate Cards Tests**: Verify that the main method correctly orchestrates the other components and properly handles errors.

## Running the Tests

Once the dependency issues are resolved, the tests can be run with:

```bash
go test -v -tags=testing ./internal/platform/gemini
```

The `-tags=testing` flag is required to include the test helper file.

## Future Improvements

For better testability, it would be beneficial to modify the `GeminiGenerator` to allow dependency injection of the API client. This would enable more comprehensive testing without relying on the actual API client.

A suggested approach is to:

1. Define an interface for the API client operations used in `GeminiGenerator`
2. Modify `NewGeminiGenerator` to accept an optional implementation of this interface
3. Create a mock implementation of this interface for testing

This would make the tests more reliable and eliminate the dependency on external services during testing.
