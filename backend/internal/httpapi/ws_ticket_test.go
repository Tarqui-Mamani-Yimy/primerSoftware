package httpapi_test

// End-to-end proof that the WS route serves ticket handshakes WITHOUT a
// Bearer gate: a browser WebSocket cannot set an Authorization header, so
// wrapping the upgrade in withAuth would block the ticket path outright.
// The upgrade itself still enforces ticket claims + membership.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/httpapi"
	"github.com/ai-uml-architect/gobackend/internal/realtime"
	"github.com/ai-uml-architect/gobackend/internal/service"
	"github.com/ai-uml-architect/gobackend/internal/store"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
)

func TestWSTicketUpgradeWithoutBearerHeader(t *testing.T) {
	ms := store.NewMemoryStore()
	hash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	ms.SeedUser(store.User{ID: anaID, DisplayName: "Ana", Email: "ana@example.com", PasswordHash: string(hash)})
	desc := "sales"
	ms.SeedProject(projectID, "Sales", &desc)
	ms.SeedMember(projectID, anaID, "OWNER")

	svc := service.New(ms)
	hub := httpapi.NewHub(svc)
	signer := realtime.NewTicketSigner([]byte("e2e-test-secret-0123456789"), nil)
	hub.SetHubOptions(realtime.HubOptions{Tickets: signer})
	srv := httpapi.NewServer(svc, "http://localhost:3000", hub, signer)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	postJSON := func(path, token string, body any) (int, map[string]any) {
		t.Helper()
		var reader *bytes.Reader
		if body != nil {
			raw, _ := json.Marshal(body)
			reader = bytes.NewReader(raw)
		} else {
			reader = bytes.NewReader(nil)
		}
		req, _ := http.NewRequest(http.MethodPost, ts.URL+path, reader)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}

	// Login.
	code, loginResp := postJSON("/api/v1/auth/login",
		"", map[string]string{"email": "ana@example.com", "password": "Password123!"})
	if code != http.StatusOK {
		t.Fatalf("login failed: %d %v", code, loginResp)
	}
	token, _ := loginResp["accessToken"].(string)
	if token == "" {
		t.Fatalf("login response lacks accessToken: %v", loginResp)
	}

	// Create diagram.
	code, created := postJSON("/api/v1/projects/"+projectID+"/diagrams", token,
		map[string]any{"schemaVersion": 1, "name": "Main", "classes": []any{}, "relationships": []any{}})
	if code != http.StatusCreated {
		t.Fatalf("create diagram failed: %d %v", code, created)
	}
	diagramID, _ := created["id"].(string)
	if diagramID == "" {
		t.Fatalf("create must return an id: %v", created)
	}

	// Issue a one-shot ticket (bearer-gated REST, as production does).
	code, ticketResp := postJSON("/api/v1/projects/"+projectID+"/diagrams/"+diagramID+"/realtime-tickets", token, nil)
	if code != http.StatusCreated {
		t.Fatalf("issue ticket failed: %d %v", code, ticketResp)
	}
	ticket, _ := ticketResp["ticket"].(string)
	if ticket == "" {
		t.Fatalf("ticket response lacks ticket: %v", ticketResp)
	}

	// Upgrade with the ticket and NO Authorization header: this is the
	// browser path that withAuth would have blocked with 401.
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") +
		"/api/v1/projects/" + projectID + "/diagrams/" + diagramID +
		"/ws?ticket=" + url.QueryEscape(ticket)
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ticket-only WS upgrade must succeed: %v", err)
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var env domain.Envelope
	if err := json.Unmarshal(msg, &env); err != nil {
		t.Fatalf("snapshot envelope malformed: %v", err)
	}
	if env.Type != domain.EnvelopeSnapshot {
		t.Fatalf("first envelope must be snapshot, got %q", env.Type)
	}
	raw, _ := json.Marshal(env.Payload)
	var snap domain.PresenceSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("snapshot payload malformed: %v", err)
	}
	if snap.Document.Name != "Main" || snap.ReviewNumber <= 0 {
		t.Fatalf("snapshot must carry the document and its review baseline: %+v", snap)
	}

	// Negative: neither ticket nor bearer header still fails.
	plainURL := "ws" + strings.TrimPrefix(ts.URL, "http") +
		"/api/v1/projects/" + projectID + "/diagrams/" + diagramID + "/ws"
	_, resp, err := websocket.DefaultDialer.Dial(plainURL, nil)
	if err == nil {
		t.Fatalf("unauthenticated WS upgrade must fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated WS upgrade must return 401, got %v", resp)
	}
}
