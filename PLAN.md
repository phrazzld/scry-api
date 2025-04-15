# User Store Implementation Plan

This plan outlines the implementation of the User Store component for the Scry API authentication system, adhering to the project's standards for simplicity, modularity, testability, and maintainability.

## Overview

The User Store will provide a data access layer for user management, including:
- A clear interface definition for CRUD operations
- A PostgreSQL implementation of this interface
- Secure password handling with bcrypt
- Comprehensive validation
- Thorough testing

This implementation follows the standard interface/implementation pattern with direct SQL for maximum control, clarity, and alignment with testing principles.

## Implementation Steps

### 1. Define Store Interface and Errors

**Location:** `internal/store/user.go`

```go
package store

import (
    "context"
    "errors"

    "github.com/google/uuid"
    "github.com/phrazzld/scry-api/internal/domain"
)

// Common store errors
var (
    ErrUserNotFound = errors.New("user not found")
    ErrEmailExists  = errors.New("email already exists")
)

// UserStore defines the interface for user data persistence.
type UserStore interface {
    // Create saves a new user to the store.
    // Returns ErrEmailExists if the email is already taken.
    Create(ctx context.Context, user *domain.User) error

    // GetByID retrieves a user by their unique ID.
    // Returns ErrUserNotFound if the user does not exist.
    GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

    // GetByEmail retrieves a user by their email address.
    // Returns ErrUserNotFound if the user does not exist.
    GetByEmail(ctx context.Context, email string) (*domain.User, error)

    // Update modifies an existing user's details.
    // Returns ErrUserNotFound if the user does not exist.
    // Returns ErrEmailExists if updating to an email that already exists.
    Update(ctx context.Context, user *domain.User) error

    // Delete removes a user from the store by their ID.
    // Returns ErrUserNotFound if the user does not exist.
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### 2. Implement PostgreSQL User Store

**Location:** `internal/platform/postgres/user_store.go`

```go
package postgres

import (
    "context"
    "database/sql"
    "errors"
    "log/slog"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/phrazzld/scry-api/internal/domain"
    "github.com/phrazzld/scry-api/internal/store"
    "golang.org/x/crypto/bcrypt"
)

const uniqueViolationCode = "23505" // PostgreSQL unique violation error code

type PostgresUserStore struct {
    db *sql.DB
}

// NewPostgresUserStore creates a new user store implementation.
func NewPostgresUserStore(db *sql.DB) *PostgresUserStore {
    return &PostgresUserStore{db: db}
}

// Implement Create, GetByID, GetByEmail, Update, Delete methods...
```

Implementation details for each method:

1. **Create**:
   - Hash the user's password using bcrypt
   - Insert user into the database
   - Handle unique constraint violations (email already exists)

2. **GetByID/GetByEmail**:
   - Query the database using parameterized queries
   - Map database rows to domain model
   - Handle "not found" cases appropriately

3. **Update**:
   - Update user details in the database
   - Rehash password if changed
   - Handle unique constraint violations

4. **Delete**:
   - Delete user from the database
   - Handle "not found" cases

### 3. Password Hashing Implementation

**Location:** Within the PostgresUserStore implementation

```go
// Example for Create method - Password hashing
func (s *PostgresUserStore) Create(ctx context.Context, user *domain.User) error {
    // Validate user
    if err := user.Validate(); err != nil {
        return err
    }

    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
    if err != nil {
        slog.ErrorContext(ctx, "Failed to hash password", "error", err)
        return err
    }

    // Set hashed password
    user.HashedPassword = string(hashedPassword)
    user.Password = "" // Clear plaintext password

    // SQL insert implementation...
}
```

### 4. Data Validation

Validation should occur in multiple layers:

1. **Domain Model Validation**: The `User` struct in the domain package should have a `Validate()` method that checks:
   - Email format (already implemented)
   - Password complexity requirements
   - Other business rules

2. **Store-Level Validation**: The store implementation should:
   - Call `user.Validate()` before database operations
   - Leverage database constraints for additional validation

### 5. Testing Implementation

**Location:** `internal/platform/postgres/user_store_test.go`

Testing will focus on integration tests with a real PostgreSQL database, following the project's testing strategy for data stores:

```go
package postgres_test

import (
    "context"
    "database/sql"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/phrazzld/scry-api/internal/domain"
    "github.com/phrazzld/scry-api/internal/platform/postgres"
    "github.com/phrazzld/scry-api/internal/store"
    "github.com/phrazzld/scry-api/internal/testutils"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Test helpers and test cases for each method...
```

Test cases should cover:
- Successful operations
- Error conditions (not found, duplicate email, etc.)
- Data integrity verification
- Password hashing verification

## Rationale for This Approach

This plan was chosen because it:

1. **Prioritizes Simplicity (CORE_PRINCIPLES.md)**:
   - Clear interface definition with focused responsibilities
   - Direct SQL for explicit control and transparency
   - No unnecessary abstractions or dependencies

2. **Ensures Separation of Concerns (ARCHITECTURE_GUIDELINES.md)**:
   - Interface defined in core application layer
   - Implementation details isolated in platform layer
   - Follows Dependency Inversion Principle

3. **Optimizes for Testability (TESTING_STRATEGY.md)**:
   - Integration tests against real PostgreSQL
   - Minimal mocking (only at external boundaries)
   - Tests verify behavior, not implementation details

4. **Follows Coding Standards (CODING_STANDARDS.md)**:
   - Strong typing with explicit error handling
   - Consistent error types and handling
   - Clear function signatures and documentation

5. **Supports Documentation (DOCUMENTATION_APPROACH.md)**:
   - Interface serves as contract documentation
   - SQL queries are self-documenting regarding database interaction
   - Comments explain "why" not just "what"

## Risks and Mitigations

1. **Risk**: SQL injection vulnerabilities
   **Mitigation**: Use parameterized queries consistently

2. **Risk**: Password security vulnerabilities
   **Mitigation**: Use industry-standard bcrypt with appropriate cost factor

3. **Risk**: Inadequate validation allowing invalid data
   **Mitigation**: Multi-layered validation (domain model, store, database constraints)

4. **Risk**: Database connection management issues
   **Mitigation**: Proper resource cleanup, appropriate connection pool settings

This approach provides a robust, maintainable implementation that aligns with project standards while prioritizing security and correctness for this critical authentication component.
