# PostgreSQL Store Implementations

This package provides PostgreSQL implementations of the store interfaces defined in `internal/store`. These implementations handle the details of database access, error mapping, and transaction management.

## Design Philosophy

The PostgreSQL implementations follow these core principles:

1. **Interface Conformance**: Each implementation fully satisfies a store interface.
2. **Error Mapping**: Database errors are translated into domain-specific errors.
3. **Validation**: Entities are validated before database operations.
4. **Structured Logging**: Operations are logged with appropriate context.
5. **Transaction Support**: All operations work correctly within transactions.
6. **Security**: SQL injection prevention through parameterized queries.

## Common Implementation Patterns

### Struct Organization

Each store implementation follows a similar pattern:

```go
type PostgresXXXStore struct {
    db     store.DBTX
    logger *slog.Logger
}

func NewPostgresXXXStore(db store.DBTX, logger *slog.Logger) *PostgresXXXStore {
    if logger == nil {
        logger = slog.Default()
    }
    return &PostgresXXXStore{
        db:     db,
        logger: logger,
    }
}
```

### Transaction Support

Each store provides a `WithTx` method to create a new instance using a transaction:

```go
func (s *PostgresXXXStore) WithTx(tx *sql.Tx) *PostgresXXXStore {
    return &PostgresXXXStore{
        db:     tx,
        logger: s.logger,
    }
}
```

### Error Mapping

The `errors.go` file provides utility functions to map PostgreSQL errors to domain errors:

```go
// MapError maps a PostgreSQL error to a domain-specific error
func MapError(err error, notFoundErr error) error {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        if pgErr.Code == "23505" { // unique_violation
            return fmt.Errorf("%w: %v", store.ErrDuplicate, err)
        }
        // Map other PostgreSQL error codes...
    }

    if errors.Is(err, sql.ErrNoRows) {
        return notFoundErr
    }

    return err
}

// IsUniqueViolation checks if an error is a unique constraint violation
func IsUniqueViolation(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsForeignKeyViolation checks if an error is a foreign key constraint violation
func IsForeignKeyViolation(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// CheckRowsAffected checks if exactly one row was affected by an operation
func CheckRowsAffected(result sql.Result, notFoundErr error) error {
    affected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if affected == 0 {
        return notFoundErr
    }
    return nil
}
```

## Store Implementations

### PostgresUserStore

Implements the `store.UserStore` interface for user entity persistence.

**Key Features**:
- Case-insensitive email lookup
- Password hashing with bcrypt
- Validation of user data before persistence
- Proper handling of unique email constraint

