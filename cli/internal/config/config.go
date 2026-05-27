package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Known config keys and where they live.
//
//	~/.config/zenv/config                      → api_url, auth_url
//	~/.config/zenv/projects/<project-id>/credentials → token, project_key (per project)
//	~/.config/zenv/credentials                 → legacy global secrets (fallback only)
//	.zenv (per-repo)                           → project, env
//
// Resolution order (highest wins):
//
//	CLI flags → .zenv → per-project credentials → legacy global credentials → env vars → defaults
const (
	KeyAPIURL     = "api_url"
	KeyAuthURL    = "auth_url"
	KeyToken      = "token"
	KeyProjectKey = "project_key"
	KeyProject    = "project"
	KeyEnv        = "env"
)

// Config holds resolved CLI configuration.
type Config struct {
	APIURL     string
	AuthURL    string
	Token      string
	ProjectKey string
	Project    string
	Env        string
}

// Load resolves config: flags → .zenv → per-project creds → global creds → env vars → defaults.
func Load(flagProject, flagEnv string) *Config {
	scrubLocalOnlyFromGlobal()

	global := loadGlobalConfig()
	legacyGlobal := loadGlobalCredentials()
	local := findDotZenv()

	project := first(local[KeyProject], os.Getenv("ZENV_PROJECT"))
	env := first(local[KeyEnv], os.Getenv("ZENV_ENV"))

	if flagProject != "" {
		project = flagProject
	}
	if flagEnv != "" {
		env = flagEnv
	}

	var projectCreds map[string]string
	if project != "" {
		migrateGlobalSecretsToProject(project)
		projectCreds = loadProjectCredentials(project)
	}

	c := &Config{
		APIURL:     first(local[KeyAPIURL], global[KeyAPIURL], os.Getenv("ZENV_API_URL"), "http://localhost:8080"),
		AuthURL:    first(local[KeyAuthURL], global[KeyAuthURL], os.Getenv("ZENV_AUTH_URL"), "http://localhost:3000"),
		Token:      first(projectCreds[KeyToken], legacyGlobal[KeyToken], os.Getenv("ZENV_TOKEN")),
		ProjectKey: first(projectCreds[KeyProjectKey], legacyGlobal[KeyProjectKey], os.Getenv("ZENV_PROJECT_KEY")),
		Project:    project,
		Env:        env,
	}

	return c
}

// ResolveProject returns the active project ID from flags, .zenv, or env (no credentials).
func ResolveProject(flagProject string) string {
	local := findDotZenv()
	if flagProject != "" {
		return flagProject
	}
	return first(local[KeyProject], os.Getenv("ZENV_PROJECT"))
}

// --- Global file paths ---

// Dir returns the global config directory: ~/.config/zenv (or ZENV_CONFIG_DIR if set)
func Dir() string {
	if custom := os.Getenv("ZENV_CONFIG_DIR"); custom != "" {
		return custom
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "zenv")
}

func globalConfigPath() string { return filepath.Join(Dir(), "config") }
func globalCredsPath() string  { return filepath.Join(Dir(), "credentials") }

// Get reads api_url / auth_url from global config only.
func Get(key string) string {
	if isLocalOnly(key) || isCredential(key) {
		return ""
	}
	return loadKV(globalConfigPath())[key]
}

// Set writes api_url / auth_url to global config only.
func Set(key, value string) error {
	if isLocalOnly(key) {
		return fmt.Errorf("%s is per-repo — use: zenv config set %s <value>  (writes .zenv)", key, key)
	}
	if isCredential(key) {
		return fmt.Errorf("%s must use per-project storage — run: zenv config set %s <value>  (with project in .zenv)\n  or dare global: zenv config set --global %s <value>", key, key, key)
	}
	return setKV(globalConfigPath(), key, value)
}

// Unset removes a key from global config (not per-project credentials).
func Unset(key string) error {
	if isLocalOnly(key) {
		return fmt.Errorf("%s is per-repo — use: zenv config unset %s  (in .zenv)", key, key)
	}
	if isCredential(key) {
		return fmt.Errorf("%s is not in global config — use: zenv config unset --project <id> %s", key, key)
	}
	return removeKV(globalConfigPath(), key)
}

// UnsetGlobalSecret removes a key from legacy global credentials.
func UnsetGlobalSecret(key string) error {
	if !isCredential(key) {
		return fmt.Errorf("%s is not a global credential", key)
	}
	return removeKV(globalCredsPath(), key)
}

// ListGlobal returns api_url/auth_url and legacy global credentials.
func ListGlobal() map[string]string {
	result := loadKV(globalConfigPath())
	for k, v := range loadKV(globalCredsPath()) {
		result[k] = v
	}
	return result
}

// GetLocal reads a single key from the nearest .zenv file.
func GetLocal(key string) string {
	kv := findDotZenv()
	if kv == nil {
		return ""
	}
	return kv[key]
}

// SetLocal writes a key to the nearest .zenv file (creates in cwd if none).
func SetLocal(key, value string) error {
	if isCredential(key) {
		return fmt.Errorf("%s must be stored per-project — use: zenv login / zenv unlock", key)
	}
	path := findDotZenvPath()
	if path == "" {
		path = ".zenv"
	}
	return setKV(path, key, value)
}

// UnsetLocal removes a key from the nearest .zenv file.
func UnsetLocal(key string) error {
	path := findDotZenvPath()
	if path == "" {
		return nil
	}
	return removeKV(path, key)
}

// ListLocal returns all key-value pairs from the nearest .zenv file.
func ListLocal() map[string]string {
	kv := findDotZenv()
	if kv == nil {
		return map[string]string{}
	}
	return kv
}

func isCredential(key string) bool {
	return key == KeyToken || key == KeyProjectKey
}

func isLocalOnly(key string) bool {
	return key == KeyProject || key == KeyEnv
}

// IsSecret is the exported version of isCredential.
func IsSecret(key string) bool { return isCredential(key) }

// IsLocalOnly reports whether a key must be stored in .zenv (not global config).
func IsLocalOnly(key string) bool { return isLocalOnly(key) }

func scrubLocalOnlyFromGlobal() {
	path := globalConfigPath()
	kv := loadKV(path)
	changed := false
	for _, k := range []string{KeyProject, KeyEnv} {
		if _, ok := kv[k]; ok {
			delete(kv, k)
			changed = true
		}
	}
	if changed {
		_ = writeKV(path, kv)
	}
}

func loadGlobalConfig() map[string]string     { return loadKV(globalConfigPath()) }
func loadGlobalCredentials() map[string]string { return loadKV(globalCredsPath()) }

func loadKV(path string) map[string]string {
	result := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		return result
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}

func setKV(path, key, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	existing := loadKV(path)
	existing[key] = value
	return writeKV(path, existing)
}

func removeKV(path, key string) error {
	existing := loadKV(path)
	if _, ok := existing[key]; !ok {
		return nil
	}
	delete(existing, key)
	return writeKV(path, existing)
}

func writeKV(path string, kv map[string]string) error {
	perm := os.FileMode(0644)
	if strings.HasSuffix(path, "credentials") {
		perm = 0600
	}

	var sb strings.Builder
	for k, v := range kv {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
		sb.WriteString("\n")
	}
	return os.WriteFile(path, []byte(sb.String()), perm)
}

func findDotZenv() map[string]string {
	path := findDotZenvPath()
	if path == "" {
		return nil
	}
	return loadKV(path)
}

func findDotZenvPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		path := filepath.Join(dir, ".zenv")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
