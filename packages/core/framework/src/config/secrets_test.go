package config

import (
	"strings"
	"testing"
)

func TestSecretsOutsideProductionUseTheDefault(t *testing.T) {
	t.Setenv("NODE_ENV", "development")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("COOKIE_SECRET", "")
	cfg := DefineConfig(InputConfig{})
	if cfg.ProjectConfig.HTTP.JwtSecret != "supersecret" || cfg.ProjectConfig.HTTP.CookieSecret != "supersecret" {
		t.Fatalf("development must default to supersecret, got %q / %q", cfg.ProjectConfig.HTTP.JwtSecret, cfg.ProjectConfig.HTTP.CookieSecret)
	}
	if err := Validate(cfg, nil); err != nil {
		t.Fatal(err)
	}
}

func TestProductionNeverFallsBackToTheDefaultSecret(t *testing.T) {
	t.Setenv("NODE_ENV", "production")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("COOKIE_SECRET", "")
	cfg := DefineConfig(InputConfig{})
	if cfg.ProjectConfig.HTTP.JwtSecret != "" || cfg.ProjectConfig.HTTP.CookieSecret != "" {
		t.Fatalf("production must not default the secrets, got %q / %q", cfg.ProjectConfig.HTTP.JwtSecret, cfg.ProjectConfig.HTTP.CookieSecret)
	}
	err := Validate(cfg, nil)
	if err == nil || err.Error() != "[config] ⚠️ http.jwtSecret not found." {
		t.Fatalf("want the jwtSecret error, got %v", err)
	}

	t.Setenv("JWT_SECRET", "jwt-from-env")
	err = Validate(DefineConfig(InputConfig{}), nil)
	if err == nil || !strings.Contains(err.Error(), "http.cookieSecret not found.") {
		t.Fatalf("want the cookieSecret error, got %v", err)
	}
}

func TestProductionSecretsFromEnvOrConfig(t *testing.T) {
	t.Setenv("NODE_ENV", "production")
	t.Setenv("JWT_SECRET", "jwt-from-env")
	t.Setenv("COOKIE_SECRET", "cookie-from-env")
	cfg := DefineConfig(InputConfig{})
	if cfg.ProjectConfig.HTTP.JwtSecret != "jwt-from-env" || cfg.ProjectConfig.HTTP.CookieSecret != "cookie-from-env" {
		t.Fatalf("env secrets not used: %+v", cfg.ProjectConfig.HTTP)
	}
	if err := Validate(cfg, nil); err != nil {
		t.Fatal(err)
	}

	in := InputConfig{}
	in.ProjectConfig.HTTP.JwtSecret = "from-config"
	in.ProjectConfig.HTTP.CookieSecret = "from-config-cookie"
	cfg = DefineConfig(in)
	if cfg.ProjectConfig.HTTP.JwtSecret != "from-config" || cfg.ProjectConfig.HTTP.CookieSecret != "from-config-cookie" {
		t.Fatalf("explicit config secrets must win: %+v", cfg.ProjectConfig.HTTP)
	}
}
