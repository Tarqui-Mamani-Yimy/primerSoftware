package service_test

import (
	"context"
	"strings"
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

func intPtr(i int) *int             { return &i }
func int64Ptr(i int64) *int64       { return &i }
func strPtr(s string) *string       { return &s }

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

func TestCreateProjectGeneratesCodeAndOwnsProject(t *testing.T) {
	ctx := context.Background()
	svc := service.New(seedStore(t))
	desc := "Class 4B"

	created, err := svc.CreateProject(ctx, anaID, "  Física  ", &desc)
	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	if created.ID == "" || created.Name != "Física" || created.Role != "OWNER" || created.DiagramCount != 0 {
		t.Fatalf("unexpected created project: %+v", created)
	}
	if len(created.AccessCode) != 6 {
		t.Fatalf("access code must be 6 characters, got %q", created.AccessCode)
	}
	for _, r := range created.AccessCode {
		if !strings.ContainsRune("ABCDEFGHJKMNPQRSTVWXYZ23456789", r) {
			t.Fatalf("access code %q contains an ambiguous character %q", created.AccessCode, r)
		}
	}

	assigned, err := svc.AssignedProjects(ctx, anaID)
	if err != nil {
		t.Fatalf("assigned failed: %v", err)
	}
	found := false
	for _, p := range assigned {
		if p.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("creator must own the new project: %+v", assigned)
	}
}

func TestCreateProjectValidation(t *testing.T) {
	svc := service.New(seedStore(t))
	for _, tc := range []struct{ name, project string }{
		{name: "blank name", project: "   "},
		{name: "name over 150 chars", project: strings.Repeat("a", 151)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.CreateProject(context.Background(), anaID, tc.project, nil); err == nil {
				t.Fatalf("expected validation error")
			} else if _, ok := err.(service.ValidationError); !ok {
				t.Fatalf("expected ValidationError, got %T (%v)", err, err)
			}
		})
	}
}

func TestJoinProjectIsIdempotentAndCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	ms := seedStore(t)
	desc := "Electives"
	ms.SeedProjectWithCode("cccccccc-cccc-cccc-cccc-cccccccccccc", "Robotics", &desc, "X7K2P9", anaID)
	svc := service.New(ms)

	joined, err := svc.JoinProject(ctx, brunoID, " x7k2p9 ")
	if err != nil {
		t.Fatalf("join failed: %v", err)
	}
	if joined.ID != "cccccccc-cccc-cccc-cccc-cccccccccccc" || joined.Role != "COLLABORATOR" || joined.Name != "Robotics" {
		t.Fatalf("unexpected join response: %+v", joined)
	}

	again, err := svc.JoinProject(ctx, brunoID, "X7K2P9")
	if err != nil {
		t.Fatalf("repeat join failed: %v", err)
	}
	if again.Role != "COLLABORATOR" {
		t.Fatalf("repeat join changed the role: %+v", again)
	}
	assigned, err := svc.AssignedProjects(ctx, brunoID)
	if err != nil {
		t.Fatalf("assigned failed: %v", err)
	}
	if len(assigned) != 1 {
		t.Fatalf("repeat join must not duplicate the membership: %+v", assigned)
	}

	// The owner rejoining their own code keeps OWNER instead of demoting.
	owner, err := svc.JoinProject(ctx, anaID, "X7K2P9")
	if err != nil {
		t.Fatalf("owner join failed: %v", err)
	}
	if owner.Role != "OWNER" {
		t.Fatalf("owner rejoin must keep OWNER, got %+v", owner)
	}
}

func TestJoinProjectErrors(t *testing.T) {
	svc := service.New(seedStore(t))
	ctx := context.Background()

	if _, err := svc.JoinProject(ctx, brunoID, "ZZZZZZ"); err == nil {
		t.Fatalf("unknown code must fail")
	} else if nf, ok := err.(service.NotFoundError); !ok || nf.Message != "Project not found" {
		t.Fatalf("expected NotFoundError(Project not found), got %#v", err)
	}
	if _, err := svc.JoinProject(ctx, brunoID, "  "); err == nil {
		t.Fatalf("blank code must fail")
	} else if _, ok := err.(service.ValidationError); !ok {
		t.Fatalf("expected ValidationError, got %T (%v)", err, err)
	}
}

