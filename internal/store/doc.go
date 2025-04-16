// Package store defines interfaces for data persistence operations.
// These interfaces abstract the underlying data storage mechanism from
// the application's core logic, allowing business rules to remain
// independent of specific database technologies or persistence details.
//
// The store package implements the repository interfaces in the clean architecture,
// serving as a boundary between the application's domain/business logic and the
// data persistence infrastructure. This package contains only interface definitions
// and related types, with the actual implementations provided in infrastructure
// packages like internal/platform/postgres.
//
// Key components:
//
// 1. Repository Interfaces:
//   - UserStore: Interface for user persistence operations
//   - MemoStore: Interface for memo persistence operations
//   - CardStore: Interface for flashcard persistence operations
//   - UserCardStatsStore: Interface for SRS statistics persistence
//
// 2. Common Types:
//   - DBTX: Interface that abstracts *sql.DB and *sql.Tx for flexible transaction handling
//   - QueryOptions: Types for pagination, sorting, and filtering
//
// 3. Error Definitions:
//   - Standard repository errors (ErrNotFound, ErrDuplicate, etc.)
//   - Clear error semantics for consistent error handling across the application
//
// By depending only on these interfaces (rather than concrete implementations),
// application services maintain isolation from specific storage technologies,
// enabling:
//
// - Easy substitution of different storage technologies
// - Simplified testing with mocks or in-memory implementations
// - Clear separation of concerns between business logic and data access
//
// All store interfaces are designed around domain entities defined in the
// internal/domain package, ensuring that persistence concerns don't leak
// into the application's core business logic.
package store
