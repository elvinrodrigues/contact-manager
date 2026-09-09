package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all environment-based configuration. Load it once at startup;
// nothing below main should read the environment directly.
type Config struct {
	Port           string
	AllowedOrigins []string

	AdminEmail string

	BaseURL      string
	ResendAPIKey string
	DebugEmail   bool

	MaxBodyBytes    int64
	ShutdownTimeout time.Duration

	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

// Defaults for the HTTP server. The zero-value http.Server has no timeouts at
// all, which leaves the process open to connections that never finish sending.
const (
	DefaultMaxBodyBytes    = 1 << 20 // 1 MiB
	DefaultShutdownTimeout = 15 * time.Second

	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 60 * time.Second
)

// Load reads configuration and validates it. It returns an error rather than
// exiting so the caller controls process lifetime.
func Load() (*Config, error) {
	if os.Getenv("JWT_SECRET") == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		AllowedOrigins:    splitList(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		AdminEmail:        strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))),
		BaseURL:           strings.TrimSuffix(getEnv("BASE_URL", "http://localhost:3000"), "/"),
		ResendAPIKey:      os.Getenv("RESEND_API_KEY"),
		DebugEmail:        boolEnv("DEBUG_EMAIL"),
		MaxBodyBytes:      int64Env("MAX_BODY_BYTES", DefaultMaxBodyBytes),
		ShutdownTimeout:   DefaultShutdownTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}

	if len(cfg.AllowedOrigins) == 0 {
		return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS must list at least one origin")
	}
	return cfg, nil
}

// Warnings returns non-fatal configuration problems worth logging at boot.
func (c *Config) Warnings() []string {
	var warnings []string
	if c.ResendAPIKey == "" && !c.DebugEmail {
		warnings = append(warnings,
			"RESEND_API_KEY is not set and DEBUG_EMAIL is false — verification and reset emails will not be sent")
	}
	if c.DebugEmail {
		warnings = append(warnings,
			"DEBUG_EMAIL=true — emails are logged as delivered rather than sent (token contents are still withheld)")
	}
	if c.AdminEmail == "" {
		warnings = append(warnings,
			"ADMIN_EMAIL is not set — no account will be granted the admin role at startup")
	}
	for _, origin := range c.AllowedOrigins {
		if origin == "*" {
			warnings = append(warnings,
				"CORS_ALLOWED_ORIGINS contains '*' — any site can call this API with a user's credentials")
		}
	}
	return warnings
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func boolEnv(key string) bool {
	return strings.EqualFold(os.Getenv(key), "true")
}

func int64Env(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
