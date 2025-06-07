//go:build test_without_external_deps

package gemini

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/generation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerator(t *testing.T) {
	// Create a temporary template file for testing
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "test_template.txt")
	templateContent := `Generate flashcards from the following memo:
{{.MemoText}}

Create exactly 3 flashcards in JSON format.`

	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		logger      *slog.Logger
		config      config.LLMConfig
		expectError bool
		errorType   error
		errorMsg    string
	}{
		{
			name:        "nil_logger_returns_error",
			logger:      nil,
			config:      config.LLMConfig{PromptTemplatePath: templatePath},
			expectError: true,
			errorMsg:    "logger cannot be nil",
		},
		{
			name:   "empty_template_path_returns_config_error",
			logger: slog.Default(),
			config: config.LLMConfig{
				PromptTemplatePath: "",
			},
			expectError: true,
			errorType:   generation.ErrInvalidConfig,
			errorMsg:    "prompt template path cannot be empty",
		},
		{
			name:   "valid_config_returns_generator",
			logger: slog.Default(),
			config: config.LLMConfig{
				PromptTemplatePath: templatePath,
				GeminiAPIKey:       "test-api-key",
				ModelName:          "gemini-pro",
				MaxRetries:         3,
				RetryDelaySeconds:  2,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			generator, err := NewGenerator(ctx, tt.logger, tt.config)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, generator)
				assert.Contains(t, err.Error(), tt.errorMsg)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, generator)
				assert.Implements(t, (*generation.Generator)(nil), generator)
			}
		})
	}
}

func TestNewGeminiGenerator(t *testing.T) {
	// Create a temporary template file for testing
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "test_template.txt")
	templateContent := `Generate flashcards from the following memo:
{{.MemoText}}

Create exactly 3 flashcards in JSON format.`

	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		logger      *slog.Logger
		config      config.LLMConfig
		expectError bool
		errorType   error
		errorMsg    string
	}{
		{
			name:        "nil_logger_returns_error",
			logger:      nil,
			config:      config.LLMConfig{PromptTemplatePath: templatePath},
			expectError: true,
			errorMsg:    "logger cannot be nil",
		},
		{
			name:   "empty_template_path_returns_config_error",
			logger: slog.Default(),
			config: config.LLMConfig{
				PromptTemplatePath: "",
			},
			expectError: true,
			errorType:   generation.ErrInvalidConfig,
			errorMsg:    "prompt template path cannot be empty",
		},
		{
			name:   "nonexistent_template_file_returns_config_error",
			logger: slog.Default(),
			config: config.LLMConfig{
				PromptTemplatePath: "/nonexistent/path/template.txt",
			},
			expectError: true,
			errorType:   generation.ErrInvalidConfig,
			errorMsg:    "failed to read prompt template",
		},
		{
			name:   "invalid_template_syntax_returns_config_error",
			logger: slog.Default(),
			config: config.LLMConfig{
				PromptTemplatePath: func() string {
					// Create a template file with invalid syntax
					invalidPath := filepath.Join(tempDir, "invalid_template.txt")
					invalidContent := `Generate flashcards: {{.InvalidSyntax}`
					_ = os.WriteFile(invalidPath, []byte(invalidContent), 0644)
					return invalidPath
				}(),
			},
			expectError: true,
			errorType:   generation.ErrInvalidConfig,
			errorMsg:    "failed to parse prompt template",
		},
		{
			name:   "valid_config_returns_generator",
			logger: slog.Default(),
			config: config.LLMConfig{
				PromptTemplatePath: templatePath,
				GeminiAPIKey:       "test-api-key",
				ModelName:          "gemini-pro",
				MaxRetries:         3,
				RetryDelaySeconds:  2,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			generator, err := NewGeminiGenerator(ctx, tt.logger, tt.config)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, generator)
				assert.Contains(t, err.Error(), tt.errorMsg)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, generator)
				assert.NotNil(t, generator.logger)
				assert.NotNil(t, generator.promptTemplate)
				assert.Equal(t, tt.config, generator.config)
			}
		})
	}
}

func TestMockGenAIClient_Methods(t *testing.T) {
	client := NewMockGenAIClient()

	t.Run("generative_model_returns_model_name", func(t *testing.T) {
		model := client.GenerativeModel("test-model")
		assert.Equal(t, "test-model", model)
	})

	t.Run("close_returns_nil_error", func(t *testing.T) {
		err := client.Close()
		assert.NoError(t, err)
	})

	t.Run("set_response_cards", func(t *testing.T) {
		cards := []CardSchema{
			{Front: "Test Front", Back: "Test Back"},
		}
		client.SetResponseCards(cards)
		assert.Equal(t, cards, client.ResponseCards)
	})

	t.Run("set_should_fail", func(t *testing.T) {
		client.SetShouldFail(true)
		assert.True(t, client.ShouldFail)

		client.SetShouldFail(false)
		assert.False(t, client.ShouldFail)
	})
}
