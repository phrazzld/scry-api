//go:build !test_without_external_deps
// +build !test_without_external_deps

package gemini

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/generation"
)

// validateConfig performs enhanced configuration validation for production environments.
// It validates that API keys and other required settings are properly set.
//
// Parameters:
//   - ctx: Context for logging and cancellation
//   - logger: Logger for recording validation results
//   - config: The LLM configuration to validate
//
// Returns:
//   - An error if validation fails, nil otherwise
func validateConfig(ctx context.Context, logger *slog.Logger, config config.LLMConfig) error {
	logger.InfoContext(ctx, "Validating production LLM configuration")

	// Validate API key
	if config.GeminiAPIKey == "" {
		logger.ErrorContext(ctx, "Missing API key in production environment",
			"error", "GeminiAPIKey is empty")
		return fmt.Errorf("%w: GeminiAPIKey cannot be empty in production environment", generation.ErrInvalidConfig)
	}

	// Validate model name
	if config.ModelName == "" {
		logger.ErrorContext(ctx, "Missing model name in production environment",
			"error", "ModelName is empty")
		return fmt.Errorf("%w: ModelName cannot be empty in production environment", generation.ErrInvalidConfig)
	}

	// Validate retry settings (if they're set)
	if config.MaxRetries < 0 {
		logger.WarnContext(ctx, "Invalid MaxRetries value",
			"value", config.MaxRetries,
			"action", "using default value")
		// We're not returning an error here since we can fall back to defaults
	}

	if config.RetryDelaySeconds < 0 {
		logger.WarnContext(ctx, "Invalid RetryDelaySeconds value",
			"value", config.RetryDelaySeconds,
			"action", "using default value")
		// We're not returning an error here since we can fall back to defaults
	}

	logger.InfoContext(ctx, "Production configuration validation passed")
	return nil
}
