package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Config holds resolved CLI configuration.
type Config struct {
	APIURL   string // ZENV_API_URL or default
	Token    string // ZENV_TOKEN
	VaultKey string // ZENV_VAULT_KEY
	Project  string // from flag, .zenv file, or ZENV_PROJECT
	Env      string // from flag, .zenv file, or ZENV_ENV
}

// Load resolves config from flags → .zenv file → env vars → defaults.
// Flag values (if non-empty) always win.
func Load(flagProject, flagEnv string) *Config {
	c := &Config{
		APIURL:   envOr("ZENV_API_URL", "http://localhost:8080"),
		Token:    envOr("ZENV_TOKEN", loadStoredToken()),
		VaultKey: os.Getenv("ZENV_VAULT_KEY"),
	}

	// .zenv file (walk up from cwd)
	dotEnv := findDotZenv()
	if dotEnv != nil {
		if c.Project == "" {
			c.Project = dotEnv["project"]
		}
		if c.Env == "" {
			c.Env = dotEnv["env"]
		}
	}

	// Env vars
	if c.Project == "" {
		c.Project = os.Getenv("ZENV_PROJECT")
	}
	if c.Env == "" {
		c.Env = os.Getenv("ZENV_ENV")
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

// findDotZenv walks up the directory tree looking for a .zenv file.
func findDotZenv() map[string]string {
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}

	for {
		path := filepath.Join(dir, ".zenv")
		if f, err := os.Open(path); err == nil {
			defer f.Close()
			return parseDotZenv(f)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil
}

func parseDotZenv(f *os.File) map[string]string {
	result := make(map[string]string)
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

// loadStoredToken reads ZENV_TOKEN from ~/.config/zenv/credentials.
func loadStoredToken() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(configDir, "zenv", "credentials"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ZENV_TOKEN=") {
			return strings.TrimPrefix(line, "ZENV_TOKEN=")
		}
	}
	return ""
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
