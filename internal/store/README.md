# Store Package

This package defines the storage interfaces used throughout the Scry API application. It follows the Repository Pattern to provide a clean separation between business logic and data access concerns.

## Design Philosophy

The store package is designed around these core principles:

1. **Interface-Based Design**: All data access is defined through interfaces, allowing for multiple implementations and easier testing.
2. **Domain-Driven**: Each interface corresponds to a specific domain entity.
3. **Clear Error Contracts**: Well-defined error types for common scenarios.
4. **Context-Aware**: All operations support context for cancellation and timeouts.
5. **Transaction Support**: Unified transaction handling through the DBTX interface.

## Core Interfaces

### DBTX Interface

This interface unifies database connections and transactions to allow methods to work with either:

```go
// DBTX represents either a database connection or a transaction
type DBTX interface {
    ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
    PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
    QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}
```

### MemoStore

Manages memo operations in the database.

```go
type MemoStore interface {
    Create(ctx context.Context, memo *domain.Memo) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error
    FindMemosByStatus(ctx context.Context, status domain.MemoStatus, limit, offset int) ([]*domain.Memo, error)
}
```

**Usage Example**:
```go
// Create a new memo
memo := &domain.Memo{
    ID:      uuid.New(),
    UserID:  userID,
    Content: "Memo content",
    Status:  domain.MemoStatusDraft,
}
err := memoStore.Create(ctx, memo)

// Get a memo by ID
memo, err := memoStore.GetByID(ctx, memoID)

// Update memo status
err := memoStore.UpdateStatus(ctx, memoID, domain.MemoStatusReady)

// Find memos by status
memos, err := memoStore.FindMemosByStatus(ctx, domain.MemoStatusReady, 10, 0)
```

### CardStore

Manages flashcard operations in the database.

```go
type CardStore interface {
    CreateMultiple(ctx context.Context, cards []*domain.Card) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error)
    UpdateContent(ctx context.Context, id uuid.UUID, content json.RawMessage) error
    Delete(ctx context.Context, id uuid.UUID) error
    GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error)
}
```

**Usage Example**:
```go
// Create multiple cards in a batch
cards := []*domain.Card{
    {
        ID:      uuid.New(),
        MemoID:  memoID,
        Content: json.RawMessage(`{"front": "Question", "back": "Answer"}`),
    },
    // More cards...
}
err := cardStore.CreateMultiple(ctx, cards)

// Get a card by ID
card, err := cardStore.GetByID(ctx, cardID)

// Update card content
newContent := json.RawMessage(`{"front": "Updated question", "back": "Updated answer"}`)
err := cardStore.UpdateContent(ctx, cardID, newContent)

// Delete a card
err := cardStore.Delete(ctx, cardID)
```

### UserCardStatsStore

Manages statistics for user-card interactions, used by the spaced repetition system.

```go
type UserCardStatsStore interface {
    Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)
    Update(ctx context.Context, stats *domain.UserCardStats) error
    Delete(ctx context.Context, userID, cardID uuid.UUID) error
}
```

**Usage Example**:
```go
// Get stats for a specific user-card pair
stats, err := statsStore.Get(ctx, userID, cardID)

// Update stats after a review
stats.Difficulty = 3
stats.Interval = 2
stats.LastReviewedAt = time.Now()
err := statsStore.Update(ctx, stats)

// Delete stats
err := statsStore.Delete(ctx, userID, cardID)
```

### UserStore

Manages user operations in the database.

```go
type UserStore interface {
    Create(ctx context.Context, user *domain.User) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
    GetByEmail(ctx context.Context, email string) (*domain.User, error)
    Update(ctx context.Context, user *domain.User) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

**Usage Example**:
```go
// Create a new user
user := &domain.User{
    ID:       uuid.New(),
    Email:    "user@example.com",
    Password: hashedPassword,
}
err := userStore.Create(ctx, user)

// Get a user by ID
user, err := userStore.GetByID(ctx, userID)

// Get a user by email
user, err := userStore.GetByEmail(ctx, "user@example.com")

// Update a user
user.Email = "new@example.com"
// Note: When updating, a complete user object (including HashedPassword) must be provided
// The HashedPassword must already be populated from a previous Get operation or properly hashed
err := userStore.Update(ctx, user)

// Delete a user
err := userStore.Delete(ctx, userID)
```

## Error Handling

The store package defines several standard error types to provide consistent error handling across implementations:

```go
// ErrNotFound is returned when a requested entity is not found
var ErrNotFound = errors.New("entity not found")

// ErrDuplicate is returned when a unique constraint is violated
var ErrDuplicate = errors.New("duplicate entity")

// ErrNotImplemented is returned for operations not yet implemented
var ErrNotImplemented = errors.New("operation not implemented")

