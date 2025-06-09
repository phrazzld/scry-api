# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Card Management API endpoints:
  - `PUT /cards/{id}` for editing card content with ownership validation
  - `DELETE /cards/{id}` for deleting cards with ownership validation
  - `POST /cards/{id}/postpone` for postponing card reviews by a specified number of days
- Core interfaces for card management:
  - `CardStore` interface with `GetByID`, `UpdateContent`, and `Delete` methods
  - `UserCardStatsStore` interface with `GetForUpdate` and `Update` methods
  - `SRSService.PostponeReview` method to handle spaced repetition algorithm logic for postponed cards
  - `CardService` interface with `UpdateCardContent`, `DeleteCard`, and `PostponeCard` methods
- Implementations of store interfaces in PostgreSQL with transaction support
- CASCADE DELETE behavior on foreign key constraints ensuring automatic cleanup of user_card_stats when cards are deleted
- Robust error handling across all layers with specific error types
- Comprehensive test coverage:
  - Unit tests for core business logic
  - Integration tests for database operations
  - HTTP API integration tests for all endpoints
- OpenAPI (Swagger) specification with detailed documentation of all endpoints
