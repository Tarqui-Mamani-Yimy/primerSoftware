// Package migrate embeds the Go-owned migration chain and applies it in
// Flyway-compatible version order.
//
// The embedded SQL replicates backend/src/main/resources/db/migration/V1__
// (sole schema authority: tables, indexes, dev seeds) and V2__ (conditional
// seed semantics) without reintroducing hibernate ddl-auto: the schema only
// ever changes through these versioned files. Applied versions are journaled
// in go_schema_migrations so each file runs exactly once.
package migrate

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/*.sql
var files embed.FS

// Migration is one versioned SQL file.
type Migration struct {
	Version string
	Name    string
	SQL     string
}

// Ordered returns the embedded migrations sorted by numeric version,
// mirroring Flyway's V1__ then V2__ application order.
func Ordered() []Migration {
	entries, err := files.ReadDir("migrations")
	if err != nil {
		panic(fmt.Sprintf("migrate: read embedded migrations: %v", err))
	}
	var out []Migration
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "V") || !strings.HasSuffix(name, ".sql") {
			continue
		}
		version := strings.TrimPrefix(name, "V")
		if i := strings.Index(version, "__"); i >= 0 {
			version = version[:i]
		}
		raw, err := files.ReadFile("migrations/" + name)
		if err != nil {
			panic(fmt.Sprintf("migrate: read %s: %v", name, err))
		}
		out = append(out, Migration{Version: version, Name: name, SQL: string(raw)})
	}
	sort.Slice(out, func(i, j int) bool {
		vi, ei := strconv.Atoi(out[i].Version)
		vj, ej := strconv.Atoi(out[j].Version)
		if ei == nil && ej == nil && vi != vj {
			return vi < vj
		}
		return out[i].Version < out[j].Version
	})
	return out
}

// Up applies every pending migration in order, journaling each in
// go_schema_migrations inside the same transaction as its SQL.
func Up(ctx context.Context, conn *pgx.Conn) error {
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS go_schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`); err != nil {
		return fmt.Errorf("migrate: create journal: %w", err)
	}
	for _, m := range Ordered() {
		var applied string
		err := conn.QueryRow(ctx, `SELECT version FROM go_schema_migrations WHERE version = $1`, m.Version).Scan(&applied)
		if err == nil {
			continue
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("migrate %s: begin: %w", m.Name, err)
		}
		if _, err := tx.Exec(ctx, m.SQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate %s: %w", m.Name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO go_schema_migrations (version) VALUES ($1)`, m.Version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate %s: journal: %w", m.Name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("migrate %s: commit: %w", m.Name, err)
		}
	}
	return nil
}
