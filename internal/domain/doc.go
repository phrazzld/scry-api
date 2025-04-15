// Package domain contains the core business entities, value objects, and
// domain logic of the application. It represents the heart of the system,
// independent of any specific infrastructure or delivery mechanism.
//
// The domain package implements the core business model of the Scry application,
// following Domain-Driven Design principles. It contains the fundamental entities,
// value objects, and business rules that define the problem space, completely
// isolated from external concerns like databases, APIs, or UI.
//
// Key components:
//
// 1. Core Entities:
//   - User: Represents application users with authentication data
//   - Memo: Text content submitted by users for flashcard generation
//   - Card: Flashcards generated from memos, containing learning content
//   - UserCardStats: Tracks user performance and SRS scheduling for each card
//
// 2. Value Objects:
//   - ReviewOutcome: Represents the possible outcomes of a card review
//
// 3. Domain Validation:
//   - Each entity includes validation logic that enforces business rules
//   - Domain-specific error types for clear error handling
//
// 4. Domain Services:
//   - SRS (Spaced Repetition System): Algorithm for optimizing card review scheduling
//
// This package contains no dependencies on external infrastructure. Instead,
// other packages depend on domain definitions, following the Dependency Inversion
// Principle to maintain a clean architecture.
package domain
