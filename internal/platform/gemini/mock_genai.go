//go:build test_without_external_deps

package gemini

import (
	"context"
	"fmt"
)

// MockGenAIClient provides a simple mock implementation for testing
type MockGenAIClient struct {
	ShouldFail    bool
	ResponseCards []CardSchema
	ErrorToReturn error // Specific error to return when ShouldFail is true
}

// NewMockGenAIClient creates a new instance of MockGenAIClient with default values
func NewMockGenAIClient() *MockGenAIClient {
	return &MockGenAIClient{
		ShouldFail: false,
		ResponseCards: []CardSchema{
			{
				Front: "Test Front",
				Back:  "Test Back",
				Hint:  "Test Hint",
				Tags:  []string{"test", "mock"},
			},
		},
	}
}

// GenerativeModel returns a string for the model name (not used in mocks)
func (m *MockGenAIClient) GenerativeModel(name string) string {
	return name
}

// Close is a no-op for the mock
func (m *MockGenAIClient) Close() error {
	return nil
}

// MockGenerateContent simulates the API call without using real genai package types
func (m *MockGenAIClient) MockGenerateContent(
	ctx context.Context,
	prompt string,
) (*ResponseSchema, error) {
	if m.ShouldFail {
		// Return the specific error if configured, otherwise a generic error
		if m.ErrorToReturn != nil {
			return nil, m.ErrorToReturn
		}
		return nil, fmt.Errorf("mock API error")
	}

	// Check for context cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Return the configured cards
	return &ResponseSchema{
		Cards: m.ResponseCards,
	}, nil
}

// SetResponseCards allows configuring the mock response
func (m *MockGenAIClient) SetResponseCards(cards []CardSchema) {
	m.ResponseCards = cards
}

// SetShouldFail configures the mock to fail
func (m *MockGenAIClient) SetShouldFail(shouldFail bool) {
	m.ShouldFail = shouldFail
}

// SetErrorToReturn configures the specific error that should be returned when ShouldFail is true
func (m *MockGenAIClient) SetErrorToReturn(err error) {
	m.ErrorToReturn = err
	// If we're setting an error, we also want to fail
	if err != nil {
		m.ShouldFail = true
	}
}
