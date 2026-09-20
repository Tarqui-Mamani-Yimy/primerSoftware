package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/httpapi"
	"github.com/ai-uml-architect/gobackend/internal/service"
	"github.com/ai-uml-architect/gobackend/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const (
	anaID     = "11111111-1111-1111-1111-111111111111"
	brunoID   = "22222222-2222-2222-2222-222222222222"
	projectID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
)

func testServer(t *testing.T) (*httpapi.Server, string) {
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
	srv := httpapi.NewServer(service.New(ms), "http://localhost:3000", nil, nil)
	return srv, ""
}

func doAuthed(t *testing.T, srv *httpapi.Server, method, path string, body io.Reader, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func login(t *testing.T, srv *httpapi.Server, email, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	rec := doAuthed(t, srv, http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("login body is not JSON: %v", err)
	}
	for _, key := range []string{"accessToken", "userId", "displayName", "email"} {
		if resp[key] == "" {
			t.Fatalf("login response lacks %q: %v", key, resp)
		}
	}
	return resp["accessToken"]
}

func TestLoginContract(t *testing.T) {
	srv, _ := testServer(t)
	token := login(t, srv, "ana@example.com", "Password123!")
	if token == "" {
		t.Fatalf("expected a token")
	}

	cases := []struct {
		name, email, password string
		want                  int
	}{
		{name: "wrong password is 401", email: "ana@example.com", password: "nope", want: http.StatusUnauthorized},
		{name: "unknown email is 401", email: "ghost@example.com", password: "Password123!", want: http.StatusUnauthorized},
		{name: "blank email is 400", email: "", password: "Password123!", want: http.StatusBadRequest},
		{name: "malformed email is 400", email: "not-an-email", password: "Password123!", want: http.StatusBadRequest},
		{name: "blank password is 400", email: "ana@example.com", password: "", want: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{"email": tc.email, "password": tc.password})
			rec := doAuthed(t, srv, http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body), "")
			if rec.Code != tc.want {
				t.Errorf("expected %d, got %d (%s)", tc.want, rec.Code, rec.Body.String())
			}
			assertErrorEnvelope(t, rec)
		})
	}
}

func TestProjectCreationAndJoinContract(t *testing.T) {
	srv, _ := testServer(t)
	anaToken := login(t, srv, "ana@example.com", "Password123!")

	body, _ := json.Marshal(map[string]any{"name": "Física", "description": "Clase 4B"})
	rec := doAuthed(t, srv, http.MethodPost, "/api/v1/projects", bytes.NewReader(body), anaToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create project failed: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("create project body is not JSON: %v", err)
	}
	for _, key := range []string{"id", "name", "description", "role", "diagramCount", "accessCode"} {
		if _, ok := created[key]; !ok {
			t.Fatalf("create project response lacks %q: %v", key, created)
		}
	}
	if created["role"] != "OWNER" || created["diagramCount"] != float64(0) {
		t.Fatalf("creator must be OWNER with 0 diagrams: %v", created)
	}
	accessCode, _ := created["accessCode"].(string)
	if len(accessCode) < 6 {
		t.Fatalf("access code must be generated: %v", created)
	}
	projectID, _ := created["id"].(string)

	brunoToken := login(t, srv, "bruno@example.com", "Password123!")
	join := func(code string) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(map[string]string{"accessCode": code})
		return doAuthed(t, srv, http.MethodPost, "/api/v1/projects/join", bytes.NewReader(raw), brunoToken)
	}

	// Lower-case input must match the stored upper-case code.
	rec = join(strings.ToLower(accessCode))
	if rec.Code != http.StatusOK {
		t.Fatalf("join failed: %d %s", rec.Code, rec.Body.String())
	}
	var joined map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &joined); err != nil {
		t.Fatalf("join body is not JSON: %v", err)
	}
	if joined["id"] != projectID || joined["role"] != "COLLABORATOR" {
		t.Fatalf("unexpected join response: %v", joined)
	}

	// Joining twice is idempotent: same role, and the dashboard still lists one copy.
	rec = join(accessCode)
	if rec.Code != http.StatusOK {
		t.Fatalf("second join failed: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &joined); err != nil {
		t.Fatalf("second join body is not JSON: %v", err)
	}
	if joined["role"] != "COLLABORATOR" {
		t.Fatalf("repeat join changed the role: %v", joined)
	}
	rec = doAuthed(t, srv, http.MethodGet, "/api/v1/projects", nil, brunoToken)
	var projects []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &projects); err != nil {
		t.Fatalf("projects body is not JSON: %v", err)
	}
	if len(projects) != 1 || projects[0]["id"] != projectID {
		t.Fatalf("joined project must appear exactly once: %v", projects)
	}

	rec = join("ZZZZZZ")
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown code must be 404, got %d (%s)", rec.Code, rec.Body.String())
	}
	assertErrorEnvelope(t, rec)

	rec = join("   ")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("blank code must be 400, got %d (%s)", rec.Code, rec.Body.String())
	}
	assertErrorEnvelope(t, rec)

	bad, _ := json.Marshal(map[string]any{"name": "   "})
	rec = doAuthed(t, srv, http.MethodPost, "/api/v1/projects", bytes.NewReader(bad), anaToken)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("blank name must be 400, got %d (%s)", rec.Code, rec.Body.String())
	}
	assertErrorEnvelope(t, rec)

	rec = doAuthed(t, srv, http.MethodPost, "/api/v1/projects/join", bytes.NewReader(body), "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("join without a token must be 401, got %d", rec.Code)
	}
	assertErrorEnvelope(t, rec)
}

