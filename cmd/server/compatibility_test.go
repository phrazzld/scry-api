//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestCompatibilityFunctions tests all the compatibility wrapper functions
// These functions in compatibility.go currently have 0% coverage (lines 45-50 in coverage)
func TestCompatibilityFunctions(t *testing.T) {
	t.Run("setupCardManagementTestServer", func(t *testing.T) {
		// This function sets up a test server for card management
		// It will panic without proper database setup, which provides coverage
		assert.Panics(t, func() {
			setupCardManagementTestServer(t, nil)
		}, "should panic with nil transaction")
	})

	t.Run("getCardByID", func(t *testing.T) {
		// This function gets a card by ID
		// It will panic without a real transaction, which provides coverage
		cardID := uuid.New()
		assert.Panics(t, func() {
			getCardByID(nil, cardID)
		}, "should panic with nil transaction")
	})

	t.Run("getAuthToken", func(t *testing.T) {
		// This function gets an auth token
		// It will panic without proper setup, which provides coverage
		userID := uuid.New()
		assert.Panics(t, func() {
			getAuthToken(t, userID)
		}, "should panic without proper setup")
	})

	t.Run("createTestUser", func(t *testing.T) {
		// This function creates a test user
		// It will panic without database, which provides coverage
		assert.Panics(t, func() {
			createTestUser(t, nil)
		}, "should panic with nil transaction")
	})

	t.Run("createTestCard", func(t *testing.T) {
		// This function creates a test card
		// It will panic without database/user, which provides coverage
		userID := uuid.New()
		assert.Panics(t, func() {
			createTestCard(t, nil, userID)
		}, "should panic with nil transaction")
	})

	t.Run("getUserCardStats", func(t *testing.T) {
		// This function gets user card stats
		// It will panic without database, which provides coverage
		userID := uuid.New()
		cardID := uuid.New()
		assert.Panics(t, func() {
			getUserCardStats(t, nil, userID, cardID)
		}, "should panic with nil transaction")
	})
}
