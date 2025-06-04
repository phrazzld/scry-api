//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"flag"
	"os"
	"testing"

	"github.com/phrazzld/scry-api/internal/config"
)

// TestMain tests the main function behavior with different flag combinations
func TestMain_Flags(t *testing.T) {
	// Save original os.Args and restore after tests
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name     string
		args     []string
		envVars  map[string]string
		wantExit int // Expected exit code (we'll need to refactor main to return this)
	}{
		{
			name: "help flag",
			args: []string{"cmd", "-h"},
			envVars: map[string]string{
				"TEST_DATABASE_URL": "postgres://test:test@localhost:5432/test",
			},
			wantExit: 0,
		},
		{
			name: "migrate status command",
			args: []string{"cmd", "-migrate=status"},
			envVars: map[string]string{
				"TEST_DATABASE_URL": "postgres://test:test@localhost:5432/test",
			},
			wantExit: 0,
		},
		{
			name: "validate migrations",
			args: []string{"cmd", "-validate-migrations"},
			envVars: map[string]string{
				"TEST_DATABASE_URL": "postgres://test:test@localhost:5432/test",
			},
			wantExit: 0,
		},
		{
			name: "verify migrations only",
			args: []string{"cmd", "-verify-migrations"},
			envVars: map[string]string{
				"TEST_DATABASE_URL": "postgres://test:test@localhost:5432/test",
			},
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that would actually run the server
			if len(tt.args) == 1 {
				t.Skip("Skipping server startup test")
			}

			// Set environment variables
			for k, v := range tt.envVars {
				oldVal := os.Getenv(k)
				os.Setenv(k, v)
				defer os.Setenv(k, oldVal)
			}

			// Set command line arguments
			os.Args = tt.args

			// TODO: We need to refactor main() to be testable
			// Currently it calls os.Exit() which terminates the test process
			// For now, we'll test the individual components instead
		})
	}
}

// TestMainComponents tests the individual components that main() uses
func TestMainComponents(t *testing.T) {
	t.Run("loadAppConfig", func(t *testing.T) {
		// Set all required environment variables for valid config
		originalEnvs := map[string]string{
			"SCRY_DATABASE_URL":             os.Getenv("SCRY_DATABASE_URL"),
			"SCRY_AUTH_JWT_SECRET":          os.Getenv("SCRY_AUTH_JWT_SECRET"),
			"SCRY_LLM_GEMINI_API_KEY":       os.Getenv("SCRY_LLM_GEMINI_API_KEY"),
			"SCRY_LLM_PROMPT_TEMPLATE_PATH": os.Getenv("SCRY_LLM_PROMPT_TEMPLATE_PATH"),
		}
		defer func() {
			for key, value := range originalEnvs {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}
		}()

		os.Setenv("SCRY_DATABASE_URL", "postgres://test:test@localhost:5432/test")
		os.Setenv("SCRY_AUTH_JWT_SECRET", "test-secret-key-for-testing-only-32-chars-long")
		os.Setenv("SCRY_LLM_GEMINI_API_KEY", "test-api-key")
		os.Setenv("SCRY_LLM_PROMPT_TEMPLATE_PATH", "../../prompts/flashcard_template.txt")
		// Set additional config values to satisfy validation
		os.Setenv("SCRY_SERVER_PORT", "8080")
		os.Setenv("SCRY_SERVER_LOG_LEVEL", "info")
		os.Setenv("SCRY_AUTH_TOKEN_LIFETIME_MINUTES", "60")
		os.Setenv("SCRY_AUTH_REFRESH_TOKEN_LIFETIME_MINUTES", "1440")

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error = %v", err)
		}

		if cfg == nil {
			t.Fatal("loadAppConfig() returned nil config")
		}

		if cfg.Database.URL == "" {
			t.Error("Config missing database URL")
		}
	})

	t.Run("setupAppLogger", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				LogLevel: "info",
			},
		}

		logger, err := setupAppLogger(cfg)
		if err != nil {
			t.Fatalf("setupAppLogger() error = %v", err)
		}

		if logger == nil {
			t.Fatal("setupAppLogger() returned nil logger")
		}
	})
}

// TestParseFlags verifies command-line flag parsing behavior
func TestParseFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantMigrate    string
		wantName       string
		wantVerbose    bool
		wantVerifyOnly bool
		wantValidate   bool
	}{
		{
			name:        "migrate up",
			args:        []string{"-migrate=up"},
			wantMigrate: "up",
		},
		{
			name:        "migrate down",
			args:        []string{"-migrate=down"},
			wantMigrate: "down",
		},
		{
			name:        "create migration with name",
			args:        []string{"-migrate=create", "-name=add_users_table"},
			wantMigrate: "create",
			wantName:    "add_users_table",
		},
		{
			name:        "verbose migration",
			args:        []string{"-migrate=up", "-verbose"},
			wantMigrate: "up",
			wantVerbose: true,
		},
		{
			name:           "verify migrations only",
			args:           []string{"-verify-migrations"},
			wantVerifyOnly: true,
		},
		{
			name:         "validate migrations",
			args:         []string{"-validate-migrations"},
			wantValidate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag.CommandLine to its default state
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			// Define flags as in main()
			migrateCmd := flag.String("migrate", "", "Run database migrations")
			migrationName := flag.String("name", "", "Name for the new migration")
			verbose := flag.Bool("verbose", false, "Enable verbose logging")
			verifyOnly := flag.Bool("verify-migrations", false, "Only verify migrations")
			validateMigrations := flag.Bool("validate-migrations", false, "Validate applied migrations")

			// Parse the test arguments
			err := flag.CommandLine.Parse(tt.args)
			if err != nil {
				t.Fatalf("flag.Parse() error = %v", err)
			}

			// Verify parsed values
			if *migrateCmd != tt.wantMigrate {
				t.Errorf("migrate flag = %v, want %v", *migrateCmd, tt.wantMigrate)
			}
			if *migrationName != tt.wantName {
				t.Errorf("name flag = %v, want %v", *migrationName, tt.wantName)
			}
			if *verbose != tt.wantVerbose {
				t.Errorf("verbose flag = %v, want %v", *verbose, tt.wantVerbose)
			}
			if *verifyOnly != tt.wantVerifyOnly {
				t.Errorf("verify-migrations flag = %v, want %v", *verifyOnly, tt.wantVerifyOnly)
			}
			if *validateMigrations != tt.wantValidate {
				t.Errorf("validate-migrations flag = %v, want %v", *validateMigrations, tt.wantValidate)
			}
		})
	}
}
