// Package generation provides interfaces and implementations for interacting
// with external AI/LLM services for content generation. It abstracts the
// details of LLM API integration (Gemini), allowing the application to generate
// flashcards from user memos without coupling to specific external services.
//
// The generation package serves as an infrastructure adapter for AI/LLM services,
// following the hexagonal architecture pattern. It provides a clean separation
// between the application's core domain logic and the specific external AI services
// used for generating content.
//
// Key components:
//
// 1. Generator Interface:
//   - Defines the contract for generating flashcards from memo content
//   - Allows swapping different AI providers without changing application logic
//
// 2. Provider Implementations:
//   - GeminiGenerator: Interacts with Google's Gemini API
//   - (Future) Support for other LLM providers like OpenAI, Anthropic, etc.
//
// 3. Prompt Management:
//   - Loads and manages prompt templates from configuration
//   - Handles dynamic prompt construction based on user content
//
// 4. Error Handling:
//   - Provides standardized error types for LLM-specific failure scenarios
//   - Implements retry logic for transient API failures
//
// Usage:
//
// The generation package is typically used by application services that need to
// generate flashcards from user-provided content. The implementation details of
// which LLM is used are hidden behind the Generator interface, allowing the
// application to remain agnostic about the specific AI service provider.
package generation
