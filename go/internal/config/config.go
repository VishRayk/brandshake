// Package config loads service configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Env            string // "development" | "production"
	Addr           string
	DatabasePath   string
	JWTSecret      []byte
	AccessTokenTTL time.Duration
	AIServiceURL   string
	AllowedOrigins []string
}

const devSecret = "dev-only-insecure-secret-change-me"

func Load() (Config, error) {
	cfg := Config{
		Env:            getenv("BRANDSHAKE_ENV", "development"),
		Addr:           getenv("BRANDSHAKE_ADDR", ":8080"),
		DatabasePath:   getenv("BRANDSHAKE_DB_PATH", "../data/brandshake.db"),
		AIServiceURL:   getenv("BRANDSHAKE_AI_URL", "http://localhost:8090"),
		AllowedOrigins: splitCSV(getenv("BRANDSHAKE_ALLOWED_ORIGINS", "http://localhost:5173")),
	}

	ttl, err := time.ParseDuration(getenv("BRANDSHAKE_ACCESS_TOKEN_TTL", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("BRANDSHAKE_ACCESS_TOKEN_TTL: %w", err)
	}
	cfg.AccessTokenTTL = ttl

	secret := os.Getenv("BRANDSHAKE_JWT_SECRET")
	switch {
	case secret != "":
		if len(secret) < 32 {
			return Config{}, errors.New("BRANDSHAKE_JWT_SECRET must be at least 32 characters")
		}
	case cfg.IsDev():
		secret = devSecret
	default:
		return Config{}, errors.New("BRANDSHAKE_JWT_SECRET is required outside development")
	}
	cfg.JWTSecret = []byte(secret)

	return cfg, nil
}

func (c Config) IsDev() bool { return c.Env == "development" }

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
