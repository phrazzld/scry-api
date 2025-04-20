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
// MemoStore defines the interface for memo data persistence.
// Version: 1.0
type MemoStore interface {
    // Create saves a new memo to the store.
    // It handles domain validation internally.
    // Returns validation errors from the domain Memo if data is invalid.
    Create(ctx context.Context, memo *domain.Memo) error

    // GetByID retrieves a memo by its unique ID.
    // Returns ErrMemoNotFound if the memo does not exist.
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error)

    // Update saves changes to an existing memo.
    // Returns ErrMemoNotFound if the memo does not exist.
    // Returns validation errors if the memo data is invalid.
    Update(ctx context.Context, memo *domain.Memo) error

    // UpdateStatus updates the status of an existing memo.
    // Returns ErrMemoNotFound if the memo does not exist.
    // Returns validation errors if the status is invalid.
    UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error

    // FindMemosByStatus retrieves all memos with the specified status.
    // Returns an empty slice if no memos match the criteria.
    // Can limit the number of results and paginate through offset.
    FindMemosByStatus(ctx context.Context, status domain.MemoStatus, limit, offset int) ([]*domain.Memo, error)

    // WithTx returns a new MemoStore instance that uses the provided transaction.
    // This allows for multiple operations to be executed within a single transaction.
    // The transaction should be created and managed by the caller (typically a service).
    WithTx(tx *sql.Tx) MemoStore
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
// CardStore defines the interface for card data persistence.
// Version: 1.0
type CardStore interface {
    // CreateMultiple saves multiple cards to the store in a single transaction.
    // All cards must be valid according to domain validation rules.
    // The transaction should be atomic - either all cards are created or none.
    // Returns validation errors if any card data is invalid.
    // May also create corresponding UserCardStats entries for each card
    // based on the implementation.
    CreateMultiple(ctx context.Context, cards []*domain.Card) error

    // GetByID retrieves a card by its unique ID.
    // Returns ErrCardNotFound if the card does not exist.
    // The returned card will have its Content field properly populated from JSONB.
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error)

    // UpdateContent modifies an existing card's content field.
    // Returns ErrCardNotFound if the card does not exist.
    // Returns validation errors if the content is invalid JSON.
    // Implementations should validate the content before updating.
    UpdateContent(ctx context.Context, id uuid.UUID, content []byte) error

    // Delete removes a card from the store by its ID.
    // Returns ErrCardNotFound if the card does not exist.
    // Depending on the implementation, this may also delete associated
    // UserCardStats entries via cascade delete in the database.
    Delete(ctx context.Context, id uuid.UUID) error

    // GetNextReviewCard retrieves the next card due for review for a user.
    // This is based on the UserCardStats.NextReviewAt field.
    // Returns ErrNotImplemented for stub implementations.
    // Returns ErrNotFound if there are no cards due for review.
    // This method may involve complex sorting/filtering logic based on SRS.
    GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error)

    // WithTx returns a new CardStore instance that uses the provided transaction.
    // This allows for multiple operations to be executed within a single transaction.
    // The transaction should be created and managed by the caller (typically a service).
    WithTx(tx *sql.Tx) CardStore
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
// UserCardStatsStore defines the interface for user card statistics data persistence.
// Version: 1.0
type UserCardStatsStore interface {
    // Get retrieves user card statistics by the combination of user ID and card ID.
    // Returns ErrUserCardStatsNotFound if the statistics entry does not exist.
    // This method retrieves a single entry that matches both IDs exactly.
    Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)

    // Update modifies an existing statistics entry.
    // It handles domain validation internally.
    // The userID and cardID fields in the stats object are used to identify the record to update.
    // Returns ErrUserCardStatsNotFound if the statistics entry does not exist.
    // Returns validation errors from the domain UserCardStats if data is invalid.
    Update(ctx context.Context, stats *domain.UserCardStats) error

    // Delete removes user card statistics by the combination of user ID and card ID.
    // Returns ErrUserCardStatsNotFound if the statistics entry does not exist.
    // This operation is permanent and cannot be undone.
    Delete(ctx context.Context, userID, cardID uuid.UUID) error

    // WithTx returns a new UserCardStatsStore instance that uses the provided transaction.
    // This allows for multiple operations to be executed within a single transaction.
    // The transaction should be created and managed by the caller (typically a service).
    WithTx(tx *sql.Tx) UserCardStatsStore
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
// UserStore defines the interface for user data persistence.
// Version: 1.0
type UserStore interface {
    // Create saves a new user to the store.
    // It handles domain validation and password hashing internally.
    // Returns ErrEmailExists if the email is already taken.
    // Returns validation errors from the domain User if data is invalid.
    Create(ctx context.Context, user *domain.User) error

    // GetByID retrieves a user by their unique ID.
    // Returns ErrUserNotFound if the user does not exist.
    // The returned user contains all fields except the plaintext password.
    GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

    // GetByEmail retrieves a user by their email address.
    // Returns ErrUserNotFound if the user does not exist.
    // The returned user contains all fields except the plaintext password.
    GetByEmail(ctx context.Context, email string) (*domain.User, error)

    // Update modifies an existing user's details.
    // The caller MUST provide a complete user object including HashedPassword.
    // If a new plain text Password is provided, it will be hashed and the HashedPassword will be updated.
    // Returns ErrUserNotFound if the user does not exist.
    // Returns ErrEmailExists if updating to an email that already exists.
    // Returns validation errors from the domain User if data is invalid.
    Update(ctx context.Context, user *domain.User) error

    // Delete removes a user from the store by their ID.
    // Returns ErrUserNotFound if the user does not exist.
    // This operation is permanent and cannot be undone.
    Delete(ctx context.Context, id uuid.UUID) error

    // WithTx returns a new UserStore instance that uses the provided transaction.
    // This allows for multiple operations to be executed within a single transaction.
    // The transaction should be created and managed by the caller (typically a service).
    WithTx(tx *sql.Tx) UserStore
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
// Common store errors used across all store implementations.
var (
    // ErrNotFound is returned when a requested entity does not exist in the store.
    // This is a generic version of the entity-specific not found errors
    // (e.g., ErrUserNotFound, ErrMemoNotFound).
    ErrNotFound = errors.New("entity not found")

    // ErrDuplicate is returned when an operation would create a duplicate
    // of a unique entity (e.g., a user with the same email).
    ErrDuplicate = errors.New("entity already exists")

    // ErrNotImplemented is returned when a store method is not yet implemented.
    // This is particularly useful for stub implementations.
    ErrNotImplemented = errors.New("method not implemented")

    // ErrInvalidEntity is returned when an entity fails validation before
    // being stored. Check the wrapped error for specific validation details.
    ErrInvalidEntity = errors.New("invalid entity")

    // ErrUpdateFailed is returned when an update operation fails, for example
    // because the entity does not exist or the update violates constraints.
    ErrUpdateFailed = errors.New("update failed")

    // ErrDeleteFailed is returned when a delete operation fails, for example
    // because the entity does not exist or is referenced by other entities.
    ErrDeleteFailed = errors.New("delete failed")

    // ErrTransactionFailed is returned when a database transaction fails
    // to commit or when an operation within a transaction fails.
    ErrTransactionFailed = errors.New("transaction failed")

    // Entity-specific "not found" errors

    // ErrUserNotFound indicates that the requested user does not exist in the store.
    ErrUserNotFound = fmt.Errorf("%w: user", ErrNotFound)

    // ErrMemoNotFound indicates that the requested memo does not exist in the store.
    ErrMemoNotFound = fmt.Errorf("%w: memo", ErrNotFound)

    // ErrCardNotFound indicates that the requested card does not exist in the store.
    ErrCardNotFound = fmt.Errorf("%w: card", ErrNotFound)

    // ErrUserCardStatsNotFound indicates that the requested user card stats do not exist in the store.
    ErrUserCardStatsNotFound = fmt.Errorf("%w: user card stats", ErrNotFound)

    // Entity-specific "duplicate" errors

    // ErrEmailExists indicates that a user with the given email already exists.
    // This is returned when attempting to create a user with an email that's already in use.
    ErrEmailExists = fmt.Errorf("%w: email", ErrDuplicate)
)
```

### Error Handling Patterns

1. **Check for general error types using errors.Is()**:
   ```go
   if errors.Is(err, store.ErrNotFound) {
       // Handle any not found case
   }
   ```

2. **Check for specific entity errors**:
   ```go
   if errors.Is(err, store.ErrUserNotFound) {
       // Handle user not found case specifically
   }
   ```

3. **Error wrapping for context**:
   ```go
   // Adding context while preserving the original error type
   if err != nil {
       return fmt.Errorf("failed to retrieve user %s: %w", userID, err)
   }
   ```

4. **Logging database details while returning opaque errors**:
   ```go
   // In store implementation
   var pgErr *pgconn.PgError
   if errors.As(err, &pgErr) {
       // Log detailed database error
       s.logger.Error("database error",
           "code", pgErr.Code,
           "message", pgErr.Message,
           "detail", pgErr.Detail)

       // Return standardized error without exposing details
       return store.ErrUpdateFailed
   }
   ```

5. **Error mapping using helpers**:
   ```go
   // Using error mapping helpers
   if err := s.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Email); err != nil {
       return nil, mapErrorToUserError(err, "failed to get user")
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
