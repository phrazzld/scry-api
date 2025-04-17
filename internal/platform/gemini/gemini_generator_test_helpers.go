//go:build testing
// +build testing

package gemini

import (
	"context"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// Test helper methods to expose unexported methods for testing

// CreatePromptForTest provides test access to the createPrompt method
func (g *GeminiGenerator) CreatePromptForTest(ctx context.Context, memoText string) (string, error) {
	return g.createPrompt(ctx, memoText)
}

// CallGeminiWithRetryForTest provides test access to the callGeminiWithRetry method
func (g *GeminiGenerator) CallGeminiWithRetryForTest(ctx context.Context, prompt string) (*ResponseSchema, error) {
	return g.callGeminiWithRetry(ctx, prompt)
}

// ParseResponseForTest provides test access to the parseResponse method
func (g *GeminiGenerator) ParseResponseForTest(
	ctx context.Context,
	response *ResponseSchema,
	userID uuid.UUID,
	memoID uuid.UUID,
) ([]*domain.Card, error) {
	return g.parseResponse(ctx, response, userID, memoID)
}
