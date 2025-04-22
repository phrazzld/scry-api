# Domain Models

This document describes the core domain models of the Scry API application, their relationships, and their purpose.

## Models Overview

### User

The `User` model represents a registered user of the application. It contains:
- Unique identifier (`ID`)
- Authentication details (`Email`, `HashedPassword`)
- Timestamps for creation and updates

Users can create Memos and review Cards that are generated from those Memos.

### Memo

The `Memo` model represents a text entry submitted by a user for generating flashcards. It contains:
- Unique identifier (`ID`)
- Reference to the user who created it (`UserID`)
- The actual memo text (`Text`)
- The processing status (`Status`)
- Timestamps for creation and updates

Memos can be in one of five states:
- `pending`: The memo has been submitted but not yet processed
- `processing`: The memo is currently being processed to generate cards
- `completed`: The memo has been successfully processed and all cards have been generated
- `completed_with_errors`: The memo was processed but some cards failed to be generated
- `failed`: The memo processing failed completely

### Card

The `Card` model represents a flashcard generated from a user's memo. It contains:
- Unique identifier (`ID`)
- References to the user (`UserID`) and memo (`MemoID`) it was generated from
- The content of the card (`Content`) as a flexible JSON structure
- Timestamps for creation and updates

Card content is stored as a JSON structure to allow for flexibility in the card format. The standard format includes:
- `front`: The question or prompt side of the card
- `back`: The answer side of the card
- `hint` (optional): A hint to help the user remember the answer
- `tags` (optional): Keywords or categories associated with the card
- `image_url` (optional): URL to an image associated with the card

### UserCardStats

The `UserCardStats` model tracks a user's spaced repetition statistics for a specific card. It contains:
- Composite key of `UserID` and `CardID`
- Spaced repetition algorithm parameters (`Interval`, `EaseFactor`, `ConsecutiveCorrect`)
- Review timing information (`LastReviewedAt`, `NextReviewAt`)
- Review statistics (`ReviewCount`)
- Timestamps for creation and updates

UserCardStats implements a modified version of the SM-2 spaced repetition algorithm to determine when cards should be reviewed based on the user's past performance.

## Relationships

The domain models have the following relationships:

1. **User-Memo**: One-to-many. A user can create multiple memos.
2. **Memo-Card**: One-to-many. A memo can generate multiple cards.
3. **User-Card**: One-to-many. A user owns multiple cards (generated from their memos).
4. **User-Card-Stats**: Many-to-many with attributes. A user has statistics for each of their cards, stored in the UserCardStats model.

## Domain Logic

The domain models encapsulate the following core business logic:

1. **User authentication**: The User model supports email-based authentication.
2. **Memo processing workflow**: The Memo model tracks the state of processing memos to generate cards.
3. **Spaced repetition scheduling**: The UserCardStats model implements the SM-2 algorithm to schedule card reviews.
4. **Card content management**: The Card model provides flexible storage for card content while maintaining references to the source memo.

Each model includes validation logic to ensure data integrity, and methods to support the required business operations (e.g., updating review statistics, changing memo status, etc.).

## Repository Pattern Standards

This section documents the standardized approach to implementing the repository pattern in the Scry API application.

### Core Principles

1. **Single Point of Data Access**: Repositories serve as the single point of access to data storage for domain entities
2. **Domain-Driven Design**: Repository interfaces are designed around domain concepts, not storage implementation details
3. **Clear Separation of Concerns**: Storage mechanisms are separated from business logic through repository abstractions
4. **Transaction Support**: All repositories support transactions for coordinating multiple operations
5. **Proper Error Handling**: Domain-specific errors with proper context are returned, not storage-specific errors

### Interface Standards

#### Naming Conventions

1. **Use *Store Suffix**: All repository interfaces must use the `*Store` naming pattern (e.g., `UserStore`, `CardStore`)
2. **Implementation Prefixes**: Concrete implementations use a prefix indicating the storage backend (e.g., `PostgresUserStore`)
3. **Method Naming**: Methods follow standard CRUD naming patterns:
   - `Create` / `CreateMultiple` for creation operations
   - `Get` or `GetByID` for retrieval by ID
   - `Update` / `UpdateContent` for modification operations
   - `Delete` for removal operations
   - `Find*` for query operations returning multiple entities

#### Interface Structure

Every repository interface should include:

