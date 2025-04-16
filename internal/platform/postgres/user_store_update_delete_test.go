package postgres_test

// This file contains tests for the Update and Delete methods of PostgresUserStore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestPostgresUserStore_Update tests the Update method
func TestPostgresUserStore_Update(t *testing.T) {
	t.Parallel() // Enable parallel testing

	testutils.WithTx(t, testDB, func(tx store.DBTX) {
		// Create a new user store
		userStore := postgres.NewPostgresUserStore(tx, bcrypt.DefaultCost)

		// Test Case 1: Successful update of user details
		t.Run("Successful update", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Insert a test user
			originalEmail := fmt.Sprintf("update-orig-%s@example.com", uuid.New().String()[:8])
			userID := insertTestUser(ctx, t, tx, originalEmail)

			// Get the user from the database to start with a valid db state
			user, err := userStore.GetByID(ctx, userID)
			require.NoError(t, err, "User retrieval should succeed")

			// Update the user details
			newEmail := fmt.Sprintf("update-new-%s@example.com", uuid.New().String()[:8])
			user.Email = newEmail
			newPassword := "NewPassword123!"
			user.Password = newPassword

			// Wait a moment to ensure updated timestamp will be different
			time.Sleep(10 * time.Millisecond)

			// Call the Update method
			err = userStore.Update(ctx, user)

			// Verify the result
			require.NoError(t, err, "User update should succeed")

			// Retrieve the updated user
			updatedUser, err := userStore.GetByID(ctx, userID)
			require.NoError(t, err, "User retrieval should succeed after update")

			// Verify the updated fields
			assert.Equal(t, newEmail, updatedUser.Email, "User email should be updated")
			assert.NotEqual(t, user.HashedPassword, updatedUser.HashedPassword, "Password hash should be different")
			assert.True(
				t,
				updatedUser.UpdatedAt.After(user.CreatedAt),
				"UpdatedAt should be later than CreatedAt",
			)
		})

		// Test Case 2: Update with empty password (should retain existing hash)
		t.Run("Update without changing password", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Insert a test user
			originalEmail := fmt.Sprintf("update-nopass-%s@example.com", uuid.New().String()[:8])
			userID := insertTestUser(ctx, t, tx, originalEmail)

			// Get the user from the database
			user, err := userStore.GetByID(ctx, userID)
			require.NoError(t, err, "User retrieval should succeed")
			originalHash := user.HashedPassword

			// Update just the email, not the password
			newEmail := fmt.Sprintf("update-nopass-new-%s@example.com", uuid.New().String()[:8])
			user.Email = newEmail
			user.Password = "" // No new password

			// Call the Update method
			err = userStore.Update(ctx, user)

			// Verify the result
			require.NoError(t, err, "User update should succeed")

			// Retrieve the updated user
			updatedUser, err := userStore.GetByID(ctx, userID)
			require.NoError(t, err, "User retrieval should succeed after update")

			// Verify the updated fields
			assert.Equal(t, newEmail, updatedUser.Email, "User email should be updated")
			assert.Equal(t, originalHash, updatedUser.HashedPassword, "Password hash should not change")
		})

		// Test Case 3: Update with non-existent user
		t.Run("Update non-existent user", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a user with a non-existent ID
			nonExistentUser := &domain.User{
				ID:        uuid.New(),
				Email:     fmt.Sprintf("non-existent-%s@example.com", uuid.New().String()[:8]),
				Password:  "Password123!",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}

			// Call the Update method
			err := userStore.Update(ctx, nonExistentUser)

			// Verify the result
			assert.ErrorIs(t, err, store.ErrUserNotFound, "Should return ErrUserNotFound")
		})

		// Test Case 4: Update with invalid email
		t.Run("Update with invalid email", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Insert a test user
			originalEmail := fmt.Sprintf("update-invalid-%s@example.com", uuid.New().String()[:8])
			userID := insertTestUser(ctx, t, tx, originalEmail)

			// Get the user from the database
			user, err := userStore.GetByID(ctx, userID)
			require.NoError(t, err, "User retrieval should succeed")

			// Update with invalid email
			user.Email = "not-an-email"

			// Call the Update method
			err = userStore.Update(ctx, user)

			// Verify the result
			assert.Error(t, err, "Update with invalid email should fail")
			assert.Equal(t, domain.ErrInvalidEmail, err, "Error should be ErrInvalidEmail")

			// Verify the user wasn't updated
			updatedUser, err := userStore.GetByID(ctx, userID)
			require.NoError(t, err, "User retrieval should succeed")
			assert.Equal(t, originalEmail, updatedUser.Email, "Email should not have been updated")
		})

		// Test Case 5: Update to an email that already exists
		t.Run("Update to existing email", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Insert two test users
			existingEmail := fmt.Sprintf("existing-%s@example.com", uuid.New().String()[:8])
			_ = insertTestUser(ctx, t, tx, existingEmail)

			originalEmail := fmt.Sprintf("update-dup-%s@example.com", uuid.New().String()[:8])
			userID := insertTestUser(ctx, t, tx, originalEmail)

			// Get the second user from the database
			user, err := userStore.GetByID(ctx, userID)
			require.NoError(t, err, "User retrieval should succeed")

			// Try to update to the first user's email
			user.Email = existingEmail

			// Call the Update method
			err = userStore.Update(ctx, user)

			// Verify the result
			assert.ErrorIs(t, err, store.ErrEmailExists, "Should return ErrEmailExists")

			// Verify the user wasn't updated
			updatedUser, err := userStore.GetByID(ctx, userID)
			require.NoError(t, err, "User retrieval should succeed")
			assert.Equal(t, originalEmail, updatedUser.Email, "Email should not have been updated")
		})
	})
}

// TestPostgresUserStore_Delete tests the Delete method
func TestPostgresUserStore_Delete(t *testing.T) {
	t.Parallel() // Enable parallel testing

	testutils.WithTx(t, testDB, func(tx store.DBTX) {
		// Create a new user store
		userStore := postgres.NewPostgresUserStore(tx, bcrypt.DefaultCost)

		// Test Case 1: Successful deletion of an existing user
		t.Run("Successful deletion", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Insert a test user
			email := fmt.Sprintf("delete-%s@example.com", uuid.New().String()[:8])
			userID := insertTestUser(ctx, t, tx, email)

			// Verify the user exists
			count := countUsers(ctx, t, tx, "id = $1", userID)
			assert.Equal(t, 1, count, "User should exist before deletion")

			// Call the Delete method
			err := userStore.Delete(ctx, userID)

			// Verify the result
			require.NoError(t, err, "User deletion should succeed")

			// Verify the user no longer exists
			count = countUsers(ctx, t, tx, "id = $1", userID)
			assert.Equal(t, 0, count, "User should not exist after deletion")

			// Attempting to retrieve the deleted user should return not found
			_, err = userStore.GetByID(ctx, userID)
			assert.ErrorIs(t, err, store.ErrUserNotFound, "GetByID should return ErrUserNotFound after deletion")
		})

		// Test Case 2: Attempt to delete a non-existent user
		t.Run("Delete non-existent user", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Generate a random user ID that doesn't exist
			nonExistentID := uuid.New()

			// Call the Delete method
			err := userStore.Delete(ctx, nonExistentID)

			// Verify the result
			assert.ErrorIs(t, err, store.ErrUserNotFound, "Should return ErrUserNotFound")
		})
	})
}
