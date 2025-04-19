package postgres_test

import (
	"context"
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

// TestUserStoreErrorLeakage tests that UserStore operations do not leak internal
// database error details in their returned errors.
func TestUserStoreErrorLeakage(t *testing.T) {
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test")
	}

	// We need to run real database tests to trigger actual PostgreSQL errors
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create a transaction for isolation
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		_ = tx.Rollback() // Intentionally ignoring error as it's cleanup code
	}()

	// Create the store
	userStore := postgres.NewPostgresUserStore(tx, 10)

	// Tests for Create operation
	t.Run("Create errors do not leak details", func(t *testing.T) {
		// Create initial user
		user, err := domain.NewUser("test-error@example.com", "Password123!")
		require.NoError(t, err)
		err = userStore.Create(ctx, user)
		require.NoError(t, err)

		// Test duplicate email error
		duplicateUser, err := domain.NewUser("test-error@example.com", "Password123!")
		require.NoError(t, err)
		err = userStore.Create(ctx, duplicateUser)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrEmailExists)
		AssertNoErrorLeakage(t, err)

		// Test invalid user with empty email (won't pass domain validation)
		invalidUser := &domain.User{
			ID:        uuid.New(),
			Email:     "", // Invalid empty email
			Password:  "Password123!",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		err = userStore.Create(ctx, invalidUser)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrInvalidEntity)
		AssertNoErrorLeakage(t, err)
	})

	// Tests for GetByID operation
	t.Run("GetByID errors do not leak details", func(t *testing.T) {
		// Test not found error
		nonExistentID := uuid.New()
		_, err := userStore.GetByID(ctx, nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrUserNotFound)
		AssertNoErrorLeakage(t, err)
	})

	// Tests for GetByEmail operation
	t.Run("GetByEmail errors do not leak details", func(t *testing.T) {
		// Test not found error
		_, err := userStore.GetByEmail(ctx, "nonexistent-email@example.com")
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrUserNotFound)
		AssertNoErrorLeakage(t, err)
	})

	// Tests for Update operation
	t.Run("Update errors do not leak details", func(t *testing.T) {
		// Create a user to update
		user, err := domain.NewUser("update-test@example.com", "Password123!")
		require.NoError(t, err)
		err = userStore.Create(ctx, user)
		require.NoError(t, err)

		// Create another user with a different email
		otherUser, err := domain.NewUser("other-email@example.com", "Password123!")
		require.NoError(t, err)
		err = userStore.Create(ctx, otherUser)
		require.NoError(t, err)

		// Test non-existent user ID error
		nonExistentUser := &domain.User{
			ID:             uuid.New(),
			Email:          "non-existent@example.com",
			HashedPassword: "hashed_password",
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		err = userStore.Update(ctx, nonExistentUser)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrUserNotFound)
		AssertNoErrorLeakage(t, err)

		// Test duplicate email error
		userCopy := *user                // Create a copy to modify
		userCopy.Email = otherUser.Email // Try to update to an email that's already taken
		err = userStore.Update(ctx, &userCopy)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrEmailExists)
		AssertNoErrorLeakage(t, err)

		// Test invalid user error
		invalidUser := *user   // Create another copy
		invalidUser.Email = "" // Invalid email
		err = userStore.Update(ctx, &invalidUser)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrInvalidEntity)
		AssertNoErrorLeakage(t, err)
	})

	// Tests for Delete operation
	t.Run("Delete errors do not leak details", func(t *testing.T) {
		// Test non-existent user ID error
		nonExistentID := uuid.New()
		err := userStore.Delete(ctx, nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrUserNotFound)
		AssertNoErrorLeakage(t, err)
	})
}
