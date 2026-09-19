// Package config loads the process environment using exactly the names from
// backend/src/main/resources/application.yml: DATABASE_URL,
// DATABASE_USERNAME, DATABASE_PASSWORD, SERVER_PORT, CORS_ALLOWED_ORIGIN
// (falling back to FRONTEND_URL, then the local default).
package config

import (
	"os"
	"strings"
)

// Config holds the runtime settings for the Go backend.
type Config struct {
	DatabaseURL       string
	DatabaseUser      string
	DatabasePassword  string
	ServerPort        string
	CORSAllowedOrigin string
}

// Load reads the environment with the application.yml defaults.
func Load() Config {
	return Config{
		DatabaseURL:       normalizeDatabaseURL(os.Getenv("DATABASE_URL")),
		DatabaseUser:      firstNonEmpty(os.Getenv("DATABASE_USERNAME"), "postgres"),
		DatabasePassword:  os.Getenv("DATABASE_PASSWORD"),
		ServerPort:        firstNonEmpty(os.Getenv("SERVER_PORT"), "8080"),
		CORSAllowedOrigin: firstNonEmpty(os.Getenv("CORS_ALLOWED_ORIGIN"), os.Getenv("FRONTEND_URL"), "http://localhost:3000"),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// normalizeDatabaseURL strips the jdbc: prefix Spring accepts so the value can
// feed the Go Postgres driver unchanged.
func normalizeDatabaseURL(raw string) string {
	return strings.TrimPrefix(strings.TrimPrefix(raw, "jdbc:"), "jdbc:")
}
