# T006 Analysis: Repository Interfaces

## Current Repository Structure

The codebase follows a multi-layered architecture with several types of repository-like interfaces:

1. **Store Interfaces**: The core persistence abstractions defined in `internal/store/`
   - Example: `CardStore`, `MemoStore`, `UserCardStatsStore`
   - These define low-level data access operations
   - Methods closely match the domain entities (Create, Get, Update, Delete)
   - All include transaction support via `WithTx` methods

2. **Service-Layer Repository Interfaces**: Defined in service packages
   - Example: `CardRepository`, `MemoRepository` in `internal/service/`
   - Additional specialized repositories like `UserCardStatsRepository` in `card_review` package
   - Subset of store interface methods, tailored to service needs
   - Include transaction support with `WithTx` and `DB()` methods

3. **Repository Adapters**: Connect service repositories to stores
   - Example: `cardRepositoryAdapter`, `MemoRepositoryAdapter`
   - Simple pass-through implementations
   - Adapt store interfaces to service repository interfaces
   - Handle transaction propagation

4. **Store Implementations**: Concrete implementations of store interfaces
   - Example: `PostgresCardStore`, `PostgresMemoStore`
   - Implement store interfaces with PostgreSQL-specific code
   - Include proper error mapping, validation, and logging
   - Support transaction propagation with `WithTx` methods

## Inconsistencies and Redundancies

1. **Naming Inconsistencies**:
   - Two different naming patterns: `*Store` vs `*Repository`
   - Some adapters are exported (e.g., `MemoRepositoryAdapter`), others are not (e.g., `cardRepositoryAdapter`)
   - Inconsistent method naming for transactions: `WithTx` vs `WithTxCardStore`
   - Some repositories include `DB()` method, others don't

2. **Interface Redundancy**:
   - Many service repository interfaces are near-duplicates of store interfaces
   - Repository adapters often just delegate to the underlying store with minimal logic
   - Multiple repository types for the same entity (e.g., `CardRepository` in service vs `CardRepository` in card_review)

3. **Transaction Handling Inconsistencies**:
   - Some services manage transactions themselves, others use `store.RunInTransaction`
   - Inconsistent way of creating transaction-aware repositories

4. **Method Duplication**:
   - `PostgresCardStore` has both `WithTxCardStore` and `WithTx` methods that are nearly identical
   - `GetForUpdate` method isn't consistently available across all repositories

5. **Layering Violations**:
   - Some store implementations have direct awareness of service-layer interfaces
   - `WithTx` method in `PostgresCardStore` (lines 378-386) is specifically for compatibility with `task.CardRepository`
   - This creates a dependency from platform layer to service layer

## Target Standardization Pattern

Based on the codebase analysis, the optimal repository pattern should:

1. **Simplify the Repository Hierarchy**:
   - Eliminate redundant adapter layers where possible
   - Let services depend directly on store interfaces instead of creating service-specific repository interfaces
   - Use store interfaces as the primary abstraction, with consistent naming

2. **Standardize Interface Design**:
   - Adopt consistent naming convention (prefer `*Store` throughout)
   - Ensure consistent method signatures across all repository types
   - Standardize transaction methods (`WithTx`) across all repositories

3. **Improve Transaction Handling**:
   - Use a consistent transaction pattern throughout the codebase
   - Standardize on `store.RunInTransaction` for transaction orchestration
   - Ensure all repositories properly participate in transactions

4. **Eliminate Layering Violations**:
   - Remove direct awareness of service interfaces from store implementations
   - Maintain clear separation between persistence layer and service layer

5. **Encourage Repository Independence**:
   - Each repository should be focused on a single domain entity
   - Repositories should not depend on each other
   - Service layer coordinates access to multiple repositories when needed

## Specific Recommendations

1. **Naming and Method Standardization**:
   - Rename all repository interfaces to use `*Store` for consistency
   - Standardize on `WithTx` method name (remove `WithTxCardStore`)
   - Ensure all store interfaces include `DB()` method for transaction support

2. **Repository Interface Consolidation**:
   - Remove service-specific repository interfaces
   - Have services depend directly on store interfaces
   - Only create specialized repository interfaces when truly needed

3. **Transaction Handling Improvements**:
   - Add `GetForUpdate` method to all store interfaces that need concurrency protection
   - Standardize transaction orchestration with `RunInTransaction`

4. **Layering Improvements**:
   - Remove service-layer awareness from store implementations
   - Use package-level interfaces to maintain proper dependency direction

This standardization will simplify the codebase, reduce redundancy, and make it easier to understand and maintain the repository pattern implementation.
