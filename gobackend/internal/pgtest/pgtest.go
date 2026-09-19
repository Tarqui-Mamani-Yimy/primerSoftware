// Package pgtest provisions an isolated PostgreSQL database for integration
// tests. When no server is reachable with the configured credentials the
// helpers Skip the test instead of failing, per the GOBE-02 contract:
// migration/integration tests run against PostgreSQL when available.
package pgtest

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// testDBName is the isolated database recreated for every integration run. It
// never touches the development uml_architect database.
const testDBName = "uml_architect_gobe02_test"

// AdminURL returns the maintenance-database URL used to create/drop the test
// database. Override with TEST_DATABASE_URL.
func AdminURL() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
}

// TestDatabaseURL returns the URL of the isolated test database.
func TestDatabaseURL() string {
	return replaceDBName(AdminURL(), testDBName)
}

func replaceDBName(raw, name string) string {
	// admin URLs have the form scheme://[user[:pass]@]host[:port]/dbname?query
	before, query, _ := cut(raw, "?")
	i := lastSlash(before)
	base := before[:i+1] + name
	if query != "" {
		return base + "?" + query
	}
	return base
}

func cut(s, sep string) (before, after string, found bool) {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return s, "", false
}

func lastSlash(s string) int {
	// Skip the scheme prefix ("scheme://").
	rest := s
	prefix := 0
	if _, after, found := cut(s, "://"); found {
		prefix = len(s) - len(after)
		rest = after
	}
	last := -1
	for i := 0; i < len(rest); i++ {
		if rest[i] == '/' {
			last = i
		}
	}
	if last < 0 {
		return len(s)
	}
	return prefix + last
}

// Connect recreates the isolated test database and returns a connection to
// it. It Skips when PostgreSQL is unavailable (never fails).
func Connect(t *testing.T) *pgx.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, AdminURL())
	if err != nil {
		t.Skipf("pg_status: unavailable (%v); set TEST_DATABASE_URL for a reachable server", err)
	}
	defer admin.Close(context.Background())

	// Terminate stray backends so DROP succeeds on reruns, then recreate.
	_, _ = admin.Exec(context.Background(),
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, testDBName)
	if _, err := admin.Exec(context.Background(),
		fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, pgx.Identifier{testDBName}.Sanitize())); err != nil {
		t.Skipf("pg_status: unavailable (cannot drop test db: %v)", err)
	}
	if _, err := admin.Exec(context.Background(),
		fmt.Sprintf(`CREATE DATABASE %s`, pgx.Identifier{testDBName}.Sanitize())); err != nil {
		t.Skipf("pg_status: unavailable (cannot create test db: %v)", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		admin, err := pgx.Connect(ctx, AdminURL())
		if err != nil {
			return
		}
		defer admin.Close(ctx)
		_, _ = admin.Exec(ctx,
			`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, testDBName)
		_, _ = admin.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, pgx.Identifier{testDBName}.Sanitize()))
	})

	conn, err := pgx.Connect(context.Background(), TestDatabaseURL())
	if err != nil {
		t.Fatalf("pg test database unreachable after create: %v", err)
	}
	t.Cleanup(func() { conn.Close(context.Background()) })
	return conn
}
