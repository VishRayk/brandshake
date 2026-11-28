// Package config loads service configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env            string // "development" | "production"
	Addr           string
	DatabaseURL    string
	JWTSecret      []byte
	AccessTokenTTL time.Duration
	AIServiceURL   string
	AllowedOrigins []string
	PublicURL      string // where the web app is served; used in email links
	Workers        int    // background job workers in this process; 0 disables
	EmailProvider  string // "log" (development)
}

const devSecret = "dev-only-insecure-secret-change-me"

func Load() (Config, error) {
	cfg := Config{
		Env:            getenv("BRANDSHAKE_ENV", "development"),
		Addr:           getenv("BRANDSHAKE_ADDR", ":8080"),
		DatabaseURL:    getenv("BRANDSHAKE_DATABASE_URL", "postgres://brandshake:brandshake@localhost:5432/brandshake?sslmode=disable"),
		AIServiceURL:   getenv("BRANDSHAKE_AI_URL", "http://localhost:8090"),
		AllowedOrigins: splitCSV(getenv("BRANDSHAKE_ALLOWED_ORIGINS", "http://localhost:5173")),
		PublicURL:      strings.TrimRight(getenv("BRANDSHAKE_PUBLIC_URL", "http://localhost:5173"), "/"),
		EmailProvider:  getenv("BRANDSHAKE_EMAIL_PROVIDER", "log"),
	}

	workers, err := strconv.Atoi(getenv("BRANDSHAKE_WORKERS", "2"))
	if err != nil || workers < 0 || workers > 32 {
		return Config{}, errors.New("BRANDSHAKE_WORKERS must be 0–32")
	}
	cfg.Workers = workers
	if cfg.EmailProvider != "log" {
		return Config{}, fmt.Errorf("BRANDSHAKE_EMAIL_PROVIDER %q is not supported (use \"log\")", cfg.EmailProvider)
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
