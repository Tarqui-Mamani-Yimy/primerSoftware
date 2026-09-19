package config_test

import (
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/config"
)

func TestDefaultsMirrorApplicationYml(t *testing.T) {
	cfg := config.Load()
	if cfg.ServerPort != "8080" {
		t.Errorf("default SERVER_PORT must be 8080, got %q", cfg.ServerPort)
	}
	if cfg.CORSAllowedOrigin != "http://localhost:3000" {
		t.Errorf("default CORS origin must be http://localhost:3000, got %q", cfg.CORSAllowedOrigin)
	}
	if cfg.DatabaseUser != "postgres" {
		t.Errorf("default DATABASE_USERNAME must be postgres, got %q", cfg.DatabaseUser)
	}
}

func TestEnvNamesMatchJavaApplicationYml(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("CORS_ALLOWED_ORIGIN", "https://app.example.com")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("DATABASE_USERNAME", "u")
	t.Setenv("DATABASE_PASSWORD", "p")
	cfg := config.Load()
	if cfg.ServerPort != "9090" || cfg.CORSAllowedOrigin != "https://app.example.com" {
		t.Errorf("env overrides not honored: %+v", cfg)
	}
	if cfg.DatabaseURL != "postgres://u:p@localhost:5432/db" || cfg.DatabaseUser != "u" || cfg.DatabasePassword != "p" {
		t.Errorf("database env names not honored: %+v", cfg)
	}
}

func TestJdbcPrefixIsStrippedForGoDriver(t *testing.T) {
	t.Setenv("DATABASE_URL", "jdbc:postgresql://localhost:5432/uml_architect")
	cfg := config.Load()
	if cfg.DatabaseURL != "postgresql://localhost:5432/uml_architect" {
		t.Errorf("jdbc: prefix must be stripped, got %q", cfg.DatabaseURL)
	}
}
