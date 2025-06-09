//go:build !compatibility && ignore_redeclarations

// Package testutils provides testing utilities for the Scry API.
//
// This package contains helpers for:
// 1. Creating test domain entities (Card, Memo, UserCardStats)
// 2. Setting up test servers for API testing
// 3. Executing API requests
// 4. Asserting API responses
// 5. Database operations for tests
//
// # Test Domain Entities
//
// For creating domain entities, use the following patterns:
//
//	// Create a card with default values:
//	card := testutils.MustCreateCardForTest(t)
//
//	// Create a card with specific options:
//	card := testutils.MustCreateCardForTest(t,
//	    testutils.WithCardUserID(userID),
//	    testutils.WithCardMemoID(memoID),
//	    testutils.WithCardContent(map[string]interface{}{
//	        "front": "Question?",
//	        "back":  "Answer",
//	    }),
//	)
//
//	// Create stats with default values:
//	stats := testutils.MustCreateStatsForTest(t)
//
//	// Create stats with specific options:
//	stats := testutils.MustCreateStatsForTest(t,
//	    testutils.WithStatsUserID(userID),
//	    testutils.WithStatsCardID(cardID),
//	    testutils.WithStatsInterval(2),
//	)
//
//	// Create a memo with default values:
//	memo := testutils.MustCreateMemoForTest(t)
//
//	// Create a memo with specific options:
//	memo := testutils.MustCreateMemoForTest(t,
//	    testutils.WithMemoUserID(userID),
//	    testutils.WithMemoText("Custom content"),
//	)
//
// # Test Servers
//
// For API testing, use the SetupCardReviewTestServer and its convenience constructors:
//
//	// Create a test server that returns a specific card:
//	server := testutils.SetupCardReviewTestServerWithNextCard(t, userID, card)
//
//	// Create a test server that returns a specific error:
//	server := testutils.SetupCardReviewTestServerWithError(t, userID, myError)
//
//	// Create a test server that returns specific stats:
//	server := testutils.SetupCardReviewTestServerWithUpdatedStats(t, userID, stats)
//
// # Request Execution
//
// For executing requests against test servers:
//
//	// GET request:
//	resp, err := testutils.ExecuteGetNextCardRequest(t, server, userID)
//
//	// POST request:
//	resp, err := testutils.ExecuteSubmitAnswerRequest(t, server, userID, cardID, domain.ReviewOutcomeGood)
//
// Response Assertions
//
//	// Assert success response:
//	testutils.AssertCardResponse(t, resp, expectedCard)
//
//	// Assert error response:
//	testutils.AssertErrorResponse(t, resp, http.StatusBadRequest, "Invalid request")
//
// # Database Operations
//
// For database operations in tests:
//
//	// Insert a memo:
//	memo := testutils.MustInsertMemo(ctx, t, tx, userID)
//
//	// Insert a card:
//	card := testutils.MustInsertCard(ctx, t, tx, userID, memoID)
//
//	// Insert stats:
//	stats := testutils.MustInsertUserCardStats(ctx, t, tx, userID, cardID)
//
//	// Count database records:
//	count := testutils.CountCards(ctx, t, tx, "user_id = $1", userID)
package testutils
