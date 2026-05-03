package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Config holds all environment-based configuration for the application.
// Use Load() at startup to populate this struct from environment variables.
type Config struct {
	DatabaseURL     string
	JWTSecret       string
	Port            string
	ResendAPIKey    string
	BaseURL         string
	DebugEmail      bool
	ForceEmail      bool
	CleanupInterval string
}

// Load reads configuration from environment variables, applying defaults
// where appropriate and fatally exiting if required values are missing.
func Load() *Config {
	cfg := &Config{
		DatabaseURL:     getEnv("DATABASE_URL", buildDSN()),
		JWTSecret:       requireEnv("JWT_SECRET"),
		Port:            getEnv("PORT", "8080"),
		ResendAPIKey:    os.Getenv("RESEND_API_KEY"),
		BaseURL:         getEnv("BASE_URL", "http://localhost:3000"),
		DebugEmail:      strings.EqualFold(os.Getenv("DEBUG_EMAIL"), "true"),
		ForceEmail:      strings.EqualFold(os.Getenv("FORCE_EMAIL"), "true"),
		CleanupInterval: os.Getenv("CLEANUP_INTERVAL"),
	}

	if cfg.ResendAPIKey == "" {
		log.Println("[BOOT] WARNING: RESEND_API_KEY not set — emails will not be sent")
	}
	if cfg.DebugEmail {
		log.Println("[BOOT] DEBUG_EMAIL=true — emails will be printed to console instead of sent")
	}

	return cfg
}

// requireEnv reads an environment variable and fatally exits if it is empty.
func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("[BOOT] FATAL: %s environment variable is required", key)
	}
	return v
}

// getEnv reads an environment variable, returning a fallback if empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// buildDSN constructs a lib/pq connection string from individual POSTGRES_* env vars.
// This allows credentials to be defined once (in .env.db) without duplication.
func buildDSN() string {
	host := getEnv("DB_HOST", "localhost")
	user := getEnv("POSTGRES_USER", "contacts_app")
	pass := os.Getenv("POSTGRES_PASSWORD")
	dbName := getEnv("POSTGRES_DB", "contacts_manager")
	sslmode := getEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s", host, user, dbName, sslmode)
	if pass != "" {
		dsn += fmt.Sprintf(" password=%s", pass)
	}
	return dsn
}
