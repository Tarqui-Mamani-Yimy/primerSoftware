package service_test

import (
	"context"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/service"
	"github.com/ai-uml-architect/gobackend/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const (
	anaID     = "11111111-1111-1111-1111-111111111111"
	brunoID   = "22222222-2222-2222-2222-222222222222"
	projectID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
)

func seedStore(t *testing.T) *store.MemoryStore {
	t.Helper()
	ms := store.NewMemoryStore()
	hash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	ms.SeedUser(store.User{ID: anaID, DisplayName: "Ana", Email: "ana@example.com", PasswordHash: string(hash)})
	ms.SeedUser(store.User{ID: brunoID, DisplayName: "Bruno", Email: "bruno@example.com", PasswordHash: string(hash)})
	desc := "sales"
	ms.SeedProject(projectID, "Sales", &desc)
	ms.SeedMember(projectID, anaID, "OWNER")
	return ms
}

func emptyDoc() domain.DiagramDocument {
	return domain.DiagramDocument{SchemaVersion: 1, Name: "Main", Classes: []domain.UmlClass{}, Relationships: []domain.Relationship{}}
}

func TestLoginSucceedsWithExactResponseKeys(t *testing.T) {
	svc := service.New(seedStore(t))
	resp, err := svc.Login(context.Background(), "ANA@example.com", "Password123!")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if resp.AccessToken == "" || resp.UserID != anaID || resp.DisplayName != "Ana" || resp.Email != "ana@example.com" {
		t.Errorf("unexpected login response: %+v", resp)
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	svc := service.New(seedStore(t))
	for _, tc := range []struct{ name, email, password string }{
		{name: "wrong password", email: "ana@example.com", password: "nope"},
		{name: "unknown email", email: "ghost@example.com", password: "Password123!"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Login(context.Background(), tc.email, tc.password); err == nil {
				t.Fatalf("expected error")
			} else if _, ok := err.(service.CredentialsError); !ok {
				t.Fatalf("expected CredentialsError, got %T (%v)", err, err)
			}
		})
	}
}

func TestAuthenticateChecksExpiryAndRevocation(t *testing.T) {
	ms := seedStore(t)
	svc := service.New(ms)
	resp, err := svc.Login(context.Background(), "ana@example.com", "Password123!")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	userID, err := svc.Authenticate(context.Background(), resp.AccessToken)
	if err != nil || userID != anaID {
		t.Fatalf("authenticate failed: userID=%q err=%v", userID, err)
	}
	if _, err := svc.Authenticate(context.Background(), "bogus"); err == nil {
		t.Errorf("unknown token must fail")
	}
}

func TestAssignedProjectsGateAndDiagramCount(t *testing.T) {
	ms := seedStore(t)
	svc := service.New(ms)
	got, err := svc.AssignedProjects(context.Background(), anaID)
	if err != nil {
		t.Fatalf("assigned failed: %v", err)
	}
	if len(got) != 1 || got[0].ID != projectID || got[0].Role != "OWNER" || got[0].DiagramCount != 0 {
		t.Fatalf("unexpected assigned: %+v", got)
	}
	got, err = svc.AssignedProjects(context.Background(), brunoID)
	if err != nil {
		t.Fatalf("assigned failed: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("non-member must see no projects, got %+v", got)
	}
}

func TestDiagramCRUDVersionsAndRestore(t *testing.T) {
	ctx := context.Background()
	svc := service.New(seedStore(t))

	created, err := svc.CreateDiagram(ctx, projectID, anaID, emptyDoc())
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.SchemaVersion != 1 || created.ID == nil || *created.ID == "" {
		t.Fatalf("create must assign schemaVersion 1 and an id: %+v", created)
	}
	diagramID := *created.ID

	got, err := svc.GetDiagram(ctx, projectID, diagramID, anaID)
	if err != nil || got.Name != "Main" {
		t.Fatalf("get failed: %+v %v", got, err)
	}

	list, err := svc.ListDiagrams(ctx, projectID, anaID)
	if err != nil || len(list) != 1 || list[0].ID != diagramID {
		t.Fatalf("list failed: %+v %v", list, err)
	}

	updated := emptyDoc()
	updated.Name = "Renamed"
	if _, err := svc.UpdateDiagram(ctx, projectID, diagramID, anaID, updated); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	versions, err := svc.ListVersions(ctx, projectID, diagramID, anaID)
	if err != nil {
		t.Fatalf("versions failed: %v", err)
	}
	if len(versions) != 2 || versions[0].VersionNumber != 2 || versions[1].VersionNumber != 1 {
		t.Fatalf("expected versions [2 1] desc, got %+v", versions)
	}

	restored, err := svc.RestoreDiagram(ctx, projectID, diagramID, anaID, 1)
	if err != nil || restored.Name != "Main" {
		t.Fatalf("restore failed: %+v %v", restored, err)
	}
	versions, _ = svc.ListVersions(ctx, projectID, diagramID, anaID)
	if len(versions) != 3 || versions[0].VersionNumber != 3 {
		t.Fatalf("restore must append version 3, got %+v", versions)
	}
}

func TestDiagramAccessDeniedAndNotFound(t *testing.T) {
	ctx := context.Background()
	svc := service.New(seedStore(t))

	if _, err := svc.CreateDiagram(ctx, projectID, brunoID, emptyDoc()); err == nil {
		t.Fatalf("non-member create must fail")
	} else if _, ok := err.(service.ForbiddenError); !ok {
		t.Fatalf("expected ForbiddenError, got %T", err)
	}
	if _, err := svc.ListDiagrams(ctx, projectID, brunoID); err == nil {
		t.Fatalf("non-member list must fail")
	}
	if _, err := svc.GetDiagram(ctx, projectID, "00000000-0000-0000-0000-000000000000", anaID); err == nil {
		t.Fatalf("missing diagram must fail")
	} else if nf, ok := err.(service.NotFoundError); !ok || nf.Message != "Diagram not found" {
		t.Fatalf("expected NotFoundError(Diagram not found), got %#v", err)
	}
	if _, err := svc.RestoreDiagram(ctx, projectID, "00000000-0000-0000-0000-000000000000", anaID, 1); err == nil {
		t.Fatalf("restore of missing diagram must fail")
	}
}

func TestDiagramValidationFailuresAreBadRequest(t *testing.T) {
	ctx := context.Background()
	svc := service.New(seedStore(t))

	bad := emptyDoc()
	bad.Name = "   "
	if _, err := svc.CreateDiagram(ctx, projectID, anaID, bad); err == nil {
		t.Fatalf("blank name must fail")
	} else if _, ok := err.(service.ValidationError); !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	dup := emptyDoc()
	dup.Classes = []domain.UmlClass{{ID: "c1", Name: "A"}, {ID: "c1", Name: "B"}}
	if _, err := svc.CreateDiagram(ctx, projectID, anaID, dup); err == nil {
		t.Fatalf("duplicate class ids must fail")
	}
}
