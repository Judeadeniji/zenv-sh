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
//	~/.config/zenv/config       → api_url, auth_url
//	~/.config/zenv/credentials  → token, vault_key
//	.zenv (local, per-project)  → project, env
//
// Resolution order (highest wins):
//
//	CLI flags → local .zenv → global config/credentials → env vars → defaults
const (
	KeyAPIURL   = "api_url"
	KeyAuthURL  = "auth_url"
	KeyToken    = "token"
	KeyVaultKey = "vault_key"
	KeyProject  = "project"
	KeyEnv      = "env"
)

// Config holds resolved CLI configuration.
type Config struct {
	APIURL   string
	AuthURL  string
	Token    string
	VaultKey string
	Project  string
	Env      string
}

// Load resolves config: flags → .zenv → global files → env vars → defaults.
func Load(flagProject, flagEnv string) *Config {
	global := loadGlobalConfig()
	creds := loadGlobalCredentials()
	local := findDotZenv()

	c := &Config{
		APIURL:   first(local[KeyAPIURL], global[KeyAPIURL], os.Getenv("ZENV_API_URL"), "http://localhost:8080"),
		AuthURL:  first(local[KeyAuthURL], global[KeyAuthURL], os.Getenv("ZENV_AUTH_URL"), "http://localhost:3000"),
		Token:    first(creds[KeyToken], os.Getenv("ZENV_TOKEN")),
		VaultKey: first(creds[KeyVaultKey], os.Getenv("ZENV_VAULT_KEY")),
		Project:  first(local[KeyProject], global[KeyProject], os.Getenv("ZENV_PROJECT")),
		Env:      first(local[KeyEnv], global[KeyEnv], os.Getenv("ZENV_ENV")),
	}

	// Flags override everything
	if flagProject != "" {
		c.Project = flagProject
	}
	if flagEnv != "" {
		c.Env = flagEnv
	}

	return c
}

// --- Global file paths ---

// Dir returns the global config directory: ~/.config/zenv
func Dir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "zenv")
}

func globalConfigPath() string   { return filepath.Join(Dir(), "config") }
func globalCredsPath() string    { return filepath.Join(Dir(), "credentials") }

// --- Read/Write helpers ---

// Get reads a single key from the appropriate global file.
func Get(key string) string {
	if isCredential(key) {
		return loadKV(globalCredsPath())[key]
	}
	return loadKV(globalConfigPath())[key]
}

// Set writes a key to the appropriate global file.
func Set(key, value string) error {
	path := globalConfigPath()
	if isCredential(key) {
		path = globalCredsPath()
	}
	return setKV(path, key, value)
}

// Unset removes a key from the appropriate global file.
func Unset(key string) error {
	path := globalConfigPath()
	if isCredential(key) {
		path = globalCredsPath()
	}
	return removeKV(path, key)
}

// ListGlobal returns all key-value pairs from both global files.
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

// isCredential returns true if the key holds a secret.
func isCredential(key string) bool {
	return key == KeyToken || key == KeyVaultKey
}

// IsSecret is the exported version of isCredential.
func IsSecret(key string) bool { return isCredential(key) }

// --- File I/O ---

func loadGlobalConfig() map[string]string      { return loadKV(globalConfigPath()) }
func loadGlobalCredentials() map[string]string  { return loadKV(globalCredsPath()) }

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
	// Credentials get restrictive permissions.
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

// --- .zenv file discovery ---

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

// first returns the first non-empty string.
func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
