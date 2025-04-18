// Package gemini provides an implementation of the generation.Generator interface
// that uses Google's Gemini API for generating flashcards from memo text.
//
// This package is an infrastructure adapter in the hexagonal architecture,
// connecting the application's domain logic to Google's external Gemini AI service.
// It translates between the application's domain models and the Gemini API
// without exposing the details of the external service to the core application.
//
// Key components:
//
// 1. GeminiGenerator:
//   - Implements the generation.Generator interface
//   - Handles communication with the Gemini API
//   - Processes structured responses into domain models
//
// 2. Prompt Management:
//   - Loads prompt templates from files
//   - Substitutes dynamic content into templates
//   - Formats prompts according to Gemini's requirements
//
// 3. Response Processing:
//   - Parses structured JSON responses from the API
//   - Validates responses against expected schema
//   - Converts API responses to domain Card objects
//
// 4. Error Handling:
//   - Implements retry logic with exponential backoff for transient errors
//   - Categorizes and translates API errors to application-specific errors
//   - Handles content filtering and safety measures
//
// The package depends on Google's generative-ai-go/genai client library
// for communicating with the Gemini API, and handles authentication,
// request formatting, and response processing according to Google's
// API specifications.
package gemini
