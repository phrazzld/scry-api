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
//   - TaskStore: Interface for background task persistence operations
//   - All interfaces MUST include a WithTx method for transaction support,
//     regardless of whether they currently participate in transactions
//
// 2. Transaction Support:
//   - DBTX: Interface that abstracts *sql.DB and *sql.Tx for flexible transaction handling
//   - RunInTransaction: Helper function to manage transaction boundaries
//   - WithTx pattern for transaction-aware store instances
//   - Transaction management is the responsibility of the service layer, not stores
//   - All stores can participate in transactions via their WithTx method
//
// 3. Error Definitions:
//   - Base error types: ErrNotFound, ErrDuplicate, ErrInvalidEntity, etc.
//   - Entity-specific errors: ErrUserNotFound, ErrMemoNotFound, etc.
//   - Error wrapping using fmt.Errorf("%w", err) and the errors package
//   - Structured error handling using errors.Is() and errors.As()
//
// 4. Error Mapping:
//   - Utilities to map database-specific errors to domain-specific errors
//   - Hiding of internal error details while preserving error semantics
//   - Consistent error handling patterns across all implementations
//
// By depending only on these interfaces (rather than concrete implementations),
// application services maintain isolation from specific storage technologies,
// enabling:
//
// - Easy substitution of different storage technologies
// - Simplified testing with mocks or in-memory implementations
// - Clear separation of concerns between business logic and data access
// - Transaction management at the service layer
//
// All store interfaces are designed around domain entities defined in the
// internal/domain package, ensuring that persistence concerns don't leak
// into the application's core business logic.
package store
