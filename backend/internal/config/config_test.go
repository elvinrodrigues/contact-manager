package config

import (
	"strings"
	"testing"
)

func TestLoadRequiresJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load succeeded without JWT_SECRET")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("AllowedOrigins = %v, want the localhost default", cfg.AllowedOrigins)
	}
	if cfg.MaxBodyBytes != DefaultMaxBodyBytes {
		t.Errorf("MaxBodyBytes = %d, want %d", cfg.MaxBodyBytes, DefaultMaxBodyBytes)
	}
	// Every HTTP timeout must be set; the zero-value server has none.
	if cfg.ReadTimeout == 0 || cfg.WriteTimeout == 0 || cfg.IdleTimeout == 0 || cfg.ReadHeaderTimeout == 0 {
		t.Error("an HTTP timeout was left at zero")
	}
}

func TestLoadParsesLists(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example.com, https://b.example.com ,")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"https://a.example.com", "https://b.example.com"}
	if len(cfg.AllowedOrigins) != len(want) {
		t.Fatalf("AllowedOrigins = %v, want %v", cfg.AllowedOrigins, want)
	}
	for i := range want {
		if cfg.AllowedOrigins[i] != want[i] {
			t.Errorf("origin %d = %q, want %q", i, cfg.AllowedOrigins[i], want[i])
		}
	}
}

func TestLoadNormalizesValues(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("ADMIN_EMAIL", "  Ada@Example.COM ")
	t.Setenv("BASE_URL", "https://app.example.com/")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AdminEmail != "ada@example.com" {
		t.Errorf("AdminEmail = %q, want it lowercased and trimmed", cfg.AdminEmail)
	}
	// A trailing slash here would produce links like "https://app//verify".
	if cfg.BaseURL != "https://app.example.com" {
		t.Errorf("BaseURL = %q, want the trailing slash removed", cfg.BaseURL)
	}
}

func TestWarnings(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	t.Setenv("RESEND_API_KEY", "")
	t.Setenv("DEBUG_EMAIL", "false")
	t.Setenv("ADMIN_EMAIL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	warnings := strings.Join(cfg.Warnings(), "\n")
	for _, want := range []string{"RESEND_API_KEY", "ADMIN_EMAIL", "'*'"} {
		if !strings.Contains(warnings, want) {
			t.Errorf("warnings do not mention %s:\n%s", want, warnings)
		}
	}
}

func TestMaxBodyBytesOverride(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")

	t.Run("valid override", func(t *testing.T) {
		t.Setenv("MAX_BODY_BYTES", "4096")
		cfg, _ := Load()
		if cfg.MaxBodyBytes != 4096 {
			t.Errorf("MaxBodyBytes = %d, want 4096", cfg.MaxBodyBytes)
		}
	})

	// A nonsense value must fall back to the default rather than disabling the cap.
	for _, bad := range []string{"0", "-1", "lots"} {
		t.Run("rejects "+bad, func(t *testing.T) {
			t.Setenv("MAX_BODY_BYTES", bad)
			cfg, _ := Load()
			if cfg.MaxBodyBytes != DefaultMaxBodyBytes {
				t.Errorf("MaxBodyBytes = %d, want the default %d", cfg.MaxBodyBytes, DefaultMaxBodyBytes)
			}
		})
	}
}
