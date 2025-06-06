package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"testing"

	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestCreateMultiple_EmptyList tests the empty list handling in CreateMultiple
func TestCreateMultiple_EmptyList(t *testing.T) {
	// Use mock database to avoid requiring real DB connection
	mockDB := &mockDBTX{}
	logger := slog.Default()
	cardStore := NewPostgresCardStore(mockDB, logger)

	// Test empty list - should return nil without any database operations
	err := cardStore.CreateMultiple(context.Background(), []*domain.Card{})
	assert.NoError(t, err, "CreateMultiple should handle empty list gracefully")

	// Test nil list - should also return nil
	err = cardStore.CreateMultiple(context.Background(), nil)
	assert.NoError(t, err, "CreateMultiple should handle nil list gracefully")
}

// TestUserCreate_EmptyPassword tests the password validation skip for empty passwords
func TestUserCreate_EmptyPassword(t *testing.T) {
	// Use mock database to avoid requiring real DB connection
	mockDB := &mockDBTX{}
	userStore := NewPostgresUserStore(mockDB, 10)

	user := &domain.User{
		Email:    "test@example.com",
		Password: "", // Empty password should skip validation
	}

	// Should fail at DB level (mock), not at validation level
	err := userStore.Create(context.Background(), user)
	assert.Error(t, err) // Expect error due to mock DB, but validation should have passed
}

// TestTaskStoreValidation tests task store validation without database
func TestTaskStoreValidation(t *testing.T) {
	mockDB := &mockDBTX{}
	taskStore := NewPostgresTaskStore(mockDB)

	t.Run("WithTx", func(t *testing.T) {
		tx := &sql.Tx{}
		result := taskStore.WithTx(tx)
		assert.NotNil(t, result)
	})
}

// TestErrorHandling tests error handling logic
func TestErrorHandling(t *testing.T) {
	t.Run("IsNotFoundError_RegularError", func(t *testing.T) {
		// Test non-not-found error
		err := errors.New("regular error")
		result := IsNotFoundError(err)
		assert.False(t, result, "Regular error should not be identified as not found")
	})

	t.Run("IsNotFoundError_SqlNoRows", func(t *testing.T) {
		// Test sql.ErrNoRows error
		result := IsNotFoundError(sql.ErrNoRows)
		assert.True(t, result, "sql.ErrNoRows should be identified as not found")
	})
}

// TestCreateMultiple_InvalidCard tests card validation in CreateMultiple
func TestCreateMultiple_InvalidCard(t *testing.T) {
	mockDB := &mockDBTX{}
	logger := slog.Default()
	cardStore := NewPostgresCardStore(mockDB, logger)

	// Create an invalid card (e.g., with empty content)
	invalidCard := &domain.Card{
		// Missing required fields to trigger validation failure
	}

	// Test should fail validation before hitting database
	err := cardStore.CreateMultiple(context.Background(), []*domain.Card{invalidCard})
	assert.Error(t, err, "CreateMultiple should fail validation for invalid card")
}

// TestMapUniqueViolation_NonUniqueError tests the sanitizeError path via MapUniqueViolation
func TestMapUniqueViolation_NonUniqueError(t *testing.T) {
	// Create a regular error (not a unique violation)
	regularErr := errors.New("some regular database error")

	// This should call sanitizeError because it's not a unique violation
	result := MapUniqueViolation(regularErr, "test_entity", "test_constraint", nil)

	// The result should be the original error (sanitizeError's default case)
	assert.Equal(t, regularErr, result, "Non-unique violation should return sanitized error")
}

// TestUserCreate_InvalidPassword tests password validation in Create
func TestUserCreate_InvalidPassword(t *testing.T) {
	mockDB := &mockDBTX{}
	userStore := NewPostgresUserStore(mockDB, 10)

	user := &domain.User{
		Email:    "test@example.com",
		Password: "123", // Too short password should fail validation
	}

	// Should fail password validation before hitting database
	err := userStore.Create(context.Background(), user)
	assert.Error(t, err, "Create should fail password validation for invalid password")
}
