//go:build test_without_external_deps

package testutils

import (
	"testing"
	"time"
)

func TestDatabaseHealthCheckConfig(t *testing.T) {
	t.Parallel()

	t.Run("default_config", func(t *testing.T) {
		config := DefaultHealthCheckConfig()

		if config.MaxRetries != 3 {
			t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
		}

		if config.InitialRetryDelay != 500*time.Millisecond {
			t.Errorf("Expected InitialRetryDelay to be 500ms, got %v", config.InitialRetryDelay)
		}

		if !config.EnableAutoHealthCheck {
			t.Error("Expected EnableAutoHealthCheck to be true")
		}
	})

	t.Run("ci_config", func(t *testing.T) {
		config := CIHealthCheckConfig()

		if config.MaxRetries != 5 {
			t.Errorf("Expected MaxRetries to be 5, got %d", config.MaxRetries)
		}

		if config.InitialRetryDelay != 1*time.Second {
			t.Errorf("Expected InitialRetryDelay to be 1s, got %v", config.InitialRetryDelay)
		}

		if config.ConnectionTimeout != 15*time.Second {
			t.Errorf("Expected ConnectionTimeout to be 15s, got %v", config.ConnectionTimeout)
		}

		if !config.EnableAutoHealthCheck {
			t.Error("Expected EnableAutoHealthCheck to be true")
		}
	})

	t.Run("disabled_config", func(t *testing.T) {
		config := DisabledHealthCheckConfig()

		if config.EnableAutoHealthCheck {
			t.Error("Expected EnableAutoHealthCheck to be false")
		}
	})
}

func TestMaskDBURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty_url",
			input:    "",
			expected: "",
		},
		{
			name:     "postgres_url",
			input:    "postgres://user:password@localhost:5432/db",
			expected: "postgres://user:****@localhost:5432/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskDBURL(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
