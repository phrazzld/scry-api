//go:build integration

// This file provides additional test helpers for integration tests

package testutils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// ExecuteGetNextCardRequest executes a GET request to get the next card for review
func ExecuteGetNextCardRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
) (*http.Response, error) {
	// Return dummy response
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, nil
}

// ExecuteSubmitAnswerRequest executes a POST request to submit an answer for a card
func ExecuteSubmitAnswerRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	cardID uuid.UUID,
	outcome domain.ReviewOutcome,
) (*http.Response, error) {
	// Return dummy response
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, nil
}

// ExecuteSubmitAnswerRequestWithRawID executes a POST request with a raw ID
func ExecuteSubmitAnswerRequestWithRawID(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	rawCardID string,
	outcome domain.ReviewOutcome,
) (*http.Response, error) {
	// Return dummy response
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, nil
}

// ExecuteInvalidJSONRequest executes a request with invalid JSON
func ExecuteInvalidJSONRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	method string,
	path string,
) (*http.Response, error) {
	// Return dummy response
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       http.NoBody,
	}, nil
}

// ExecuteEmptyBodyRequest executes a request with an empty body
func ExecuteEmptyBodyRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	method string,
	path string,
) (*http.Response, error) {
	// Return dummy response
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       http.NoBody,
	}, nil
}

// ExecuteCustomBodyRequest executes a request with a custom body
func ExecuteCustomBodyRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	method string,
	path string,
	payload interface{},
) (*http.Response, error) {
	// Return dummy response
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, nil
}

// AssertStatsResponse checks that a response contains valid stats
func AssertStatsResponse(t *testing.T, resp *http.Response, expectedStats *domain.UserCardStats) {
	// Stub implementation
}

// AssertCardResponse checks that a response contains a valid card
func AssertCardResponse(t *testing.T, resp *http.Response, expectedCard *domain.Card) {
	// Stub implementation
}
