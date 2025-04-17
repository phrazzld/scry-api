//go:build test_without_external_deps
// +build test_without_external_deps

// Package gemini_test provides a testing environment for the gemini package.
package gemini_test

// This file provides mock definitions for the generativelanguage v1beta package,
// allowing us to test our code without the actual dependencies.

import "context"

// MockGenerativelanguageService mocks generativelanguage.Service for testing
type MockGenerativelanguageService struct {
	Models *MockModelsService
}

// MockModelsService mocks generativelanguage.ModelsService for testing
type MockModelsService struct{}

// MockGenerateContentResponse mocks generativelanguage.GenerateContentResponse for testing
type MockGenerateContentResponse struct {
	Candidates []*MockCandidate
}

// MockCandidate mocks generativelanguage.Candidate for testing
type MockCandidate struct {
	Content     *MockContent
	FinishReason string
}

// MockContent mocks generativelanguage.Content for testing
type MockContent struct {
	Parts []*MockPart
}

// MockPart mocks generativelanguage.Part for testing
type MockPart struct {
	Text string
}

// MockClientOption mocks option.ClientOption for testing
type MockClientOption struct{}

// NewService mocks generativelanguage.NewService for testing
func NewMockService() *MockGenerativelanguageService {
	return &MockGenerativelanguageService{
		Models: &MockModelsService{},
	}
}

// WithAPIKey mocks option.WithAPIKey for testing
func WithMockAPIKey(apiKey string) MockClientOption {
	return MockClientOption{}
}

// GenerateContent mocks ModelService.GenerateContent
func (s *MockModelsService) GenerateContent(modelName string, req interface{}) *MockGenerateContentCall {
	return &MockGenerateContentCall{}
}

// MockGenerateContentCall mocks generativelanguage.GenerateContentCall
type MockGenerateContentCall struct{}

// Context adds a context to the call (mock implementation)
func (c *MockGenerateContentCall) Context(ctx context.Context) *MockGenerateContentCall {
	return c
}

// Do executes the call (mock implementation)
func (c *MockGenerateContentCall) Do() (*MockGenerateContentResponse, error) {
	// Return a mock response by default
	return &MockGenerateContentResponse{
		Candidates: []*MockCandidate{
			{
				Content: &MockContent{
					Parts: []*MockPart{
						{
							Text: `{"cards":[{"front":"Test Front","back":"Test Back"}]}`,
						},
					},
				},
				FinishReason: "STOP",
			},
		},
	}, nil
}