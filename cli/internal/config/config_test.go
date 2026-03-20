package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear all env vars that could affect config.
	for _, k := range []string{"ZENV_API_URL", "ZENV_AUTH_URL", "ZENV_TOKEN", "ZENV_VAULT_KEY", "ZENV_PROJECT", "ZENV_ENV"} {
		t.Setenv(k, "")
	}

	// Use a temp dir so no real config files are read.
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := Load("", "")

	if cfg.APIURL != "http://localhost:8080" {
		t.Errorf("APIURL = %q, want default", cfg.APIURL)
	}
	if cfg.AuthURL != "http://localhost:3000" {
		t.Errorf("AuthURL = %q, want default", cfg.AuthURL)
	}
	if cfg.Token != "" {
		t.Errorf("Token = %q, want empty", cfg.Token)
	}
	if cfg.VaultKey != "" {
		t.Errorf("VaultKey = %q, want empty", cfg.VaultKey)
	}
	if cfg.Project != "" {
		t.Errorf("Project = %q, want empty", cfg.Project)
	}
	if cfg.Env != "" {
		t.Errorf("Env = %q, want empty", cfg.Env)
	}
}

func TestLoad_EnvVarsOverrideDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("ZENV_API_URL", "http://custom-api:9090")
	t.Setenv("ZENV_AUTH_URL", "http://custom-auth:4000")
	t.Setenv("ZENV_TOKEN", "ze_dev_testtoken")
	t.Setenv("ZENV_VAULT_KEY", "my-vault-key")
	t.Setenv("ZENV_PROJECT", "proj-123")
	t.Setenv("ZENV_ENV", "staging")

	cfg := Load("", "")

	if cfg.APIURL != "http://custom-api:9090" {
		t.Errorf("APIURL = %q", cfg.APIURL)
	}
	if cfg.AuthURL != "http://custom-auth:4000" {
		t.Errorf("AuthURL = %q", cfg.AuthURL)
	}
	if cfg.Token != "ze_dev_testtoken" {
		t.Errorf("Token = %q", cfg.Token)
	}
	if cfg.VaultKey != "my-vault-key" {
		t.Errorf("VaultKey = %q", cfg.VaultKey)
	}
	if cfg.Project != "proj-123" {
		t.Errorf("Project = %q", cfg.Project)
	}
	if cfg.Env != "staging" {
		t.Errorf("Env = %q", cfg.Env)
	}
}

func TestLoad_FlagsOverrideEverything(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("ZENV_PROJECT", "env-project")
	t.Setenv("ZENV_ENV", "env-env")

	cfg := Load("flag-project", "flag-env")

	if cfg.Project != "flag-project" {
		t.Errorf("Project = %q, want flag-project", cfg.Project)
	}
	if cfg.Env != "flag-env" {
		t.Errorf("Env = %q, want flag-env", cfg.Env)
	}
}

func TestLoad_GlobalConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Clear env vars so config file values win.
	for _, k := range []string{"ZENV_API_URL", "ZENV_AUTH_URL", "ZENV_TOKEN", "ZENV_VAULT_KEY", "ZENV_PROJECT", "ZENV_ENV"} {
		t.Setenv(k, "")
	}

	zenvDir := filepath.Join(tmpDir, "zenv")
	os.MkdirAll(zenvDir, 0700)

	// Write global config.
	os.WriteFile(filepath.Join(zenvDir, "config"), []byte("api_url=http://file-api\nauth_url=http://file-auth\n"), 0644)

	// Write credentials.
	os.WriteFile(filepath.Join(zenvDir, "credentials"), []byte("token=ze_dev_filetoken\nvault_key=file-vault\n"), 0600)

	cfg := Load("", "")

	if cfg.APIURL != "http://file-api" {
		t.Errorf("APIURL = %q, want http://file-api", cfg.APIURL)
	}
	if cfg.AuthURL != "http://file-auth" {
		t.Errorf("AuthURL = %q, want http://file-auth", cfg.AuthURL)
	}
	if cfg.Token != "ze_dev_filetoken" {
		t.Errorf("Token = %q, want ze_dev_filetoken", cfg.Token)
	}
	if cfg.VaultKey != "file-vault" {
		t.Errorf("VaultKey = %q, want file-vault", cfg.VaultKey)
	}
}

func TestSet_GlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := Set("api_url", "http://set-test"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got := Get("api_url")
	if got != "http://set-test" {
		t.Errorf("Get = %q, want http://set-test", got)
	}
}

func TestSet_Credentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := Set("token", "ze_dev_secret"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Token should be in credentials file, not config.
	credPath := filepath.Join(tmpDir, "zenv", "credentials")
	data, err := os.ReadFile(credPath)
	if err != nil {
		t.Fatalf("read credentials: %v", err)
	}
	if string(data) == "" || !contains(string(data), "token=ze_dev_secret") {
		t.Errorf("credentials file = %q, want to contain token", string(data))
	}

	// Check permissions.
	info, _ := os.Stat(credPath)
	if info.Mode().Perm() != 0600 {
		t.Errorf("credentials perm = %o, want 0600", info.Mode().Perm())
	}
}

func TestUnset(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	Set("api_url", "http://remove-me")
	if err := Unset("api_url"); err != nil {
		t.Fatalf("Unset: %v", err)
	}

	got := Get("api_url")
	if got != "" {
		t.Errorf("Get after Unset = %q, want empty", got)
	}
}

func TestIsSecret(t *testing.T) {
	if !IsSecret("token") {
		t.Error("token should be secret")
	}
	if !IsSecret("vault_key") {
		t.Error("vault_key should be secret")
	}
	if IsSecret("api_url") {
		t.Error("api_url should not be secret")
	}
}

func TestSetLocal_WritesToDotZenv(t *testing.T) {
	tmpDir := t.TempDir()
	// Change to tmpDir so .zenv is written there.
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	if err := SetLocal("project", "local-proj-123"); err != nil {
		t.Fatalf("SetLocal: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".zenv"))
	if err != nil {
		t.Fatalf("read .zenv: %v", err)
	}
	if !contains(string(data), "project=local-proj-123") {
		t.Errorf(".zenv = %q, want to contain project", string(data))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
