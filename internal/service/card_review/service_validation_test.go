//go:build test_without_external_deps

package card_review

import (
	"testing"

	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestIsValidOutcome(t *testing.T) {
	tests := []struct {
		name     string
		outcome  domain.ReviewOutcome
		expected bool
	}{
		{
			name:     "valid_outcome_again",
			outcome:  domain.ReviewOutcomeAgain,
			expected: true,
		},
		{
			name:     "valid_outcome_hard",
			outcome:  domain.ReviewOutcomeHard,
			expected: true,
		},
		{
			name:     "valid_outcome_good",
			outcome:  domain.ReviewOutcomeGood,
			expected: true,
		},
		{
			name:     "valid_outcome_easy",
			outcome:  domain.ReviewOutcomeEasy,
			expected: true,
		},
		{
			name:     "invalid_outcome_empty_string",
			outcome:  "",
			expected: false,
		},
		{
			name:     "invalid_outcome_random_string",
			outcome:  "invalid",
			expected: false,
		},
		{
			name:     "invalid_outcome_numeric",
			outcome:  "123",
			expected: false,
		},
		{
			name:     "invalid_outcome_mixed_case",
			outcome:  "Good",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidOutcome(tt.outcome)
			assert.Equal(t, tt.expected, result)
		})
	}
}
