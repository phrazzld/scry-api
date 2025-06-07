//go:build test_without_external_deps

package testutils

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// CreateTestMemo creates a test memo with the provided user ID.
// This is a compatibility function used in tests that don't require external dependencies.
func CreateTestMemo(t *testing.T, userID uuid.UUID) *domain.Memo {
	t.Helper()

	// Create a test memo with default values
	return &domain.Memo{
		ID:        uuid.New(),
		UserID:    userID,
		Text:      "Test memo content " + uuid.New().String()[:8],
		Status:    domain.MemoStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// AssertNoErrorLeakage checks that the error does not leak internal database details.
// This is particularly important for testing error handling to ensure sensitive
// implementation details are not exposed to users.
func AssertNoErrorLeakage(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		return
	}

	errMsg := err.Error()

	// Database specific terms that should not be leaked to users
	sensitiveTerms := []string{
		// PostgreSQL specific
		"postgres", "postgresql", "pq:", "pg:", "pgx:",
		"23505", "23503", "23502", "23514", // PostgreSQL error codes
		"duplicate key", "violates unique constraint",
		"violates foreign key constraint",
		"violates not-null constraint",
		"constraint", "table", "column",

		// SQL specific
		"sql:", "sql.ErrNoRows", "database/sql",
		"query", "syntax error",

		// Internal details
		"position:", "line:", "file:", "detail:", "hint:",
		"internal query:", "where:", "schema",
	}

	for _, term := range sensitiveTerms {
		if strings.Contains(errMsg, term) {
			t.Errorf("Error message leaks internal detail: %q. Full error: %q", term, errMsg)
		}
	}

	// In a production app, also verify it doesn't leak too much technical information
	// by keeping error messages to a reasonable length
	if len(errMsg) >= 200 {
		t.Errorf("Error message is suspiciously long which may indicate leakage of internal details: %q", errMsg)
	}
}
