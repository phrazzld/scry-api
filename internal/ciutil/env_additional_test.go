//go:build test_without_external_deps

package ciutil_test

import (
	"os"
	"testing"

	"github.com/phrazzld/scry-api/internal/ciutil"
	"github.com/stretchr/testify/assert"
)

func TestIsGitLabCI(t *testing.T) {
	// Save original environment
	originalGitLabCI := os.Getenv(ciutil.EnvGitLabCI)
	originalGitLabProjectDir := os.Getenv(ciutil.EnvGitLabProjectDir)

	// Cleanup function
	cleanup := func() {
		_ = os.Setenv(ciutil.EnvGitLabCI, originalGitLabCI)
		_ = os.Setenv(ciutil.EnvGitLabProjectDir, originalGitLabProjectDir)
	}
	defer cleanup()

	tests := []struct {
		name             string
		gitlabCI         string
		gitlabProjectDir string
		expected         bool
	}{
		{
			name:             "no_gitlab_env_vars",
			gitlabCI:         "",
			gitlabProjectDir: "",
			expected:         false,
		},
		{
			name:             "gitlab_ci_only",
			gitlabCI:         "true",
			gitlabProjectDir: "",
			expected:         false,
		},
		{
			name:             "gitlab_project_dir_only",
			gitlabCI:         "",
			gitlabProjectDir: "/builds/user/project",
			expected:         false,
		},
		{
			name:             "both_gitlab_env_vars_set",
			gitlabCI:         "true",
			gitlabProjectDir: "/builds/user/project",
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for this test
			_ = os.Setenv(ciutil.EnvGitLabCI, tt.gitlabCI)
			_ = os.Setenv(ciutil.EnvGitLabProjectDir, tt.gitlabProjectDir)

			result := ciutil.IsGitLabCI()
			assert.Equal(t, tt.expected, result)
		})
	}
}
