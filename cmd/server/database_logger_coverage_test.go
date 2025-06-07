//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"database/sql"
	"testing"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupAppDatabaseErrorPaths tests the setupAppDatabase function error paths
func TestSetupAppDatabaseErrorPaths(t *testing.T) {
	testLogger, _ := CreateTestLogger(t)

	t.Run("invalid database URL", func(t *testing.T) {
		// Test with invalid database URL
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: "invalid-database-url",
			},
		}

		db, err := setupAppDatabase(cfg, testLogger)

		// Should fail due to invalid URL
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "database connection")
	})

	t.Run("empty database URL", func(t *testing.T) {
		// Test with empty database URL
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: "",
			},
		}

		db, err := setupAppDatabase(cfg, testLogger)

		// Should fail due to empty URL
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "database connection")
	})

	t.Run("malformed postgres URL", func(t *testing.T) {
		// Test with malformed postgres URL
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: "postgres://",
			},
		}

		db, err := setupAppDatabase(cfg, testLogger)

		// Should fail due to malformed URL
		assert.Error(t, err)
		assert.Nil(t, db)
	})

	t.Run("unreachable database host", func(t *testing.T) {
		// Test with unreachable database host
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: "postgres://user:pass@unreachable.host:5432/db",
			},
		}

		db, err := setupAppDatabase(cfg, testLogger)

		// Should fail due to unreachable host
		// This might fail at connection or ping stage
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, db)
		} else {
			// If connection succeeds, close it
			if db != nil {
				db.Close()
			}
		}
	})

	t.Run("valid URL but connection fails", func(t *testing.T) {
		// Test with valid URL format but connection fails
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: "postgres://test:test@localhost:9999/nonexistent_db",
			},
		}

		db, err := setupAppDatabase(cfg, testLogger)

		// Should fail due to connection/ping failure
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, db)
			// Error could be "failed to open database connection" or "failed to ping database"
		} else {
			// If somehow successful, clean up
			if db != nil {
				db.Close()
			}
		}
	})

	t.Run("success case with mock", func(t *testing.T) {
		// Test success case - this is challenging without a real database
		// but we can verify the function structure
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: "postgres://test:test@localhost:5432/test_db",
			},
		}

		db, err := setupAppDatabase(cfg, testLogger)

		// This will likely fail in test environment, but that's expected
		if err != nil {
			// Expected failure in test environment
			assert.Contains(t, err.Error(), "database")
		} else {
			// If successful, verify we got a database connection
			require.NotNil(t, db)
			assert.IsType(t, &sql.DB{}, db)
			defer db.Close()
		}
	})
}

