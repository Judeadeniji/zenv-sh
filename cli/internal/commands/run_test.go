package commands

import (
	"reflect"
	"sort"
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

func TestInjectBuildKitSecrets_SingleKey(t *testing.T) {
	keys := []string{"MY_SECRET"}
	args := []string{"docker", "build", "."}
	result := injectBuildKitSecrets(args, keys)

	expected := []string{"docker", "build", "--secret", "id=MY_SECRET,env=MY_SECRET", "."}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("injectBuildKitSecrets() = %v, want %v", result, expected)
	}
}

func TestInjectBuildKitSecrets_AdditionalTools(t *testing.T) {
	keys := []string{"SECRET"}

	tests := []struct {
<<<<<<< HEAD
		name    string
		args    []string
		wantInj bool // whether secrets should be injected
=======
		name     string
		args     []string
		wantInj  bool // whether secrets should be injected
>>>>>>> origin/main
	}{
		{
			name:    "podman build",
			args:    []string{"podman", "build", "-t", "img", "."},
			wantInj: true,
		},
		{
			name:    "buildah build",
			args:    []string{"buildah", "build", "--layers", "."},
			wantInj: true,
		},
		{
			name:    "single element args (no-op)",
			args:    []string{"docker"},
			wantInj: false,
		},
		{
			name:    "docker without build subcommand (no-op)",
			args:    []string{"docker", "run", "myimage"},
			wantInj: false,
		},
		{
			name:    "docker buildx without build (no-op)",
			args:    []string{"docker", "buildx", "ls"},
			wantInj: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectBuildKitSecrets(tt.args, keys)
			containsSecret := false
			for _, arg := range result {
				if arg == "--secret" {
					containsSecret = true
					break
				}
			}
			if containsSecret != tt.wantInj {
				t.Errorf("injectBuildKitSecrets(%v) secret injected=%v, want %v", tt.args, containsSecret, tt.wantInj)
			}
			// Original args must remain present in result.
			for _, orig := range tt.args {
				found := false
				for _, r := range result {
					if r == orig {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("original arg %q missing from result %v", orig, result)
				}
			}
		})
	}
}

<<<<<<< HEAD
=======
func TestInjectBuildKitSecrets_SingleKey(t *testing.T) {
	keys := []string{"MY_SECRET"}
	args := []string{"docker", "build", "."}
	result := injectBuildKitSecrets(args, keys)

	expected := []string{"docker", "build", "--secret", "id=MY_SECRET,env=MY_SECRET", "."}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("injectBuildKitSecrets() = %v, want %v", result, expected)
	}
}

>>>>>>> origin/main
func TestMergeEnv_OverridesWin(t *testing.T) {
	base := []string{"FOO=original", "BAR=keep", "PATH=/usr/bin"}
	overrides := map[string]string{
		"FOO": "overridden",
		"NEW": "added",
	}

	result := mergeEnv(base, overrides)

	// Convert to a map for easy lookup.
	env := make(map[string]string)
	for _, kv := range result {
		idx := 0
		for idx < len(kv) && kv[idx] != '=' {
			idx++
		}
		if idx < len(kv) {
			env[kv[:idx]] = kv[idx+1:]
		}
	}

	if env["FOO"] != "overridden" {
		t.Errorf("Expected FOO=overridden, got %s", env["FOO"])
	}
	if env["BAR"] != "keep" {
		t.Errorf("Expected BAR=keep, got %s", env["BAR"])
	}
	if env["NEW"] != "added" {
		t.Errorf("Expected NEW=added, got %s", env["NEW"])
	}
	if env["PATH"] != "/usr/bin" {
		t.Errorf("Expected PATH=/usr/bin, got %s", env["PATH"])
	}
}

func TestMergeEnv_NoDuplicates(t *testing.T) {
	base := []string{"KEY=old", "OTHER=val"}
	overrides := map[string]string{"KEY": "new"}

	result := mergeEnv(base, overrides)

	count := 0
	for _, kv := range result {
<<<<<<< HEAD
		if len(kv) >= 4 && kv[:4] == "KEY=" {
=======
		if len(kv) >= 3 && kv[:4] == "KEY=" {
>>>>>>> origin/main
			count++
		}
	}
	if count != 1 {
		t.Errorf("Expected exactly 1 KEY entry, got %d in %v", count, result)
	}
}

func TestMergeEnv_MalformedEntryPreserved(t *testing.T) {
	base := []string{"MALFORMED_NO_EQUALS", "GOOD=val"}
	overrides := map[string]string{}

	result := mergeEnv(base, overrides)

	// Malformed entry should be included as-is.
	found := false
	for _, kv := range result {
		if kv == "MALFORMED_NO_EQUALS" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected malformed entry to be preserved, got %v", result)
	}
}

func TestMergeEnv_EmptyBase(t *testing.T) {
	overrides := map[string]string{"A": "1", "B": "2"}
	result := mergeEnv([]string{}, overrides)

	if len(result) != 2 {
		t.Errorf("Expected 2 entries, got %d: %v", len(result), result)
	}
	sort.Strings(result)
	if result[0] != "A=1" {
		t.Errorf("Expected A=1, got %s", result[0])
	}
	if result[1] != "B=2" {
		t.Errorf("Expected B=2, got %s", result[1])
	}
}

func TestMergeEnv_EmptyOverrides(t *testing.T) {
	base := []string{"X=1", "Y=2"}
	result := mergeEnv(base, map[string]string{})

	if !reflect.DeepEqual(result, base) {
		t.Errorf("Expected unchanged base, got %v", result)
	}
}

<<<<<<< HEAD
func TestMergeEnv_ValueContainsEquals(t *testing.T) {
	// Env vars like BASE64=abc=def= should be split only on the first '='
	base := []string{"ENCODED=abc=def="}
	overrides := map[string]string{}

	result := mergeEnv(base, overrides)

	if len(result) != 1 || result[0] != "ENCODED=abc=def=" {
		t.Errorf("Expected ENCODED=abc=def= preserved intact, got %v", result)
	}
}

=======
>>>>>>> origin/main
func TestZeroBytes(t *testing.T) {
	b := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0xFF}
	zeroBytes(b)

	for i, v := range b {
		if v != 0 {
			t.Errorf("Expected b[%d]=0, got %d", i, v)
		}
	}
}

func TestZeroBytes_Empty(t *testing.T) {
	// Should not panic on empty slice.
	zeroBytes([]byte{})
}

func TestZeroBytes_Nil(t *testing.T) {
	// Should not panic on nil slice.
	zeroBytes(nil)
}
<<<<<<< HEAD

func TestZeroBytes_SingleByte(t *testing.T) {
	b := []byte{0xFF}
	zeroBytes(b)
	if b[0] != 0 {
		t.Errorf("Expected b[0]=0, got %d", b[0])
	}
}
=======
>>>>>>> origin/main
