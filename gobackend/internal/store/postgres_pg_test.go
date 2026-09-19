package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/migrate"
	"github.com/ai-uml-architect/gobackend/internal/pgtest"
	"github.com/ai-uml-architect/gobackend/internal/service"
	"github.com/ai-uml-architect/gobackend/internal/store"
)

// TestPostgresContract exercises the full service surface over PostgreSQL:
// opaque login, membership gating, CRUD, versions/restore, and the Instant
// wire format. Skips when PostgreSQL is unavailable.
func TestPostgresContract(t *testing.T) {
	conn := pgtest.Connect(t)
	ctx := context.Background()
	if err := migrate.Up(ctx, conn); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}
	pool, err := store.OpenPool(ctx, pgtest.TestDatabaseURL())
	if err != nil {
		t.Fatalf("open pool failed: %v", err)
	}
	defer pool.Close()
	svc := service.New(store.NewPostgres(pool))

	anaID := "11111111-1111-1111-1111-111111111111"
	brunoID := "22222222-2222-2222-2222-222222222222"
	projectA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	projectB := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	resp, err := svc.Login(ctx, "ANA@example.COM", "Password123!")
	if err != nil {
		t.Fatalf("seed login failed: %v", err)
	}
	if resp.UserID != anaID || resp.AccessToken == "" {
		t.Fatalf("unexpected login response: %+v", resp)
	}
	if _, err := svc.Login(ctx, "ana@example.com", "wrong"); err == nil {
		t.Fatalf("wrong password must fail")
	}

	assigned, err := svc.AssignedProjects(ctx, anaID)
	if err != nil {
		t.Fatalf("assigned failed: %v", err)
	}
	if len(assigned) != 2 {
		t.Fatalf("ana must see 2 projects, got %+v", assigned)
	}
	for _, p := range assigned {
		if p.ID == projectA && p.DiagramCount != 1 {
			t.Errorf("project A must report 1 persisted diagram, got %+v", p)
		}
	}

	if _, err := svc.ListDiagrams(ctx, projectB, brunoID); err == nil {
		t.Fatalf("bruno must be forbidden on project B")
	} else if _, ok := err.(service.ForbiddenError); !ok {
		t.Fatalf("expected ForbiddenError, got %T", err)
	}

	created, err := svc.CreateDiagram(ctx, projectB, "33333333-3333-3333-3333-333333333333",
		domain.DiagramDocument{SchemaVersion: 1, Name: "Lib", Classes: []domain.UmlClass{}, Relationships: []domain.Relationship{}})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	diagramID := *created.ID

	list, err := svc.ListDiagrams(ctx, projectB, "33333333-3333-3333-3333-333333333333")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 diagram, got %+v", list)
	}
	if _, err := time.Parse(time.RFC3339Nano, list[0].UpdatedAt); err != nil {
		t.Errorf("updatedAt must be Instant wire format, got %q (%v)", list[0].UpdatedAt, err)
	}

	updated := domain.DiagramDocument{SchemaVersion: 1, Name: "Lib v2", Classes: []domain.UmlClass{}, Relationships: []domain.Relationship{}}
	if _, err := svc.UpdateDiagram(ctx, projectB, diagramID, "33333333-3333-3333-3333-333333333333", updated); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	versions, err := svc.ListVersions(ctx, projectB, diagramID, "33333333-3333-3333-3333-333333333333")
	if err != nil || len(versions) != 2 || versions[0].VersionNumber != 2 {
		t.Fatalf("expected versions [2 1], got %+v %v", versions, err)
	}
	restored, err := svc.RestoreDiagram(ctx, projectB, diagramID, "33333333-3333-3333-3333-333333333333", 1)
	if err != nil || restored.Name != "Lib" {
		t.Fatalf("restore failed: %+v %v", restored, err)
	}
	versions, _ = svc.ListVersions(ctx, projectB, diagramID, "33333333-3333-3333-3333-333333333333")
	if len(versions) != 3 || versions[0].VersionNumber != 3 {
		t.Fatalf("restore must append version 3, got %+v", versions)
	}
}
