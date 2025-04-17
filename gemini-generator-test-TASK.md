# Task T015: Create tests for GeminiGenerator implementation

## Background
As part of the Generation Service implementation, the GeminiGenerator has been created to generate flashcards from memo text using Google's Gemini API. The implementation includes prompt creation, API call with retry logic, response parsing, and the main GenerateCards method. These components need to be thoroughly tested to ensure the implementation works correctly.

## Task Description
Create comprehensive tests for the GeminiGenerator implementation in `internal/platform/gemini/gemini_generator_test.go`. The tests should cover:

1. Constructor (`NewGeminiGenerator`) including validation of configuration parameters
2. Prompt template parsing and generation (`createPrompt` method)
3. API call with retry logic (`callGeminiWithRetry` method)
4. Response parsing logic (`parseResponse` method)
5. The main `GenerateCards` method implementing the Generator interface

## Key Considerations
- Follow the project's testing strategy and patterns
- Test both successful cases and error handling
- Mock only external dependencies like the Gemini API client, not internal collaborators
- Ensure tests are clear, maintainable, and focus on behavior
- Test that retry logic works correctly with exponential backoff
- Verify proper error propagation and type wrapping
- Use test helpers and fixtures as appropriate but maintain clarity

## Expected Deliverables
1. A comprehensive test file `internal/platform/gemini/gemini_generator_test.go`
2. Test coverage for all major methods of the GeminiGenerator struct
3. Properly mocked external dependencies
4. Tests that verify both happy path and error scenarios

## Related Files
- `/internal/platform/gemini/gemini_generator.go` - The implementation to test
- `/internal/generation/generator.go` - The Generator interface
- `/internal/generation/errors.go` - Error types for generation
- `/internal/domain/card.go` - Domain model for cards
- `/internal/config/config.go` - Configuration structure

## DEVELOPMENT_PHILOSOPHY.md Highlights
- Embrace the Unix Philosophy: Components should "do one thing and do it well"
- Strict Separation of Concerns: Core business logic separated from infrastructure
- Dependency Inversion: High-level policy must not depend on low-level details
- Consistent Error Handling: Fail predictably and informatively
- Mocking Policy: Mock only true external system boundaries defined by local interfaces, never internal collaborators
- Favor Pure Functions: Isolate side effects, implement core logic and transformations as pure functions
- Test Characteristics (FIRST): Fast, Independent/Isolated, Repeatable/Reliable, Self-Validating, Timely/Thorough