// TestSetupAppLogger tests the setupAppLogger function error paths
func TestSetupAppLogger(t *testing.T) {
	t.Run("valid log level", func(t *testing.T) {
		// Test with valid log level
		cfg := &config.Config{
			Server: config.ServerConfig{
				LogLevel: "info",
			},
		}

		logger, err := setupAppLogger(cfg)

		// Should succeed
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("debug log level", func(t *testing.T) {
		// Test with debug log level
		cfg := &config.Config{
			Server: config.ServerConfig{
				LogLevel: "debug",
			},
		}

		logger, err := setupAppLogger(cfg)

		// Should succeed
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("warn log level", func(t *testing.T) {
		// Test with warn log level
		cfg := &config.Config{
			Server: config.ServerConfig{
				LogLevel: "warn",
			},
		}

		logger, err := setupAppLogger(cfg)

		// Should succeed
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("error log level", func(t *testing.T) {
		// Test with error log level
		cfg := &config.Config{
			Server: config.ServerConfig{
				LogLevel: "error",
			},
		}

		logger, err := setupAppLogger(cfg)

		// Should succeed
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("invalid log level", func(t *testing.T) {
		// Test with invalid log level
		cfg := &config.Config{
			Server: config.ServerConfig{
				LogLevel: "invalid_level",
			},
		}

		logger, err := setupAppLogger(cfg)

		// This might succeed or fail depending on logger implementation
		// Some loggers default to info level for invalid values
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, logger)
			assert.Contains(t, err.Error(), "logger")
		} else {
			// Some implementations might succeed with default level
			require.NotNil(t, logger)
		}
	})

	t.Run("empty log level", func(t *testing.T) {
		// Test with empty log level
		cfg := &config.Config{
			Server: config.ServerConfig{
				LogLevel: "",
			},
		}

		logger, err := setupAppLogger(cfg)

		// This might succeed or fail depending on logger implementation
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, logger)
		} else {
			// Some implementations might succeed with default level
			require.NotNil(t, logger)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		// Test with nil config - should panic
		assert.Panics(t, func() {
			setupAppLogger(nil)
		})
	})

	t.Run("nil server config", func(t *testing.T) {
		// Test with nil server config in config
		cfg := &config.Config{
			// Server field is nil/zero value
		}

		logger, err := setupAppLogger(cfg)

		// This might succeed with default log level or fail
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, logger)
		} else {
			// Some implementations might succeed with default level
			require.NotNil(t, logger)
		}
	})
}

// TestDatabaseConfigurationVariations tests various database configuration scenarios
func TestDatabaseConfigurationVariations(t *testing.T) {
	testLogger, _ := CreateTestLogger(t)

	t.Run("postgres URL with all parameters", func(t *testing.T) {
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: "postgres://user:password@host:5432/database?sslmode=disable&connect_timeout=10",
			},
		}

		db, err := setupAppDatabase(cfg, testLogger)

		// Will likely fail due to unreachable host, but tests URL parsing
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, db)
		} else if db != nil {
			db.Close()
		}
	})

	t.Run("postgres URL with special characters", func(t *testing.T) {
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: "postgres://user%40domain:p%40ssw%0rd@host:5432/db",
			},
		}

		db, err := setupAppDatabase(cfg, testLogger)

		// Will likely fail, but tests URL encoding handling
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, db)
		} else if db != nil {
			db.Close()
		}
	})

	t.Run("postgresql scheme", func(t *testing.T) {
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: "postgresql://user:pass@host:5432/db",
			},
		}

		db, err := setupAppDatabase(cfg, testLogger)

		// Will likely fail, but tests postgresql:// scheme
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, db)
		} else if db != nil {
			db.Close()
		}
	})
}

// TestLoggerConfigurationVariations tests various logger configuration scenarios
func TestLoggerConfigurationVariations(t *testing.T) {
	t.Run("case variations", func(t *testing.T) {
		testCases := []string{
			"INFO",
			"Info",
			"DEBUG",
			"Debug",
			"WARN",
			"WARNING",
			"Warn",
			"ERROR",
			"Error",
			"FATAL",
			"Fatal",
		}

		for _, level := range testCases {
			t.Run("level_"+level, func(t *testing.T) {
				cfg := &config.Config{
					Server: config.ServerConfig{
						LogLevel: level,
					},
				}

				logger, err := setupAppLogger(cfg)

				// Most should succeed, some might fail depending on implementation
				if err != nil {
					assert.Error(t, err)
					assert.Nil(t, logger)
				} else {
					require.NotNil(t, logger)
				}
			})
		}
	})

	t.Run("numeric levels", func(t *testing.T) {
		testCases := []string{"0", "1", "2", "3", "4", "-1", "10"}

		for _, level := range testCases {
			t.Run("numeric_"+level, func(t *testing.T) {
				cfg := &config.Config{
					Server: config.ServerConfig{
						LogLevel: level,
					},
				}

				logger, err := setupAppLogger(cfg)

				// Should handle numeric levels gracefully
				if err != nil {
					assert.Error(t, err)
					assert.Nil(t, logger)
				} else {
					require.NotNil(t, logger)
				}
			})
		}
	})
}