1. **CRUD Operations**: Core methods for creating, reading, updating, and deleting entities
2. **Transaction Support**: A `WithTx` method that returns a new instance operating within a transaction
3. **DB Access**: A `DB()` method that returns the underlying database connection for transaction management
4. **Concurrency Protection**: `GetForUpdate` methods for entities that require optimistic locking
5. **Comprehensive Documentation**: Each method must include detailed documentation explaining:
   - Purpose and behavior
   - Parameters and return values
   - Error conditions and handling
   - Transaction requirements
   - Usage examples

Example interface template:

```go
// EntityStore defines the interface for entity data persistence.
type EntityStore interface {
    // Create saves a new entity to the store.
    // It handles domain validation internally.
    Create(ctx context.Context, entity *domain.Entity) error

    // GetByID retrieves an entity by its unique ID.
    // Returns ErrEntityNotFound if the entity does not exist.
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Entity, error)

    // Update saves changes to an existing entity.
    // Returns ErrEntityNotFound if the entity does not exist.
    Update(ctx context.Context, entity *domain.Entity) error

    // Delete removes an entity from the store by its ID.
    // Returns ErrEntityNotFound if the entity does not exist.
    Delete(ctx context.Context, id uuid.UUID) error

    // WithTx returns a new EntityStore instance that uses the provided transaction.
    // This allows for multiple operations to be executed within a single transaction.
    WithTx(tx *sql.Tx) EntityStore

    // DB returns the underlying database connection for transaction management.
    DB() *sql.DB
}
```

### Transaction Handling

Repositories must support transactions with the following requirements:

1. **Explicit Transaction Support**: All repositories must provide a `WithTx` method
2. **No Internal Transactions**: Repositories should not start/commit transactions internally
3. **Atomic Operations**: Complex operations should use transactions to ensure atomicity
4. **Proper Error Propagation**: Errors inside transactions must be propagated for proper rollback
5. **Standard Transaction Pattern**: Use the `store.RunInTransaction` helper for consistent transaction handling

Example transaction usage:

```go
// Example of proper transaction usage
err := store.RunInTransaction(ctx, entityStore.DB(), func(ctx context.Context, tx *sql.Tx) error {
    // Get transactional repositories
    txEntityStore := entityStore.WithTx(tx)
    txRelatedStore := relatedStore.WithTx(tx)

    // Execute operations within the transaction
    if err := txEntityStore.Create(ctx, entity); err != nil {
        return err
    }

    if err := txRelatedStore.Create(ctx, related); err != nil {
        return err
    }

    return nil
})
```

### Layering Guidelines

To maintain proper separation of concerns:

1. **No Service Awareness**: Store implementations must not be aware of service-layer interfaces
2. **No Cross-Repository Dependencies**: Repositories should not directly depend on other repositories
3. **Service Orchestration**: Services coordinate operations across multiple repositories
4. **Store as Primary Abstraction**: Services should depend directly on store interfaces when possible
5. **Domain-Focused Interfaces**: Store interfaces should be focused on domain entity operations

### Implementation Requirements

Concrete repository implementations must:

1. **Compile-Time Interface Checks**: Use compile-time interface checks to ensure implementation correctness
2. **Error Mapping**: Map storage-specific errors to domain-specific errors
3. **Proper Logging**: Include structured logging with appropriate context
4. **Validation**: Validate entities before persistence operations
5. **Consistent Returns**: Return domain models, not storage-specific models
6. **Transaction Propagation**: Properly implement the `WithTx` method to propagate transactions

Example implementation pattern:

```go
// Compile-time check to ensure implementation satisfies interface
var _ store.EntityStore = (*PostgresEntityStore)(nil)

// PostgresEntityStore implements the store.EntityStore interface
type PostgresEntityStore struct {
    db     store.DBTX
    logger *slog.Logger
}

// NewPostgresEntityStore creates a new PostgreSQL implementation of EntityStore
func NewPostgresEntityStore(db store.DBTX, logger *slog.Logger) *PostgresEntityStore {
    // Validate inputs
    if db == nil {
        panic("db cannot be nil")
    }

    // Use provided logger or create default
    if logger == nil {
        logger = slog.Default()
    }

    return &PostgresEntityStore{
        db:     db,
        logger: logger.With(slog.String("component", "entity_store")),
    }
}

// WithTx implements store.EntityStore.WithTx
func (s *PostgresEntityStore) WithTx(tx *sql.Tx) store.EntityStore {
    return &PostgresEntityStore{
        db:     tx,
        logger: s.logger,
    }
}

// Additional method implementations...
```

By following these guidelines, we ensure consistent, maintainable, and robust data access across the application.
