package config_test

import (
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/config"
)

func TestDefaultsAreDocumented(t *testing.T) {
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
	if cfg.DatabaseHost != "localhost" {
		t.Errorf("default DATABASE_HOST must be localhost, got %q", cfg.DatabaseHost)
	}
	if cfg.DatabasePort != "5432" {
		t.Errorf("default DATABASE_PORT must be 5432, got %q", cfg.DatabasePort)
	}
	if cfg.DatabaseName != "postgres" {
		t.Errorf("default DATABASE_NAME must be postgres, got %q", cfg.DatabaseName)
	}
	if cfg.DeepgramModel != "nova-3" || cfg.DeepgramLanguage != "es" {
		t.Errorf("Deepgram defaults must be nova-3/es, got %q/%q", cfg.DeepgramModel, cfg.DeepgramLanguage)
	}
	if cfg.DeepgramBaseURL != "https://api.deepgram.com/v1" {
		t.Errorf("default Deepgram base URL is wrong: %q", cfg.DeepgramBaseURL)
	}
}

func TestEnvNamesAreHonored(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("CORS_ALLOWED_ORIGIN", "https://app.example.com")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("DATABASE_USERNAME", "u")
	t.Setenv("DATABASE_PASSWORD", "p")
	t.Setenv("DEEPGRAM_API_KEY", "test-key")
	t.Setenv("DEEPGRAM_MODEL", "nova-2")
	t.Setenv("DEEPGRAM_LANGUAGE", "es-419")
	t.Setenv("DEEPGRAM_BASE_URL", "https://deepgram.test/v1")
	cfg := config.Load()
	if cfg.ServerPort != "9090" || cfg.CORSAllowedOrigin != "https://app.example.com" {
		t.Errorf("env overrides not honored: %+v", cfg)
	}
	if cfg.DatabaseURL != "postgres://u:p@localhost:5432/db" || cfg.DatabaseUser != "u" || cfg.DatabasePassword != "p" {
		t.Errorf("database env names not honored: %+v", cfg)
	}
}

func TestDeepgramEnvironmentOverrides(t *testing.T) {
	t.Setenv("DEEPGRAM_API_KEY", "test-key")
	t.Setenv("DEEPGRAM_MODEL", "nova-2")
	t.Setenv("DEEPGRAM_LANGUAGE", "es-419")
	t.Setenv("DEEPGRAM_BASE_URL", "https://deepgram.test/v1")
	cfg := config.Load()
	if cfg.DeepgramAPIKey != "test-key" || cfg.DeepgramModel != "nova-2" || cfg.DeepgramLanguage != "es-419" || cfg.DeepgramBaseURL != "https://deepgram.test/v1" {
		t.Fatalf("Deepgram environment overrides not honored: %+v", cfg)
	}
}

func TestJdbcPrefixIsStrippedForGoDriver(t *testing.T) {
	t.Setenv("DATABASE_URL", "jdbc:postgresql://localhost:5432/uml_architect")
	cfg := config.Load()
	if cfg.DatabaseURL != "postgresql://localhost:5432/uml_architect" {
		t.Errorf("jdbc: prefix must be stripped, got %q", cfg.DatabaseURL)
	}
}

func TestDatabasePartsAreEnvDriven(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_USERNAME", "appuser")
	t.Setenv("DATABASE_PASSWORD", "s3cret")
	t.Setenv("DATABASE_HOST", "db.internal")
	t.Setenv("DATABASE_PORT", "5433")
	t.Setenv("DATABASE_NAME", "appdb")
	cfg := config.Load()
	if cfg.DatabaseUser != "appuser" || cfg.DatabasePassword != "s3cret" {
		t.Errorf("database user/password env not honored: %+v", cfg)
	}
	if cfg.DatabaseHost != "db.internal" || cfg.DatabasePort != "5433" || cfg.DatabaseName != "appdb" {
		t.Errorf("database host/port/name env not honored: %+v", cfg)
	}
	if got := cfg.EffectiveDatabaseURL(); got != "postgres://appuser:s3cret@db.internal:5433/appdb" {
		t.Errorf("fallback URL must be built from env parts, got %q", got)
	}
}

func TestEffectiveDatabaseURLPrefersFullURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("DATABASE_HOST", "ignored")
	cfg := config.Load()
	if got := cfg.EffectiveDatabaseURL(); got != "postgres://u:p@localhost:5432/db" {
		t.Errorf("DATABASE_URL must win over parts, got %q", got)
	}
}

func TestEffectiveDatabaseURLEscapesPassword(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_USERNAME", "appuser")
	t.Setenv("DATABASE_PASSWORD", "p@ss:w/rd")
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_NAME", "appdb")
	cfg := config.Load()
	if got := cfg.EffectiveDatabaseURL(); got != "postgres://appuser:p%40ss%3Aw%2Frd@localhost:5432/appdb" {
		t.Errorf("password must be URL-escaped, got %q", got)
	}
}