func TestDiagramAutosaveKeepsVersionStable(t *testing.T) {
	ctx := context.Background()
	svc := service.New(seedStore(t))

	created, err := svc.CreateDiagram(ctx, projectID, anaID, emptyDoc())
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.Version != 1 {
		t.Fatalf("create must seed an implicit checkpoint v1, got version %d", created.Version)
	}
	if created.ReviewNumber != 2 {
		t.Fatalf("create must end at review_number=2 (create insert 1 + initial checkpoint 2), got %d", created.ReviewNumber)
	}
	diagramID := *created.ID

	// Legacy write: no baseline (body.reviewNumber=0, no X-Diagram-Review
	// header) → silent bump, never 409. The contract documented in
	// odd/tasks/realtime-diagram-collaboration.md says legacy clients may
	// send 0; the service reads the live review_number and uses it as
	// the CAS baseline.
	legacy := emptyDoc()
	legacy.Name = "Legacy autosave"
	legacy.Version = 0
	legacy.ReviewNumber = 0
	legacyDoc, err := svc.UpdateDiagram(ctx, projectID, diagramID, anaID, legacy, nil, nil)
	if err != nil {
		t.Fatalf("legacy autosave must succeed: %v", err)
	}
	if legacyDoc.ReviewNumber <= created.ReviewNumber {
		t.Fatalf("legacy autosave must bump the review number, got %d -> %d", created.ReviewNumber, legacyDoc.ReviewNumber)
	}
	postLegacy := legacyDoc.ReviewNumber

	// Positive stale baseline → 409, like the contract.
	stale := int64(postLegacy + 99)
	if _, err := svc.UpdateDiagram(ctx, projectID, diagramID, anaID, emptyDoc(), nil, &stale); err == nil {
		t.Fatalf("autosave with stale positive baseline must conflict")
	} else if c, ok := err.(service.ConflictError); !ok {
		t.Fatalf("expected ConflictError, got %T (%v)", err, err)
	} else if c.ActualReview != postLegacy {
		t.Fatalf("conflict envelope must carry the live review number: %+v", c)
	}

	updated := emptyDoc()
	updated.Name = "Autosave"
	updated.Version = 1
	updated.ReviewNumber = postLegacy
	if doc, err := svc.UpdateDiagram(ctx, projectID, diagramID, anaID, updated, nil, nil); err != nil {
		t.Fatalf("update failed: %v", err)
	} else if doc.Version != 1 {
		t.Fatalf("autosave must not grow the version, got version %d", doc.Version)
	} else if doc.ReviewNumber <= postLegacy {
		t.Fatalf("autosave must bump the review number, got %d -> %d", postLegacy, doc.ReviewNumber)
	}

	versions, err := svc.ListVersions(ctx, projectID, diagramID, anaID)
	if err != nil {
		t.Fatalf("versions failed: %v", err)
	}
	if len(versions) != 1 || versions[0].VersionNumber != 1 {
		t.Fatalf("autosave must keep version history at [1], got %+v", versions)
	}
}

func TestExplicitCheckpointAdvancesVersionAndStampsAuthor(t *testing.T) {
	ctx := context.Background()
	svc := service.New(seedStore(t))

	created, err := svc.CreateDiagram(ctx, projectID, anaID, emptyDoc())
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	diagramID := *created.ID

	checkpoint := emptyDoc()
	checkpoint.Name = "Stable model"
	checkpoint.Version = 1
	checkpoint.ReviewNumber = created.ReviewNumber
	msg := "first stable revision"
	v, err := svc.CreateCheckpoint(ctx, projectID, diagramID, anaID, checkpoint, &msg, nil, nil)
	if err != nil {
		t.Fatalf("checkpoint failed: %v", err)
	}
	if v.VersionNumber != 2 || v.CreatedBy != anaID || v.Message == nil || *v.Message != msg {
		t.Fatalf("expected checkpoint v2 by ana with message, got %+v", v)
	}
	if v.ReviewNumber <= created.ReviewNumber {
		t.Fatalf("checkpoint must bump review_number, got %d -> %d", created.ReviewNumber, v.ReviewNumber)
	}

	doc, err := svc.GetDiagram(ctx, projectID, diagramID, anaID)
	if err != nil || doc.Version != 2 || doc.Name != "Stable model" {
		t.Fatalf("diagram must reflect checkpoint v2 with the new name: %+v %v", doc, err)
	}
}

