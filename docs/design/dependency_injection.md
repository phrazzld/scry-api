# Dependency Injection Approach

This document outlines the dependency injection (DI) approach used in the Scry API project, providing guidelines for consistent implementation and proper repository injection.

## Core Principles

1. **Constructor-Based Injection**: Dependencies are provided through constructors rather than being created inside components.
2. **Interface-Based Dependencies**: Components depend on interfaces rather than concrete implementations.
3. **Main.go as Composition Root**: All dependency creation and wiring occurs in main.go or its helper functions.
4. **Adapter Pattern**: Adapters are used to bridge between different layers without creating circular dependencies.

## Dependency Flow

The dependency graph flows in a specific direction:

```
main.go (composition root)
  │
  ├── creates concrete store implementations (postgres)
  │
  ├── creates adapters when needed
  │
  ├── creates services with repositories/stores
  │
  └── creates API handlers with services
```

## Repository Injection Guidelines

### 1. All concrete store implementations are created in main.go

```go
// Example from startServer function in main.go
userStore := postgres.NewPostgresUserStore(db, bcrypt.DefaultCost)
taskStore := postgres.NewPostgresTaskStore(db)
memoStore := postgres.NewPostgresMemoStore(db, logger)
cardStore := postgres.NewPostgresCardStore(db, logger)
userCardStatsStore := postgres.NewPostgresUserCardStatsStore(db, logger)
```

### 2. Repository adapters are created in main.go, not in router setup

```go
// Correct approach (in startServer):
memoRepoAdapter := service.NewMemoRepositoryAdapter(deps.MemoStore, deps.DB)
cardRepoAdapter := service.NewCardRepositoryAdapter(deps.CardStore, deps.DB)
statsRepoAdapter := service.NewStatsRepositoryAdapter(deps.UserCardStatsStore)

// Incorrect approach (seen in setupRouter):
memoRepoAdapter := service.NewMemoRepositoryAdapter(deps.MemoStore, deps.DB)
```

### 3. Services receive repository interfaces, not concrete implementations

```go
// Good example - card review service depends on store interfaces
func NewCardReviewService(
    cardStore store.CardStore,
    statsStore store.UserCardStatsStore,
    srsService srs.Service,
    logger *slog.Logger,
) (CardReviewService, error) {
    // ...
}
```

### 4. Adapters bridge between different interface expectations

Repository adapters:
- Make store implementations compatible with service requirements
- Add transaction handling capabilities
- Expose the database connection for transactions
- Prevent circular dependencies between packages

```go
// Example adapter
type cardRepositoryAdapter struct {
    cardStore store.CardStore
    db        *sql.DB
}

// WithTx supports transaction propagation
func (a *cardRepositoryAdapter) WithTx(tx *sql.Tx) CardRepository {
    return &cardRepositoryAdapter{
        cardStore: a.cardStore.WithTx(tx),
        db:        a.db,
    }
}
```

## Best Practices

1. **Verify Interface Compliance**: Use compile-time checks with type assertions:
   ```go
   var _ CardReviewService = (*cardReviewServiceImpl)(nil)
   ```

2. **Services Should Check Dependencies**: Services must validate that injected dependencies are not nil:
   ```go
   if cardStore == nil {
       return nil, fmt.Errorf("cardStore cannot be nil")
   }
   ```

3. **Propagate Transactions**: All repositories must support transaction propagation through WithTx methods:
   ```go
   func (s *store) WithTx(tx *sql.Tx) SomeStore {
       return &store{db: tx, /* other fields */}
   }
   ```

4. **Use The Adapter Pattern**: When there's a mismatch between interfaces, create adapters:
   ```go
   // Adapter makes StoreA conform to RepositoryB interface
   func NewRepositoryBAdapter(storeA StoreA) RepositoryB {
       return &repositoryBAdapter{store: storeA}
   }
   ```

## Adaptation Needed

Based on the code review, the following adaptation is needed:

1. **Move adapter creation out of setupRouter**:

   The `setupRouter` function in main.go currently creates adapters:

   ```go
   // In setupRouter:
   memoRepoAdapter := service.NewMemoRepositoryAdapter(deps.MemoStore, deps.DB)
   ```

   This should be moved to the `startServer` function and passed via the dependencies struct.

2. **Ensure handlers follow constructor injection**:

   All handlers should receive their dependencies through constructors, not create them internally.