// Entity-specific errors
var ErrUserNotFound = fmt.Errorf("%w: user", ErrNotFound)
var ErrMemoNotFound = fmt.Errorf("%w: memo", ErrNotFound)
var ErrCardNotFound = fmt.Errorf("%w: card", ErrNotFound)
var ErrUserCardStatsNotFound = fmt.Errorf("%w: user card stats", ErrNotFound)
```

### Error Handling Patterns

1. **Check for specific errors using errors.Is()**:
   ```go
   if errors.Is(err, store.ErrNotFound) {
       // Handle not found case
   }
   ```

2. **Check for more specific entity errors**:
   ```go
   if errors.Is(err, store.ErrUserNotFound) {
       // Handle user not found case
   }
   ```

3. **Unwrap errors to get the original cause**:
   ```go
   var pgErr *pgconn.PgError
   if errors.As(err, &pgErr) {
       // Handle specific PostgreSQL error
   }
   ```

## Transaction Management

The Scry API uses a consistent pattern for transaction management, with clear ownership and boundaries.

### The WithTx Pattern

All store interfaces include a `WithTx` method that allows the caller to create a transactional version of the store:

```go
// WithTx returns a new UserStore instance that uses the provided transaction.
// This allows for multiple operations to be executed within a single transaction.
// The transaction should be created and managed by the caller (typically a service).
WithTx(tx *sql.Tx) UserStore
```

Store implementations create a new instance with the same configuration but use the transaction for database operations:

```go
func (s *PostgresUserStore) WithTx(tx *sql.Tx) store.UserStore {
    return &PostgresUserStore{
        db:         tx,
        bcryptCost: s.bcryptCost,
    }
}
```

### Transaction Ownership

1. **Service Layer Ownership**: The service layer owns transaction boundaries, not the store layer.
2. **Single Transaction, Multiple Stores**: A single transaction can span multiple store operations.
3. **Explicit Transaction Management**: Transactions are explicitly started, committed, or rolled back.

### Helper Function: RunInTransaction

The `RunInTransaction` helper provides a clean pattern for transaction management:

```go
// RunInTransaction executes the given function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func RunInTransaction(ctx context.Context, db *sql.DB, fn TxFn) error {
    // TxFn is a function that executes within a transaction
    // type TxFn func(ctx context.Context, tx *sql.Tx) error

    // Implementation details...
}
```

### Transaction Usage Patterns

#### Basic Transaction Pattern

```go
err := store.RunInTransaction(ctx, s.db, func(ctx context.Context, tx *sql.Tx) error {
    // Get transactional store instances
    txUserStore := s.userStore.WithTx(tx)
    txMemoStore := s.memoStore.WithTx(tx)

    // Perform operations within the transaction
    if err := txUserStore.Create(ctx, user); err != nil {
        return err
    }

    if err := txMemoStore.Create(ctx, memo); err != nil {
        return err
    }

    return nil // Success - transaction will be committed
})
```

#### Complex Service-Level Transaction Example

```go
func (s *MemoService) UpdateMemoStatus(ctx context.Context, memoID uuid.UUID, status domain.MemoStatus) error {
    return store.RunInTransaction(ctx, s.memoRepo.DB(), func(ctx context.Context, tx *sql.Tx) error {
        // Get a transactional repo
        txRepo := s.memoRepo.WithTx(tx)

        // 1. Retrieve the memo
        memo, err := txRepo.GetByID(ctx, memoID)
        if err != nil {
            return fmt.Errorf("failed to retrieve memo for status update: %w", err)
        }

        // 2. Update the memo's status (domain logic)
        err = memo.UpdateStatus(status)
        if err != nil {
            return fmt.Errorf("invalid status transition: %w", err)
        }

        // 3. Save the updated memo
        return txRepo.Update(ctx, memo)
    })
}
```

### Transaction Best Practices

1. **Always Use RunInTransaction**: Prefer using the `RunInTransaction` helper over manual transaction management.
2. **Defer Rollback**: When managing transactions manually, always defer a rollback call.
3. **Clear Error Handling**: Properly wrap and propagate errors from transactional operations.
4. **Minimize Transaction Scope**: Keep transactions as short as possible to reduce lock contention.
5. **Error Mapping**: Map all database errors to domain-specific errors within store implementations.
6. **Business Logic Separation**: Business logic should live in the service layer or domain model, not in stores.

### Atomicity Guarantees

The transaction pattern ensures atomic operations. If any step in a transaction fails:

1. All changes made within the transaction are rolled back
2. No data is persisted to the database
3. The system remains in a consistent state

This is critical for operations that must succeed or fail as a unit, such as creating a user and their associated profile, or updating multiple related records.
