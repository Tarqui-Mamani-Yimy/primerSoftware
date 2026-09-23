package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/httpapi"
)

const (
	projectPath = "/api/v1/projects/11111111-1111-1111-1111-111111111111"
	diagramPath = projectPath + "/diagrams/22222222-2222-2222-2222-222222222222"
)

func TestRouteTableMatchesPublicContract(t *testing.T) {
	want := []httpapi.Route{
		{Method: http.MethodPost, Pattern: "/api/v1/auth/login"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/join"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects/{projectId}/diagrams"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/{projectId}/diagrams"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}"},
		{Method: http.MethodPut, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/checkpoints"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/versions"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/versions/{version}/restore"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/artifact"},
		{Method: http.MethodPost, Pattern: "/api/v1/voice/transcriptions"},
	}
	got := httpapi.Routes()
	if len(got) != len(want) {
		t.Fatalf("expected %d routes, got %v", len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("route %d: expected %+v, got %+v", i, w, got[i])
		}
	}
}

func protectedCases() []struct{ name, method, path string } {
	return []struct{ name, method, path string }{
		{name: "list projects", method: http.MethodGet, path: "/api/v1/projects"},
		{name: "create project", method: http.MethodPost, path: "/api/v1/projects"},
		{name: "join project", method: http.MethodPost, path: "/api/v1/projects/join"},
		{name: "list diagrams", method: http.MethodGet, path: projectPath + "/diagrams"},
		{name: "create diagram", method: http.MethodPost, path: projectPath + "/diagrams"},
		{name: "get diagram", method: http.MethodGet, path: diagramPath},
		{name: "update diagram", method: http.MethodPut, path: diagramPath},
		{name: "list versions", method: http.MethodGet, path: diagramPath + "/versions"},
		{name: "restore version", method: http.MethodPost, path: diagramPath + "/versions/3/restore"},
		{name: "voice transcription", method: http.MethodPost, path: "/api/v1/voice/transcriptions"},
	}
}

func doRequest(t *testing.T, method, path, auth string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	httpapi.NewMux().ServeHTTP(rec, req)
	return rec
}

func assertErrorEnvelope(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body is not JSON: %v (%q)", err, rec.Body.String())
	}
	msg, ok := body["message"]
	if !ok {
		t.Fatalf("error body lacks {message} key: %v", body)
	}
	if s, ok := msg.(string); !ok || s == "" {
		t.Errorf("error message must be a non-empty string: %v", body)
	}
}

func TestProtectedRoutesRequireBearerScheme(t *testing.T) {
	for _, tc := range protectedCases() {
		t.Run(tc.name+" without header is 401", func(t *testing.T) {
			rec := doRequest(t, tc.method, tc.path, "")
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", rec.Code)
			}
			assertErrorEnvelope(t, rec)
		})
		t.Run(tc.name+" with non-bearer is 401", func(t *testing.T) {
			rec := doRequest(t, tc.method, tc.path, "Basic abc123")
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", rec.Code)
			}
			assertErrorEnvelope(t, rec)
		})
		t.Run(tc.name+" with bearer reaches stub", func(t *testing.T) {
			rec := doRequest(t, tc.method, tc.path, "Bearer opaque-token")
			if rec.Code != http.StatusNotImplemented {
				t.Errorf("expected 501 stub, got %d", rec.Code)
			}
			assertErrorEnvelope(t, rec)
		})
	}
}

func TestLoginPermittedWithoutAuth(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/v1/auth/login",
		"")
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("login must not require auth, got 401")
	}
	if rec.Code != http.StatusNotImplemented {
		t.Errorf("expected 501 stub, got %d", rec.Code)
	}
	assertErrorEnvelope(t, rec)
}

func TestMethodNotAllowed(t *testing.T) {
	rec := doRequest(t, http.MethodDelete, "/api/v1/projects", "Bearer opaque-token")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestUnknownPathIsNotFound(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/v1/nope", "Bearer opaque-token")
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestBearerSchemeIsCaseSensitivePrefix(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/v1/projects", "bearer opaque-token")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for lowercase bearer, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Errorf("expected JSON content type, got %q", rec.Header().Get("Content-Type"))
	}
}