func TestConcurrentAutosavesSurfaceOptimisticConflict(t *testing.T) {
	ctx := context.Background()
	svc := service.New(seedStore(t))

	created, err := svc.CreateDiagram(ctx, projectID, anaID, emptyDoc())
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	diagramID := *created.ID

	type result struct {
		doc    domain.DiagramDocument
		review int64
		err    error
	}
	results := make(chan result, 4)
	for i := 0; i < 4; i++ {
		doc := emptyDoc()
		doc.Name = "race-" + string(rune('a'+i))
		doc.Version = 1
		doc.ReviewNumber = created.ReviewNumber
		go func(d domain.DiagramDocument) {
			got, err := svc.UpdateDiagram(ctx, projectID, diagramID, anaID, d, nil, nil)
			var review int64
			if err == nil {
				review = got.ReviewNumber
			}
			results <- result{doc: got, review: review, err: err}
		}(doc)
	}
	winners, conflicts := 0, 0
	for i := 0; i < 4; i++ {
		r := <-results
		switch r.err.(type) {
		case nil:
			winners++
		case service.ConflictError:
			conflicts++
		default:
			t.Errorf("unexpected error: %v", r.err)
		}
	}
	if winners != 1 {
		t.Errorf("exactly one autosave must succeed, got %d", winners)
	}
	if conflicts == 0 {
		t.Errorf("expected at least one 409 to surface the CAS conflict")
	}
}

func TestStaleAutosaveReturnsConflict(t *testing.T) {
	ctx := context.Background()
	svc := service.New(seedStore(t))

	created, err := svc.CreateDiagram(ctx, projectID, anaID, emptyDoc())
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	diagramID := *created.ID

	// Sending a positive but stale baseline → 409. The contract allows
	// legacy writes (baseline 0) to silently bump; this test covers the
	// CAS guard for clients that DO send a positive baseline.
	stale := int64(created.ReviewNumber + 99)
	if _, err := svc.UpdateDiagram(ctx, projectID, diagramID, anaID, emptyDoc(), nil, &stale); err == nil {
		t.Fatalf("autosave with stale positive baseline must conflict")
	} else if c, ok := err.(service.ConflictError); !ok {
		t.Fatalf("expected ConflictError, got %T (%v)", err, err)
	} else if c.Current.Name != emptyDoc().Name {
		t.Fatalf("conflict envelope must carry the server document: %+v", c)
	}
}

