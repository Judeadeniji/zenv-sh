package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all configuration for the API server, loaded from environment variables.
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	Verbose     bool   // Enable debug-level logging (request logs, etc.)
	CORSOrigins string // Comma-separated allowed CORS origins

	// AuthServerURL is the Better Auth HTTP base URL, e.g. https://auth.example.com/api/auth
	// (no trailing slash). Handlers reach the auth server only via internal/auth_client
	// (no Better Auth Go SDK in this repo).
	AuthServerURL string
}

// Load reads configuration from environment variables.
// Returns an error if any required variable is missing.
func Load() (*Config, error) {
	verbose := os.Getenv("VERBOSE") == "1" || os.Getenv("VERBOSE") == "true" || os.Getenv("LOG_LEVEL") == "debug"

	c := &Config{
		Port:        envOr("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    envOr("REDIS_URL", "redis://localhost:6379/0"),
		Verbose:     verbose,
		CORSOrigins: os.Getenv("CORS_ORIGINS"),
		AuthServerURL: strings.TrimRight(strings.TrimSpace(os.Getenv("AUTH_SERVER_URL")), "/"),
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.AuthServerURL == "" {
		return nil, fmt.Errorf("AUTH_SERVER_URL is required (Better Auth base URL, e.g. https://auth.example.com/api/auth)")
	}

	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
