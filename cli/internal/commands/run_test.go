package commands

import (
	"reflect"
	"testing"
)

func TestInjectBuildKitSecrets(t *testing.T) {
	keys := []string{"API_KEY", "DB_URL"}

	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name: "docker build auto-detection",
			args: []string{"docker", "build", "-t", "myapp", "."},
			expected: []string{
				"docker", "build",
				"--secret", "id=API_KEY,env=API_KEY",
				"--secret", "id=DB_URL,env=DB_URL",
				"-t", "myapp", ".",
			},
		},
		{
			name: "docker buildx build auto-detection",
			args: []string{"docker", "buildx", "build", "--progress=plain", "."},
			expected: []string{
				"docker", "buildx", "build",
				"--secret", "id=API_KEY,env=API_KEY",
				"--secret", "id=DB_URL,env=DB_URL",
				"--progress=plain", ".",
			},
		},
		{
			name: "nerdctl build (fallback)",
			args: []string{"nerdctl", "build", "-t", "myapp", "."},
			expected: []string{
				"nerdctl", "build",
				"--secret", "id=API_KEY,env=API_KEY",
				"--secret", "id=DB_URL,env=DB_URL",
				"-t", "myapp", ".",
			},
		},
		{
			name: "unknown command (no-op)",
			args: []string{"mybuilder", "--some-flag"},
			expected: []string{
				"mybuilder",
				"--some-flag",
			},
		},
		{
			name: "make build (no-op)",
			args: []string{"make", "build", "all"},
			expected: []string{
				"make",
				"build",
				"all",
			},
		},
		{
			name:     "empty keys (no-op)",
			args:     []string{"docker", "build", "."},
			expected: []string{"docker", "build", "."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := keys
			if tt.name == "empty keys (no-op)" {
				k = nil
			}
			result := injectBuildKitSecrets(tt.args, k)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("injectBuildKitSecrets() = %v, want %v", result, tt.expected)
			}
		})
	}
}
