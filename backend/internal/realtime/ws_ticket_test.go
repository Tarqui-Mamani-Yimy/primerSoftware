package realtime

// Ticket-handshake coverage for the WS upgrade path: the ticket is
// parsed/consumed exactly once, claims bind the (project, diagram, user)
// triple without re-parse, and bearer auth keeps working alongside.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/gorilla/websocket"
)

func ticketTestSetup(t *testing.T, res *stubResolver) (*Hub, *TicketSigner, context.CancelFunc) {
	t.Helper()
	hub := NewHub(res, Config{FreshnessWindow: time.Second, Now: time.Now})
	signer := NewTicketSigner([]byte("test-secret-0123456789abcdef"), nil)
	hub.SetHubOptions(HubOptions{Tickets: signer})
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)
	return hub, signer, cancel
}

// bearerDenied forces the ticket path: any Authorization header is rejected.
func bearerDenied(_ *http.Request) (string, bool) { return "", false }

func dialTicket(t *testing.T, srvURL, projectID, diagramID, ticket string, hdr http.Header) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srvURL, "http") + "/" + projectID + "/" + diagramID + "?ticket=" + ticket
	if hdr == nil {
		hdr = http.Header{}
	}
	return websocket.DefaultDialer.Dial(url, hdr)
}

func readSnapshot(t *testing.T, ws *websocket.Conn) domain.PresenceSnapshot {
	t.Helper()
	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
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
	var snap domain.PresenceSnapshot
	raw, _ := json.Marshal(env.Payload)
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("snapshot payload malformed: %v", err)
	}
	return snap
}

func TestTicketUpgradeSucceedsAndConsumesOnce(t *testing.T) {
	res := newStubResolver()
	projectID := "11111111-1111-1111-1111-111111111111"
	diagramID := "22222222-2222-2222-2222-222222222222"
	docID := diagramID
	res.seed(projectID, "alice", domain.DiagramDocument{ID: &docID, Name: "Alpha", ReviewNumber: 7})

	hub, signer, cancel := ticketTestSetup(t, res)
	defer cancel()
	srv := httptest.NewServer(hub.Upgrade(bearerDenied))
	defer srv.Close()

	ticket, err := signer.Issue(projectID, diagramID, "alice", time.Minute)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}
	ws, _, err := dialTicket(t, srv.URL, projectID, diagramID, ticket, nil)
	if err != nil {
		t.Fatalf("ticket dial must succeed (single parse): %v", err)
	}
	defer ws.Close()
	if snap := readSnapshot(t, ws); snap.Document.Name != "Alpha" || snap.ReviewNumber != 7 {
		t.Fatalf("snapshot mismatch: %+v", snap)
	}

	// The same ticket is one-shot: a second upgrade with it must fail.
	_, resp, err := dialTicket(t, srv.URL, projectID, diagramID, ticket, nil)
	if err == nil {
		t.Fatalf("ticket reuse must fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("ticket reuse must return 401, got %v", resp)
	}
}

func TestTicketTripleMismatchRejected(t *testing.T) {
	res := newStubResolver()
	projectID := "11111111-1111-1111-1111-111111111111"
	diagramA := "22222222-2222-2222-2222-222222222222"
	diagramB := "33333333-3333-3333-3333-333333333333"
	docA := diagramA
	res.seed(projectID, "alice", domain.DiagramDocument{ID: &docA, Name: "Alpha"})

	hub, signer, cancel := ticketTestSetup(t, res)
	defer cancel()
	srv := httptest.NewServer(hub.Upgrade(bearerDenied))
	defer srv.Close()

	// Ticket bound to diagram B, used on diagram A's path.
	ticket, err := signer.Issue(projectID, diagramB, "alice", time.Minute)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}
	ws, _, err := dialTicket(t, srv.URL, projectID, diagramA, ticket, nil)
	if err != nil {
		t.Fatalf("HTTP upgrade succeeds before the triple check: %v", err)
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatalf("triple-mismatched ticket must be closed with policy violation")
	}
}

func TestTicketNonMemberRejected(t *testing.T) {
	res := newStubResolver()
	projectID := "11111111-1111-1111-1111-111111111111"
	diagramID := "22222222-2222-2222-2222-222222222222"
	docID := diagramID
	res.seed(projectID, "alice", domain.DiagramDocument{ID: &docID, Name: "Alpha"})

	hub, signer, cancel := ticketTestSetup(t, res)
	defer cancel()
	srv := httptest.NewServer(hub.Upgrade(bearerDenied))
	defer srv.Close()

	ticket, err := signer.Issue(projectID, diagramID, "bob", time.Minute)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}
	_, resp, err := dialTicket(t, srv.URL, projectID, diagramID, ticket, nil)
	if err == nil {
		t.Fatalf("non-member ticket dial must fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("non-member ticket dial must return 401, got %v", resp)
	}
}

func TestTicketMalformedRejected(t *testing.T) {
	res := newStubResolver()
	projectID := "11111111-1111-1111-1111-111111111111"
	diagramID := "22222222-2222-2222-2222-222222222222"

	hub, _, cancel := ticketTestSetup(t, res)
	defer cancel()
	srv := httptest.NewServer(hub.Upgrade(bearerDenied))
	defer srv.Close()

	_, resp, err := dialTicket(t, srv.URL, projectID, diagramID, "not-a-ticket", nil)
	if err == nil {
		t.Fatalf("malformed ticket dial must fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("malformed ticket dial must return 401, got %v", resp)
	}
}

func TestBearerStillWorksWithTicketsConfigured(t *testing.T) {
	res := newStubResolver()
	projectID := "11111111-1111-1111-1111-111111111111"
	diagramID := "22222222-2222-2222-2222-222222222222"
	docID := diagramID
	res.seed(projectID, "alice", domain.DiagramDocument{ID: &docID, Name: "Alpha"})

	hub, _, cancel := ticketTestSetup(t, res)
	defer cancel()
	srv := httptest.NewServer(hub.Upgrade(func(r *http.Request) (string, bool) {
		if id := r.Header.Get("X-User"); id != "" {
			return id, true
		}
		return "", false
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/" + projectID + "/" + diagramID
	ws, _, err := websocket.DefaultDialer.Dial(url, http.Header{"X-User": []string{"alice"}})
	if err != nil {
		t.Fatalf("bearer dial must succeed: %v", err)
	}
	defer ws.Close()
	readSnapshot(t, ws)
}

func TestProjectAndDiagramFromIDParam(t *testing.T) {
	projectID := "11111111-1111-1111-1111-111111111111"
	diagramID := "22222222-2222-2222-2222-222222222222"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/{projectId}/diagrams/{id}/ws", func(w http.ResponseWriter, r *http.Request) {
		p, d := projectAndDiagramFromRequest(r)
		if p != projectID || d != diagramID {
			t.Errorf("extracted (%q, %q), want (%q, %q)", p, d, projectID, diagramID)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/api/v1/projects/" + projectID + "/diagrams/" + diagramID + "/ws")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}
