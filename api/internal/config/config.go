package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the API server, loaded from environment variables.
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	HMACKey     string // Server-side HMAC key for secret name hashing
}

// Load reads configuration from environment variables.
// Returns an error if any required variable is missing.
func Load() (*Config, error) {
	c := &Config{
		Port:        envOr("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    envOr("REDIS_URL", "redis://localhost:6379/0"),
		HMACKey:     os.Getenv("HMAC_KEY"),
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
