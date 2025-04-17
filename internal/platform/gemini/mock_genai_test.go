//go:build test_without_external_deps
// +build test_without_external_deps

// Package gemini_test provides a testing environment for the gemini package.
package gemini_test

// This file provides mock definitions for the genai package,
// allowing us to test our code without the actual dependencies.

// Mock genai.Client for testing
type MockGenaiClient struct{}

// Mock genai.GenerativeModel for testing
type MockGenerativeModel struct{}

// Mock option.ClientOption for testing
type MockClientOption struct{}

// NewClient mocks genai.NewClient for testing
func NewMockClient() *MockGenaiClient {
	return &MockGenaiClient{}
}

// WithAPIKey mocks option.WithAPIKey for testing
func WithMockAPIKey(apiKey string) MockClientOption {
	return MockClientOption{}
}

// Close mocks client.Close for testing
func (c *MockGenaiClient) Close() error {
	return nil
}

// GenerativeModel mocks client.GenerativeModel for testing
func (c *MockGenaiClient) GenerativeModel(modelName string) *MockGenerativeModel {
	return &MockGenerativeModel{}
}