func TestProjectAndDiagramContract(t *testing.T) {
	srv, _ := testServer(t)
	token := login(t, srv, "ana@example.com", "Password123!")

	rec := doAuthed(t, srv, http.MethodGet, "/api/v1/projects", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list projects failed: %d %s", rec.Code, rec.Body.String())
	}
	var projects []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &projects); err != nil {
		t.Fatalf("projects body is not JSON: %v", err)
	}
	if len(projects) != 1 || projects[0]["role"] != "OWNER" {
		t.Fatalf("unexpected projects: %v", projects)
	}

	diagramPath := "/api/v1/projects/" + projectID + "/diagrams"
	doc := map[string]any{"schemaVersion": 1, "name": "Main", "classes": []any{}, "relationships": []any{}}
	raw, _ := json.Marshal(doc)
	rec = doAuthed(t, srv, http.MethodPost, diagramPath, bytes.NewReader(raw), token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create diagram failed: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("create body is not JSON: %v", err)
	}
	diagramID, _ := created["id"].(string)
	if diagramID == "" {
		t.Fatalf("create must return an id: %v", created)
	}

	rec = doAuthed(t, srv, http.MethodGet, diagramPath, nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list diagrams failed: %d %s", rec.Code, rec.Body.String())
	}

	itemPath := diagramPath + "/" + diagramID
	rec = doAuthed(t, srv, http.MethodGet, itemPath, nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("get diagram failed: %d %s", rec.Code, rec.Body.String())
	}

	update := map[string]any{"schemaVersion": 1, "name": "Renamed", "classes": []any{}, "relationships": []any{}}
	raw, _ = json.Marshal(update)
	rec = doAuthed(t, srv, http.MethodPut, itemPath, bytes.NewReader(raw), token)
	if rec.Code != http.StatusOK {
		t.Fatalf("update diagram failed: %d %s", rec.Code, rec.Body.String())
	}

	// PUT is autosave only: it must NOT create a version row. The next
	// version entry is created by an explicit POST /checkpoints below.
	checkpoint := map[string]any{"schemaVersion": 1, "name": "Stable", "classes": []any{}, "relationships": []any{}, "message": "v2"}
	raw, _ = json.Marshal(checkpoint)
	rec = doAuthed(t, srv, http.MethodPost, itemPath+"/checkpoints", bytes.NewReader(raw), token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("checkpoint failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = doAuthed(t, srv, http.MethodGet, itemPath+"/versions", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list versions failed: %d %s", rec.Code, rec.Body.String())
	}
	var versions []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &versions); err != nil {
		t.Fatalf("versions body is not JSON: %v", err)
	}
	if len(versions) != 2 || versions[0]["versionNumber"] != float64(2) {
		t.Fatalf("expected versions [2 1], got %v", versions)
	}

	rec = doAuthed(t, srv, http.MethodPost, itemPath+"/versions/1/restore", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("restore failed: %d %s", rec.Code, rec.Body.String())
	}
}

