//go:build exported_core_functions

package main

import (
	"fmt"
	"log/slog"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/logger"
)

// setupAppLogger configures and initializes the application logger based on config settings.
// Returns the configured logger or an error if setup fails.
func setupAppLogger(cfg *config.Config) (*slog.Logger, error) {
	loggerConfig := logger.LoggerConfig{
		Level: cfg.Server.LogLevel,
	}

	l, err := logger.Setup(loggerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to set up logger: %w", err)
	}

	return l, nil
}
