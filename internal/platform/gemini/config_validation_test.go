//go:build test_without_external_deps
// +build test_without_external_deps

package gemini

import (
	"context"
	"log/slog"

	"github.com/phrazzld/scry-api/internal/config"
)

// validateConfig is a minimal validation function for test environments.
// It relaxes the validation requirements when run in test mode with the test_without_external_deps tag.
// This allows tests to run with minimal configuration, without requiring real API keys.
//
// Parameters:
//   - ctx: Context for logging and cancellation
//   - logger: Logger for recording validation results
//   - config: The LLM configuration to validate
//
// Returns:
//   - Always returns nil in test mode to allow tests to run with minimal configuration
func validateConfig(ctx context.Context, logger *slog.Logger, config config.LLMConfig) error {
	logger.InfoContext(ctx, "Minimal validation for test environment - skipping API key validation")
	return nil
}