func TestDiagramErrorMapping(t *testing.T) {
	srv, _ := testServer(t)
	token := login(t, srv, "ana@example.com", "Password123!")
	diagramPath := "/api/v1/projects/" + projectID + "/diagrams"

	cases := []struct {
		name, method, path, token string
		body                      []byte
		want                      int
	}{
		{name: "unknown diagram is 404", method: http.MethodGet, path: diagramPath + "/00000000-0000-0000-0000-000000000000", token: token, want: http.StatusNotFound},
		{name: "malformed diagram id is 400", method: http.MethodGet, path: diagramPath + "/nope", token: token, want: http.StatusBadRequest},
		{name: "malformed project id is 400", method: http.MethodGet, path: "/api/v1/projects/nope/diagrams", token: token, want: http.StatusBadRequest},
		{name: "unknown version is 404", method: http.MethodPost, path: diagramPath + "/00000000-0000-0000-0000-000000000000/versions/9/restore", token: token, want: http.StatusNotFound},
		{name: "invalid document is 400", method: http.MethodPost, path: diagramPath, token: token, body: []byte(`{"schemaVersion":1,"name":"","classes":[],"relationships":[]}`), want: http.StatusBadRequest},
		{name: "bad token is 401", method: http.MethodGet, path: diagramPath, token: "bogus", want: http.StatusUnauthorized},
		{name: "missing token is 401", method: http.MethodGet, path: diagramPath, token: "", want: http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var reader *bytes.Reader
			if tc.body != nil {
				reader = bytes.NewReader(tc.body)
			}
			var rec *httptest.ResponseRecorder
			if reader != nil {
				rec = doAuthed(t, srv, tc.method, tc.path, reader, tc.token)
			} else {
				rec = doAuthed(t, srv, tc.method, tc.path, nil, tc.token)
			}
			if rec.Code != tc.want {
				t.Errorf("expected %d, got %d (%s)", tc.want, rec.Code, rec.Body.String())
			}
			assertErrorEnvelope(t, rec)
		})
	}
}

func TestNonMemberIsForbidden(t *testing.T) {
	ms := store.NewMemoryStore()
	hash, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	ms.SeedUser(store.User{ID: anaID, DisplayName: "Ana", Email: "ana@example.com", PasswordHash: string(hash)})
	ms.SeedUser(store.User{ID: "22222222-2222-2222-2222-222222222222", DisplayName: "Bruno", Email: "bruno@example.com", PasswordHash: string(hash)})
	desc := "private"
	other := "99999999-9999-9999-9999-999999999999"
	ms.SeedProject(other, "Private", &desc)
	ms.SeedMember(other, "22222222-2222-2222-2222-222222222222", "OWNER")
	srv := httpapi.NewServer(service.New(ms), "http://localhost:3000", nil, nil)
	token := login(t, srv, "ana@example.com", "Password123!")

	rec := doAuthed(t, srv, http.MethodGet, "/api/v1/projects/"+other+"/diagrams", nil, token)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d (%s)", rec.Code, rec.Body.String())
	}
	assertErrorEnvelope(t, rec)
}

func TestCORSBehavior(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/projects", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
		t.Errorf("preflight must succeed, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("preflight must echo the configured origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") == "https://evil.example.com" {
		t.Errorf("unconfigured origin must not be reflected")
	}
}
