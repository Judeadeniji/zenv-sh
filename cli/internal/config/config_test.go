package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	for _, k := range []string{"ZENV_API_URL", "ZENV_AUTH_URL", "ZENV_TOKEN", "ZENV_PROJECT_KEY", "ZENV_PROJECT", "ZENV_ENV"} {
		t.Setenv(k, "")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := Load("", "")

	if cfg.APIURL != "http://localhost:8080" {
		t.Errorf("APIURL = %q, want default", cfg.APIURL)
	}
	if cfg.Token != "" {
		t.Errorf("Token = %q, want empty", cfg.Token)
	}
}

func TestLoad_EnvVarsOverrideDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("ZENV_TOKEN", "ze_dev_testtoken")
	t.Setenv("ZENV_PROJECT_KEY", "my-project-key")
	t.Setenv("ZENV_PROJECT", "proj-123")
	t.Setenv("ZENV_ENV", "staging")

	cfg := Load("", "")

	if cfg.Token != "ze_dev_testtoken" {
		t.Errorf("Token = %q", cfg.Token)
	}
	if cfg.ProjectKey != "my-project-key" {
		t.Errorf("ProjectKey = %q", cfg.ProjectKey)
	}
}

func TestLoad_PerProjectCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	for _, k := range []string{"ZENV_TOKEN", "ZENV_PROJECT_KEY", "ZENV_PROJECT", "ZENV_ENV"} {
		t.Setenv(k, "")
	}

	projectID := "31a4884b-ec44-437d-a7c7-e17752137cfa"
	credDir := filepath.Join(tmpDir, "zenv", "projects", projectID)
	os.MkdirAll(credDir, 0700)
	os.WriteFile(filepath.Join(credDir, "credentials"),
		[]byte("token=ze_proj_token\nproject_key=proj-key\n"), 0600)

	workDir := filepath.Join(tmpDir, "repo")
	os.MkdirAll(workDir, 0700)
	os.WriteFile(filepath.Join(workDir, ".zenv"), []byte("project="+projectID+"\nenv=development\n"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(workDir)
	defer os.Chdir(origDir)

	cfg := Load("", "")

	if cfg.Token != "ze_proj_token" {
		t.Errorf("Token = %q, want per-project token", cfg.Token)
	}
	if cfg.ProjectKey != "proj-key" {
		t.Errorf("ProjectKey = %q, want per-project key", cfg.ProjectKey)
	}
}

func TestLoad_ProjectCredsBeatGlobal(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	for _, k := range []string{"ZENV_TOKEN", "ZENV_PROJECT_KEY"} {
		t.Setenv(k, "")
	}

	projectID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	zenvDir := filepath.Join(tmpDir, "zenv")
	os.MkdirAll(filepath.Join(zenvDir, "projects", projectID), 0700)
	os.WriteFile(filepath.Join(zenvDir, "credentials"), []byte("token=ze_global\nproject_key=global-key\n"), 0600)
	os.WriteFile(filepath.Join(zenvDir, "projects", projectID, "credentials"),
		[]byte("token=ze_project\nproject_key=project-key\n"), 0600)

	workDir := filepath.Join(tmpDir, "repo")
	os.MkdirAll(workDir, 0700)
	os.WriteFile(filepath.Join(workDir, ".zenv"), []byte("project="+projectID+"\n"), 0644)
	origDir, _ := os.Getwd()
	os.Chdir(workDir)
	defer os.Chdir(origDir)

	cfg := Load("", "")

	if cfg.Token != "ze_project" {
		t.Errorf("Token = %q, want project-scoped", cfg.Token)
	}
	if cfg.ProjectKey != "project-key" {
		t.Errorf("ProjectKey = %q, want project-scoped", cfg.ProjectKey)
	}
}

func TestSetForProject(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	projectID := "11111111-2222-3333-4444-555555555555"
	if err := SetForProject(projectID, KeyToken, "ze_saved"); err != nil {
		t.Fatalf("SetForProject: %v", err)
	}
	got := GetForProject(projectID, KeyToken)
	if got != "ze_saved" {
		t.Errorf("GetForProject = %q", got)
	}
}

func TestSet_RejectsCredentialViaSet(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := Set("token", "ze_bad"); err == nil {
		t.Fatal("Set(token) should fail — use SetForProject")
	}
}

func TestSetGlobalSecret(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := SetGlobalSecret("token", "ze_global"); err != nil {
		t.Fatalf("SetGlobalSecret: %v", err)
	}
	if loadGlobalCredentials()[KeyToken] != "ze_global" {
		t.Error("global credentials should contain token")
	}
}

func TestListConfiguredProjects(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	_ = SetForProject("aaaaaaaa-bbbb-cccc-dddd-111111111111", KeyToken, "ze_a")
	_ = SetForProject("aaaaaaaa-bbbb-cccc-dddd-222222222222", KeyToken, "ze_b")

	ids, err := ListConfiguredProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("got %d projects, want 2", len(ids))
	}
}

func TestSetLocal_WritesToDotZenv(t *testing.T) {
	tmpDir := t.TempDir()
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
		t.Errorf(".zenv = %q", string(data))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
