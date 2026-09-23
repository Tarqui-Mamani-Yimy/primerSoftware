// Package config loads the Go backend runtime settings from the process
// environment. Every setting is env-driven with a documented default; no
// credentials, hosts, ports, or database names are hardcoded.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the runtime settings for the Go backend.
//
// Environment (defaults in parentheses):
//   - DATABASE_URL: full postgres URL; when set it wins over the parts below.
//   - DATABASE_USERNAME (postgres), DATABASE_PASSWORD (empty),
//     DATABASE_HOST (localhost), DATABASE_PORT (5432),
//     DATABASE_NAME (postgres).
//   - SERVER_PORT (8080).
//   - CORS_ALLOWED_ORIGIN, falling back to FRONTEND_URL, then
//     http://localhost:3000 (the local frontend dev server).
type Config struct {
	DatabaseURL          string
	DatabaseUser         string
	DatabasePassword     string
	DatabaseHost         string
	DatabasePort         string
	DatabaseName         string
	ServerPort           string
	CORSAllowedOrigin    string
	RealtimeTicketSecret string
	GeminiAPIKey         string
	GeminiModel          string
	GeminiModels         []string
	GeminiBaseURL        string
	DeepgramAPIKey       string
	DeepgramModel        string
	DeepgramLanguage     string
	DeepgramBaseURL      string
}

// Load reads the environment, applying the documented defaults.
func Load() Config {
	loadDotEnv()
	return Config{
		DatabaseURL:          normalizeDatabaseURL(os.Getenv("DATABASE_URL")),
		DatabaseUser:         firstNonEmpty(os.Getenv("DATABASE_USERNAME"), "postgres"),
		DatabasePassword:     os.Getenv("DATABASE_PASSWORD"),
		DatabaseHost:         firstNonEmpty(os.Getenv("DATABASE_HOST"), "localhost"),
		DatabasePort:         firstNonEmpty(os.Getenv("DATABASE_PORT"), "5432"),
		DatabaseName:         firstNonEmpty(os.Getenv("DATABASE_NAME"), "postgres"),
		ServerPort:           firstNonEmpty(os.Getenv("SERVER_PORT"), "8080"),
		CORSAllowedOrigin:    firstNonEmpty(os.Getenv("CORS_ALLOWED_ORIGIN"), os.Getenv("FRONTEND_URL"), "http://localhost:3000"),
		RealtimeTicketSecret: firstNonEmpty(os.Getenv("REALTIME_TICKET_SECRET"), "dev-realtime-ticket-secret-do-not-use-in-prod"),
		GeminiAPIKey:         os.Getenv("GEMINI_API_KEY"),
		GeminiModel:          firstNonEmpty(os.Getenv("GEMINI_MODEL"), "gemini-2.5-flash"),
		GeminiModels:         geminiModels(os.Getenv("GEMINI_MODELS"), os.Getenv("GEMINI_MODEL")),
		GeminiBaseURL:        firstNonEmpty(os.Getenv("GEMINI_BASE_URL"), "https://generativelanguage.googleapis.com/v1beta"),
		DeepgramAPIKey:       os.Getenv("DEEPGRAM_API_KEY"),
		DeepgramModel:        firstNonEmpty(os.Getenv("DEEPGRAM_MODEL"), "nova-3"),
		DeepgramLanguage:     firstNonEmpty(os.Getenv("DEEPGRAM_LANGUAGE"), "es"),
		DeepgramBaseURL:      firstNonEmpty(os.Getenv("DEEPGRAM_BASE_URL"), "https://api.deepgram.com/v1"),
	}
}

// geminiModels preserves the operator's best-to-worst order. Both commas and
// whitespace are accepted so .env values can be written as a compact list or
// as a space-separated value. GEMINI_MODEL remains a compatibility fallback
// and is tokenized with the same parser so a legacy value like
// "gemini-3.1-pro,gemini-2.5-flash" produces multiple ordered models rather
// than a single invalid ID.
func geminiModels(raw, legacy string) []string {
	split := func(s string) []string {
		return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r' })
	}
	models := make([]string, 0, len(split(raw))+1)
	seen := make(map[string]bool)
	for _, model := range split(raw) {
		model = strings.TrimSpace(model)
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		models = append(models, model)
	}
	// GEMINI_MODELS is empty: tokenize the legacy GEMINI_MODEL value
	// with the same parser so a comma-separated list is not treated
	// as a single invalid model ID (which caused 404→502).
	if len(models) == 0 {
		for _, model := range split(legacy) {
			model = strings.TrimSpace(model)
			if model == "" || seen[model] {
				continue
			}
			seen[model] = true
			models = append(models, model)
		}
	}
	if len(models) == 0 {
		models = append(models, "gemini-2.5-flash")
	}
	return models
}

func loadDotEnv() {
	paths := []string{".env", filepath.Join("backend", ".env")}
	for _, path := range paths {
		file, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			panic(fmt.Errorf("config: open %s: %w", path, err))
		}
		if err := readDotEnv(file); err != nil {
			_ = file.Close()
			panic(fmt.Errorf("config: read %s: %w", path, err))
		}
		if err := file.Close(); err != nil {
			panic(fmt.Errorf("config: close %s: %w", path, err))
		}
		return
	}
}

func readDotEnv(file io.Reader) error {
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		name, value, ok := strings.Cut(line, "=")
		if !ok || !validEnvName(name) {
			return fmt.Errorf("invalid .env entry")
		}
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if _, exists := os.LookupEnv(name); !exists {
			if err := os.Setenv(name, value); err != nil {
				return fmt.Errorf("set %s: %w", name, err)
			}
		}
	}
	return scanner.Err()
}

func validEnvName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for index, char := range name {
		isLetter := (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z')
		isDigit := char >= '0' && char <= '9'
		if !isLetter && !isDigit && char != '_' {
			return false
		}
		if index == 0 && isDigit {
			return false
		}
	}
	return true
}

// EffectiveDatabaseURL returns DATABASE_URL when set; otherwise it builds a
// postgres URL from the DATABASE_* parts so the user, password, host, port,
// and database name always come from the environment. Credentials are
// URL-escaped via net/url, so special characters in the password are safe.
func (c Config) EffectiveDatabaseURL() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	u := &url.URL{
		Scheme: "postgres",
		Host:   c.DatabaseHost + ":" + c.DatabasePort,
		Path:   "/" + c.DatabaseName,
	}
	switch {
	case c.DatabaseUser != "" && c.DatabasePassword != "":
		u.User = url.UserPassword(c.DatabaseUser, c.DatabasePassword)
	case c.DatabaseUser != "":
		u.User = url.User(c.DatabaseUser)
	}
	return u.String()
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
