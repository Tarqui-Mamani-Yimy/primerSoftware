package migrate_test

import (
	"context"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/migrate"
	"github.com/ai-uml-architect/gobackend/internal/pgtest"
)

// TestMigrateUpAgainstPostgres applies the embedded chain to a fresh database
// and verifies schema, seeds, and V2 conditional semantics. Skips when
// PostgreSQL is unavailable.
func TestMigrateUpAgainstPostgres(t *testing.T) {
	conn := pgtest.Connect(t)
	ctx := context.Background()

	if err := migrate.Up(ctx, conn); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	var journaled int
	if err := conn.QueryRow(ctx, `SELECT COUNT(*) FROM go_schema_migrations`).Scan(&journaled); err != nil {
		t.Fatalf("journal missing: %v", err)
	}
	if journaled != 3 {
		t.Errorf("expected 3 journaled migrations, got %d", journaled)
	}

	checks := []struct {
		name  string
		query string
		want  int
	}{
		{name: "seed users (V1 demos + V3 credential user)", query: `SELECT COUNT(*) FROM users`, want: 4},
		{name: "seed projects", query: `SELECT COUNT(*) FROM projects`, want: 2},
		{name: "seed memberships", query: `SELECT COUNT(*) FROM project_memberships`, want: 4},
		{name: "seed diagrams", query: `SELECT COUNT(*) FROM diagrams`, want: 1},
		{name: "seed versions (V1 v1 + V2 v2)", query: `SELECT COUNT(*) FROM diagram_versions`, want: 2},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			var got int
			if err := conn.QueryRow(ctx, tc.query).Scan(&got); err != nil {
				t.Fatalf("query failed: %v", err)
			}
			if got != tc.want {
				t.Errorf("expected %d, got %d", tc.want, got)
			}
		})
	}

	var classCount int
	err := conn.QueryRow(ctx, `SELECT jsonb_array_length(document->'classes')
  FROM diagrams WHERE id = 'cccccccc-cccc-cccc-cccc-cccccccccccc'`).Scan(&classCount)
	if err != nil {
		t.Fatalf("seed diagram unreadable: %v", err)
	}
	if classCount != 6 {
		t.Errorf("V2 conditional seed must populate 6 classes, got %d", classCount)
	}

	// Reruns are idempotent: journaled versions are skipped, no duplicate rows.
	if err := migrate.Up(ctx, conn); err != nil {
		t.Fatalf("second migrate up failed: %v", err)
	}
	var versions int
	if err := conn.QueryRow(ctx, `SELECT COUNT(*) FROM diagram_versions`).Scan(&versions); err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if versions != 2 {
		t.Errorf("rerun must not duplicate rows, got %d versions", versions)
	}
}
