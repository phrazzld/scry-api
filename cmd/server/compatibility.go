//go:build disabled_compatibility

// DEPRECATED: This file is no longer used and is scheduled for removal.
// Please use the standardized helpers from internal/testutils/api/* instead.
//
// This file was previously used as a compatibility layer to ease migration,
// but it has been replaced by direct usage of the standardized helpers.

package main

import (
	"database/sql"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/testutils/api"
)

// Compatibility layer for cmd/server to smooth transition to new structure

// setupCardManagementTestServer delegates to api.SetupCardManagementTestServer for compatibility
func setupCardManagementTestServer(t *testing.T, tx *sql.Tx) *httptest.Server {
	return api.SetupCardManagementTestServer(t, tx)
}

// getCardByID delegates to api.GetCardByID for compatibility
func getCardByID(tx *sql.Tx, cardID uuid.UUID) (*domain.Card, error) {
	return api.GetCardByID(tx, cardID)
}

// getAuthToken delegates to api.GetAuthToken for compatibility
func getAuthToken(t *testing.T, userID uuid.UUID) string {
	return api.GetAuthToken(t, userID)
}

// createTestUser delegates to api.CreateTestUser for compatibility
func createTestUser(t *testing.T, tx *sql.Tx) uuid.UUID {
	return api.CreateTestUser(t, tx)
}

// createTestCard delegates to api.CreateTestCard for compatibility
func createTestCard(t *testing.T, tx *sql.Tx, userID uuid.UUID) *domain.Card {
	return api.CreateTestCard(t, tx, userID)
}

// getUserCardStats delegates to api.GetUserCardStats for compatibility
func getUserCardStats(t *testing.T, tx *sql.Tx, userID, cardID uuid.UUID) *domain.UserCardStats {
	return api.GetUserCardStats(t, tx, userID, cardID)
}
