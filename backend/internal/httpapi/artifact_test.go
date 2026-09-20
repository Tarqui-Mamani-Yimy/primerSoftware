package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/httpapi"
	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
	"github.com/ai-uml-architect/gobackend/internal/jhipster"
	"github.com/ai-uml-architect/gobackend/internal/service"
	"github.com/ai-uml-architect/gobackend/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const (
	artifactDiagramID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	unknownDiagramID  = "cccccccc-cccc-cccc-cccc-cccccccccccc"
)

// fakeArtifactGen implements service.ArtifactGenerator: it records the last
// document/options and returns a fixed ZIP unless errored.
type fakeArtifactGen struct {
	doc     domain.DiagramDocument
	opts    jdlgen.Options
	calls   int
	err     error
	content []byte
}

func (f *fakeArtifactGen) Generate(_ context.Context, doc domain.DiagramDocument, o jdlgen.Options) (jhipster.Result, error) {
	f.calls++
	f.doc = doc
	f.opts = o
	if f.err != nil {
		return jhipster.Result{}, f.err
	}
	return jhipster.Result{FileName: o.BaseName + "-jhipster-backend.zip", Content: f.content}, nil
}

func validArtifactDocument() domain.DiagramDocument {
	return domain.DiagramDocument{
		SchemaVersion: 1,
		Version:       3,
		Name:          "Shop",
		Classes:       []domain.UmlClass{{ID: "a", Name: "Alpha"}},
		Relationships: []domain.Relationship{},
	}
}

func seedArtifactDiagram(ms *store.MemoryStore) {
	ms.SeedDiagramWithReview(store.DiagramRecord{ID: artifactDiagramID, ProjectID: projectID, Name: "Shop", CreatedBy: anaID, ReviewNumber: 1})
}

func artifactServer(t *testing.T, gen service.ArtifactGenerator) *httpapi.Server {
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
	seedArtifactDiagram(ms)
	return httpapi.NewServer(service.NewWithJhipster(ms, gen), "http://localhost:3000", nil, nil)
}

func artifactPath(diagramID string) string {
	return fmt.Sprintf("/api/v1/projects/%s/diagrams/%s/artifact", projectID, diagramID)
}