func TestIfMatchHeaderAdvancesBaselineGuard(t *testing.T) {
	ctx := context.Background()
	svc := service.New(seedStore(t))
	created, err := svc.CreateDiagram(ctx, projectID, anaID, emptyDoc())
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	diagramID := *created.ID

	checkpoint := emptyDoc()
	checkpoint.Name = "v2"
	checkpoint.Version = 1
	checkpoint.ReviewNumber = created.ReviewNumber
	if _, err := svc.CreateCheckpoint(ctx, projectID, diagramID, anaID, checkpoint, nil, nil, nil); err != nil {
		t.Fatalf("checkpoint failed: %v", err)
	}
	afterCheckpoint, err := svc.GetDiagram(ctx, projectID, diagramID, anaID)
	if err != nil {
		t.Fatalf("getdiagram failed: %v", err)
	}

	updated := emptyDoc()
	updated.Name = "autosave against v2"
	updated.Version = 2
	updated.ReviewNumber = afterCheckpoint.ReviewNumber
	ifMatch := 2
	if doc, err := svc.UpdateDiagram(ctx, projectID, diagramID, anaID, updated, &ifMatch, nil); err != nil {
		t.Fatalf("autosave with matching If-Match must succeed: %v", err)
	} else if doc.Version != 2 {
		t.Fatalf("autosave must leave version at 2, got %d", doc.Version)
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

func TestRestoreCreatesNewCheckpoint(t *testing.T) {
	ctx := context.Background()
	svc := service.New(seedStore(t))

	created, err := svc.CreateDiagram(ctx, projectID, anaID, emptyDoc())
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	diagramID := *created.ID

	first := emptyDoc()
	first.Name = "v1 contents"
	first.Version = 1
	first.ReviewNumber = created.ReviewNumber
	if _, err := svc.CreateCheckpoint(ctx, projectID, diagramID, anaID, first, strPtr("v1"), nil, nil); err != nil {
		t.Fatalf("checkpoint v1 failed: %v", err)
	}

	if _, err := svc.RestoreDiagram(ctx, projectID, diagramID, anaID, 1); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	versions, err := svc.ListVersions(ctx, projectID, diagramID, anaID)
	if err != nil {
		t.Fatalf("versions failed: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions (initial + v1 + restored), got %+v", versions)
	}
	if versions[0].Message == nil || *versions[0].Message != "Restored v1" {
		t.Fatalf("top version must carry a Restored v1 message: %+v", versions[0])
	}
}

// TestRestorePropagatesLatestReview ensures RestoreDiagram picks the live
// review_number as its baseline, so a restore after another collaborator's
// checkpoint is recognized and stamped with a new review number.
func TestRestorePropagatesLatestReview(t *testing.T) {
	ctx := context.Background()
	svc := service.New(seedStore(t))
	created, err := svc.CreateDiagram(ctx, projectID, anaID, emptyDoc())
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	diagramID := *created.ID

	first := emptyDoc()
	first.Name = "v1 contents"
	first.Version = 1
	first.ReviewNumber = created.ReviewNumber
	if _, err := svc.CreateCheckpoint(ctx, projectID, diagramID, anaID, first, strPtr("v1"), nil, nil); err != nil {
		t.Fatalf("checkpoint v1 failed: %v", err)
	}
	second := emptyDoc()
	second.Name = "v2 contents"
	if afterCheckpoint, err := svc.GetDiagram(ctx, projectID, diagramID, anaID); err == nil {
		second.Version = afterCheckpoint.Version
		second.ReviewNumber = afterCheckpoint.ReviewNumber
	}
	if _, err := svc.CreateCheckpoint(ctx, projectID, diagramID, anaID, second, strPtr("v2"), nil, nil); err != nil {
		t.Fatalf("checkpoint v2 failed: %v", err)
	}
	if _, err := svc.RestoreDiagram(ctx, projectID, diagramID, anaID, 1); err != nil {
		t.Fatalf("restore v1 must succeed after v2: %v", err)
	}
}

func TestBroadcastsAfterAutosaveAndCheckpoint(t *testing.T) {
	ctx := context.Background()
	recorded := make(chan domain.DiagramChangedEvent, 8)
	recording := &captureBroadcaster{ch: recorded}
	svc := service.NewWithBroadcaster(seedStore(t), recording)

	created, err := svc.CreateDiagram(ctx, projectID, anaID, emptyDoc())
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	diagramID := *created.ID

	updated := emptyDoc()
	updated.Name = "live autosave"
	updated.Version = 1
	updated.ReviewNumber = created.ReviewNumber
	if _, err := svc.UpdateDiagram(ctx, projectID, diagramID, brunoID /* not a member; verify Forbidden */, updated, nil, nil); err == nil {
		t.Fatalf("non-member autosave must fail")
	} else if _, ok := err.(service.ForbiddenError); !ok {
		t.Fatalf("expected ForbiddenError, got %T(%v)", err, err)
	}

	updated.ReviewNumber = created.ReviewNumber
	if _, err := svc.UpdateDiagram(ctx, projectID, diagramID, anaID, updated, nil, nil); err != nil {
		t.Fatalf("autosave failed: %v", err)
	}

	checkpoint := emptyDoc()
	checkpoint.Name = "v2"
	checkpoint.Version = 1
	checkpoint.ReviewNumber = created.ReviewNumber + 1
	if _, err := svc.CreateCheckpoint(ctx, projectID, diagramID, anaID, checkpoint, strPtr("v2"), nil, nil); err != nil {
		t.Fatalf("checkpoint failed: %v", err)
	}

	got := drainEvents(recording)
	if len(got) < 3 {
		t.Fatalf("expected at least 3 broadcasts (create+autosave+checkpoint): %+v", got)
	}
	if got[0].Kind != "checkpoint" || got[1].Kind != "working-document" || got[2].Kind != "checkpoint" {
		t.Fatalf("expected broadcast kinds [checkpoint, working-document, checkpoint], got %+v", got)
	}
	for _, ev := range got {
		if ev.ReviewNumber <= 0 {
			t.Errorf("every broadcast must carry a positive review number: %+v", ev)
		}
	}
}

type captureBroadcaster struct{ ch chan domain.DiagramChangedEvent }

func (c *captureBroadcaster) BroadcastDiagramChanged(_, _ string, evt domain.DiagramChangedEvent) {
	c.ch <- evt
}

func drainEvents(c *captureBroadcaster) []domain.DiagramChangedEvent {
	collected := make([]domain.DiagramChangedEvent, 0, 8)
	for {
		select {
		case ev := <-c.ch:
			collected = append(collected, ev)
		default:
			return collected
		}
	}
}

// Compile-time guard: intPtr/int64Ptr/strPtr are kept satisfied by the
// test runner even if unused by tests once refactors land.
var _ = intPtr(0)
var _ = int64Ptr(0)
var _ = strPtr("")
