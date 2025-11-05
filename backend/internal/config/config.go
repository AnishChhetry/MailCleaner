package config

import (
	"errors"
	"os"
)

// Config holds runtime configuration.
type Config struct {
	HttpAddr           string
	PostgresDSN        string
	RedisURL           string // Changed from RedisAddr
	GoogleClientID     string
	GoogleClientSecret string
}

// Load loads from environment variables or .env.
func Load() (*Config, error) {
	cfg := &Config{
		HttpAddr:           getEnv("HTTP_ADDR", ":8080"),
		PostgresDSN:        os.Getenv("POSTGRES_DSN"),
		RedisURL:           os.Getenv("REDIS_URL"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	}

	// Validate required fields
	if cfg.PostgresDSN == "" || cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" || cfg.RedisURL == "" {
		return nil, errors.New("missing required environment variables: POSTGRES_DSN, GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, REDIS_URL")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
