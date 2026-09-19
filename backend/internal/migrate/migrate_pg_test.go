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
	if journaled != 4 {
		t.Errorf("expected 4 journaled migrations, got %d", journaled)
	}

	checks := []struct {
		name  string
		query string
		want  int
	}{
		{name: "seed users (V1 demos + V3 credential user; V4 adds none)", query: `SELECT COUNT(*) FROM users`, want: 4},
		{name: "seed projects (V1 pair + V4 lonely-user bootstrap)", query: `SELECT COUNT(*) FROM projects`, want: 3},
		{name: "seed memberships (V1 set + V4 OWNER bootstrap)", query: `SELECT COUNT(*) FROM project_memberships`, want: 5},
		{name: "seed diagrams (V1 one + V4 starter)", query: `SELECT COUNT(*) FROM diagrams`, want: 2},
		{name: "seed versions (V1 v1 + V2 v2 + V4 v1)", query: `SELECT COUNT(*) FROM diagram_versions`, want: 3},
		{name: "V4 leaves no user without a membership", query: `SELECT COUNT(*) FROM users u WHERE NOT EXISTS (SELECT 1 FROM project_memberships pm WHERE pm.user_id = u.id)`, want: 0},
		{name: "diagram documents keep id equal to the row id", query: `SELECT COUNT(*) FROM diagrams WHERE document->>'id' != id::text`, want: 0},
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
	if versions != 3 {
		t.Errorf("rerun must not duplicate rows, got %d versions", versions)
	}
}