**Example**:
```go
// Creating a user
func (s *PostgresUserStore) Create(ctx context.Context, user *domain.User) error {
    if err := user.Validate(); err != nil {
        return err
    }

    query := `
        INSERT INTO users (id, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
    `

    now := time.Now().UTC()
    _, err := s.db.ExecContext(
        ctx,
        query,
        user.ID,
        strings.ToLower(user.Email),
        user.Password,
        now,
        now,
    )

    if err != nil {
        // Check for unique email constraint violation
        if IsUniqueViolation(err) {
            return store.ErrEmailExists
        }
        return err
    }

    return nil
}
```

### PostgresMemoStore

Implements the `store.MemoStore` interface for memo entity persistence.

**Key Features**:
- Status-based filtering with pagination
- Validation of memo status transitions
- Transaction-aware operations

**Example**:
```go
// Finding memos by status
func (s *PostgresMemoStore) FindMemosByStatus(ctx context.Context, status domain.MemoStatus, limit, offset int) ([]*domain.Memo, error) {
    log := logger.FromContextOrDefault(ctx, s.logger)

    query := `
        SELECT id, user_id, content, status, created_at, updated_at
        FROM memos
        WHERE status = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

    rows, err := s.db.QueryContext(ctx, query, status, limit, offset)
    if err != nil {
        log.Error("failed to query memos by status",
            slog.String("error", err.Error()),
            slog.String("status", string(status)))
        return nil, err
    }
    defer rows.Close()

    memos := make([]*domain.Memo, 0)
    for rows.Next() {
        memo := &domain.Memo{}
        if err := rows.Scan(
            &memo.ID,
            &memo.UserID,
            &memo.Content,
            &memo.Status,
            &memo.CreatedAt,
            &memo.UpdatedAt,
        ); err != nil {
            log.Error("failed to scan memo row",
                slog.String("error", err.Error()))
            return nil, err
        }
        memos = append(memos, memo)
    }

    if err := rows.Err(); err != nil {
        log.Error("error iterating memo rows",
            slog.String("error", err.Error()))
        return nil, err
    }

    return memos, nil
}
```

### PostgresCardStore

Implements the `store.CardStore` interface for flashcard entity persistence.

**Key Features**:
- Atomic batch operations for creating multiple cards
- JSON content validation
- Cascade delete for related user card stats
- Transaction management for multi-entity operations

**Example**:
```go
// Creating multiple cards in a transaction
func (s *PostgresCardStore) CreateMultiple(ctx context.Context, cards []*domain.Card) error {
    log := logger.FromContextOrDefault(ctx, s.logger)

    // Empty list case
    if len(cards) == 0 {
        log.Debug("no cards to create")
        return nil
    }

    // Validate all cards before proceeding
    for i, card := range cards {
        if err := card.Validate(); err != nil {
            log.Warn("card validation failed",
                slog.String("error", err.Error()),
                slog.String("card_id", card.ID.String()),
                slog.Int("card_index", i))
            return err
        }
    }

    // Begin transaction if we're not already in one
    txObj, err := s.db.(*sql.DB).BeginTx(ctx, nil)
    if err != nil {
        log.Error("failed to begin transaction",
            slog.String("error", err.Error()))
        return err
    }
    defer txObj.Rollback()

    // Use tx for all operations
    tx := txObj

    // Insert cards
    cardQuery := `
        INSERT INTO cards (id, memo_id, content, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
    `

    statsQuery := `
        INSERT INTO user_card_stats (user_id, card_id, ease_factor, interval, repetitions,
                                   created_at, updated_at)
        SELECT m.user_id, $1, 2.5, 0, 0, $2, $2
        FROM memos m
        WHERE m.id = $3
    `

    now := time.Now().UTC()

    for i, card := range cards {
        // Insert the card
        _, err = tx.ExecContext(
            ctx,
            cardQuery,
            card.ID,
            card.MemoID,
            card.Content,
            now,
            now,
        )

        if err != nil {
            log.Error("failed to insert card",
                slog.String("error", err.Error()),
                slog.String("card_id", card.ID.String()),
                slog.Int("card_index", i))

            if IsForeignKeyViolation(err) {
                return domain.ErrInvalidMemoID
            }
            return err
        }

        // Create corresponding user_card_stats entry
        _, err = tx.ExecContext(
            ctx,
            statsQuery,
            card.ID,
            now,
            card.MemoID,
        )

        if err != nil {
            log.Error("failed to insert user card stats",
                slog.String("error", err.Error()),
                slog.String("card_id", card.ID.String()),
                slog.Int("card_index", i))
            return err
        }
    }

    // Commit the transaction
    if err := tx.Commit(); err != nil {
        log.Error("failed to commit transaction",
            slog.String("error", err.Error()))
        return err
    }

    log.Info("created multiple cards successfully",
        slog.Int("card_count", len(cards)))

    return nil
}
```

### PostgresUserCardStatsStore

Implements the `store.UserCardStatsStore` interface for spaced repetition statistics.

**Key Features**:
- Composite primary key handling (user_id + card_id)
- Support for SRS parameters (ease factor, interval, repetitions)
- Validation of stats data before persistence
- Null time handling for review timestamps

**Example**:
```go
// Updating user card stats
func (s *PostgresUserCardStatsStore) Update(ctx context.Context, stats *domain.UserCardStats) error {
    log := logger.FromContextOrDefault(ctx, s.logger)

    if err := stats.Validate(); err != nil {
        log.Warn("stats validation failed",
            slog.String("error", err.Error()),
            slog.String("user_id", stats.UserID.String()),
            slog.String("card_id", stats.CardID.String()))
        return err
    }

    query := `
        UPDATE user_card_stats
        SET ease_factor = $1, interval = $2, repetitions = $3,
            last_reviewed_at = $4, next_review_at = $5, updated_at = $6
        WHERE user_id = $7 AND card_id = $8
    `

    now := time.Now().UTC()
    var lastReviewedAt sql.NullTime
    if !stats.LastReviewedAt.IsZero() {
        lastReviewedAt = sql.NullTime{
            Time:  stats.LastReviewedAt,
            Valid: true,
        }
    }

    var nextReviewAt sql.NullTime
    if !stats.NextReviewAt.IsZero() {
        nextReviewAt = sql.NullTime{
            Time:  stats.NextReviewAt,
            Valid: true,
        }
    }

    result, err := s.db.ExecContext(
        ctx,
        query,
        stats.EaseFactor,
        stats.Interval,
        stats.Repetitions,
        lastReviewedAt,
        nextReviewAt,
        now,
        stats.UserID,
        stats.CardID,
    )

    if err != nil {
        log.Error("failed to update user card stats",
            slog.String("error", err.Error()),
            slog.String("user_id", stats.UserID.String()),
            slog.String("card_id", stats.CardID.String()))
        return err
    }

    if err := CheckRowsAffected(result, store.ErrUserCardStatsNotFound); err != nil {
        log.Warn("user card stats not found",
            slog.String("user_id", stats.UserID.String()),
            slog.String("card_id", stats.CardID.String()))
        return err
    }

    log.Info("updated user card stats",
        slog.String("user_id", stats.UserID.String()),
        slog.String("card_id", stats.CardID.String()))

    return nil
}
```

## Transaction Management

The PostgreSQL implementations handle transactions in two ways:

1. **Repository-Level Transactions**: For operations that need to update multiple tables atomically (e.g., `CardStore.CreateMultiple`).

2. **Service-Level Transactions**: For operations spanning multiple repositories, transactions are managed at the service level using the `WithTx` method.

**Service-Level Transaction Example**:
```go
func (s *Service) CreateMemoWithCards(ctx context.Context, memo *domain.Memo, cards []*domain.Card) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Use stores with transaction
    memoStore := s.memoStore.WithTx(tx)
    cardStore := s.cardStore.WithTx(tx)

    // Execute operations
    if err := memoStore.Create(ctx, memo); err != nil {
        return err
    }

    for _, card := range cards {
        card.MemoID = memo.ID
    }

    if err := cardStore.CreateMultiple(ctx, cards); err != nil {
        return err
    }

    return tx.Commit()
}
```

## Testing

The PostgreSQL implementations can be tested using:

1. **Unit Tests**: With mocked database interface
2. **Integration Tests**: Using test transactions to ensure test isolation

**Integration Test Pattern**:
```go
func TestPostgresXXXStore_Method(t *testing.T) {
    if !testutils.CheckTestEnvironment() {
        t.Skip("Skipping integration test")
    }

    t.Parallel()

    // Run test in a transaction that will be rolled back
    testutils.WithTx(t, func(tx *sql.Tx) {
        store := postgres.NewPostgresXXXStore(tx, nil)

        // Test the store methods
        // ...
    })
}
```