func artifactBody(t *testing.T, doc domain.DiagramDocument, cfg map[string]any) *bytes.Reader {
	t.Helper()
	payload := map[string]any{"document": doc}
	if cfg != nil {
		payload["config"] = cfg
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return bytes.NewReader(raw)
}

func TestArtifactEndpointGeneratesZip(t *testing.T) {
	gen := &fakeArtifactGen{content: []byte("PK\x03\x04fakezip")}
	srv := artifactServer(t, gen)
	token := login(t, srv, "ana@example.com", "Password123!")

	rec := doAuthed(t, srv, http.MethodPost, artifactPath(artifactDiagramID), artifactBody(t, validArtifactDocument(), nil), token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename="UmlArchitect-jhipster-backend.zip"` {
		t.Errorf("Content-Disposition = %q", cd)
	}
	if rec.Body.String() != "PK\x03\x04fakezip" {
		t.Errorf("body does not match generator output")
	}
	if gen.calls != 1 {
		t.Fatalf("generator calls = %d, want 1", gen.calls)
	}
	if gen.opts != jdlgen.DefaultOptions() {
		t.Errorf("options = %+v, want defaults %+v", gen.opts, jdlgen.DefaultOptions())
	}
	if gen.doc.Name != "Shop" || len(gen.doc.Classes) != 1 {
		t.Errorf("document not passed through: %+v", gen.doc)
	}
}

func TestArtifactEndpointConfigOverride(t *testing.T) {
	gen := &fakeArtifactGen{content: []byte("PKfake")}
	srv := artifactServer(t, gen)
	token := login(t, srv, "ana@example.com", "Password123!")

	cfg := map[string]any{"baseName": "Shop", "packageName": "com.example.shop", "buildTool": "gradle"}
	rec := doAuthed(t, srv, http.MethodPost, artifactPath(artifactDiagramID), artifactBody(t, validArtifactDocument(), cfg), token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename="Shop-jhipster-backend.zip"` {
		t.Errorf("Content-Disposition = %q", cd)
	}
	want := jdlgen.Options{BaseName: "Shop", PackageName: "com.example.shop", BuildTool: "gradle", AuthenticationType: "jwt"}
	if gen.opts != want {
		t.Errorf("options = %+v, want %+v", gen.opts, want)
	}
}

func TestArtifactEndpointErrors(t *testing.T) {
	t.Run("missing token => 401", func(t *testing.T) {
		srv := artifactServer(t, &fakeArtifactGen{})
		rec := doAuthed(t, srv, http.MethodPost, artifactPath(artifactDiagramID), artifactBody(t, validArtifactDocument(), nil), "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("malformed body => 400", func(t *testing.T) {
		srv := artifactServer(t, &fakeArtifactGen{})
		token := login(t, srv, "ana@example.com", "Password123!")
		rec := doAuthed(t, srv, http.MethodPost, artifactPath(artifactDiagramID), bytes.NewReader([]byte("{not json")), token)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid document => 400", func(t *testing.T) {
		srv := artifactServer(t, &fakeArtifactGen{})
		token := login(t, srv, "ana@example.com", "Password123!")
		bad := validArtifactDocument()
		bad.Classes[0].Name = ""
		rec := doAuthed(t, srv, http.MethodPost, artifactPath(artifactDiagramID), artifactBody(t, bad, nil), token)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "name") {
			t.Errorf("error body should mention the invalid name: %s", rec.Body.String())
		}
	})

	t.Run("invalid config => 400", func(t *testing.T) {
		srv := artifactServer(t, &fakeArtifactGen{})
		token := login(t, srv, "ana@example.com", "Password123!")
		cfg := map[string]any{"buildTool": "npm"}
		rec := doAuthed(t, srv, http.MethodPost, artifactPath(artifactDiagramID), artifactBody(t, validArtifactDocument(), cfg), token)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "buildTool") {
			t.Errorf("error body should mention buildTool: %s", rec.Body.String())
		}
	})

	t.Run("unknown diagram => 404", func(t *testing.T) {
		srv := artifactServer(t, &fakeArtifactGen{})
		token := login(t, srv, "ana@example.com", "Password123!")
		rec := doAuthed(t, srv, http.MethodPost, artifactPath(unknownDiagramID), artifactBody(t, validArtifactDocument(), nil), token)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("non-member => 403", func(t *testing.T) {
		srv := artifactServer(t, &fakeArtifactGen{})
		token := login(t, srv, "bruno@example.com", "Password123!")
		rec := doAuthed(t, srv, http.MethodPost, artifactPath(artifactDiagramID), artifactBody(t, validArtifactDocument(), nil), token)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("generator failure => 502", func(t *testing.T) {
		gen := &fakeArtifactGen{err: service.GenerationError{Message: "JHipster generation failed: boom"}}
		srv := artifactServer(t, gen)
		token := login(t, srv, "ana@example.com", "Password123!")
		rec := doAuthed(t, srv, http.MethodPost, artifactPath(artifactDiagramID), artifactBody(t, validArtifactDocument(), nil), token)
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "boom") {
			t.Errorf("error body should carry the sanitized message: %s", rec.Body.String())
		}
	})

	t.Run("generator not configured => 502", func(t *testing.T) {
		// A Service without a generator must not fake a ZIP: it reports that
		// generation is not configured instead.
		ms := store.NewMemoryStore()
		hash, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
		ms.SeedUser(store.User{ID: anaID, DisplayName: "Ana", Email: "ana@example.com", PasswordHash: string(hash)})
		desc := "sales"
		ms.SeedProject(projectID, "Sales", &desc)
		ms.SeedMember(projectID, anaID, "OWNER")
		seedArtifactDiagram(ms)
		srv := httpapi.NewServer(service.New(ms), "http://localhost:3000", nil, nil)
		token := login(t, srv, "ana@example.com", "Password123!")
		rec := doAuthed(t, srv, http.MethodPost, artifactPath(artifactDiagramID), artifactBody(t, validArtifactDocument(), nil), token)
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "not configured") {
			t.Errorf("error body should say generation is not configured: %s", rec.Body.String())
		}
	})
}

func TestArtifactEndpointGenerationTimeoutIs502(t *testing.T) {
	// The handler runs generation under a bounded context; when the budget is
	// exhausted the generator error (context deadline exceeded) must surface as
	// a 502 with the bounded message, never a hang or a fabricated ZIP.
	gen := &fakeArtifactGen{err: fmt.Errorf("JHipster generation failed: context deadline exceeded")}
	srv := artifactServer(t, gen)
	token := login(t, srv, "ana@example.com", "Password123!")
	rec := doAuthed(t, srv, http.MethodPost, artifactPath(artifactDiagramID), artifactBody(t, validArtifactDocument(), nil), token)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "deadline") {
		t.Errorf("body should mention the timeout: %s", rec.Body.String())
	}
}
