package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/gorilla/websocket"
)

// stubResolver satisfies MembershipResolver. Members is keyed by
// "<projectId>:<userId>"; presence tests toggle membership to drive
// the hub's authorize gate deterministically.
type stubResolver struct {
	mu      sync.Mutex
	members map[string]bool
	docs    map[string]domain.DiagramDocument
}

func newStubResolver() *stubResolver {
	return &stubResolver{members: map[string]bool{}, docs: map[string]domain.DiagramDocument{}}
}

func (s *stubResolver) seed(projectID, userID string, doc domain.DiagramDocument) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.members[projectID+":"+userID] = true
	if doc.ID != nil {
		s.docs[projectID+":"+*doc.ID] = doc
	}
}

func (s *stubResolver) DisplayName(userID string) string { return userID }

func (s *stubResolver) IsMember(projectID, userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.members[projectID+":"+userID]
}

func (s *stubResolver) LatestDocument(_ context.Context, projectID, diagramID string) (domain.DiagramDocument, int, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc := s.docs[projectID+":"+diagramID]
	return doc, 0, doc.ReviewNumber, nil
}

func TestHubAuthorizesAndSendsSnapshot(t *testing.T) {
	res := newStubResolver()
	docID := "22222222-2222-2222-2222-222222222222"
	projectID := "11111111-1111-1111-1111-111111111111"
	res.seed(projectID, "alice", domain.DiagramDocument{ID: &docID, Name: "Alpha", ReviewNumber: 7})

	hub := NewHub(res, Config{FreshnessWindow: time.Second, Now: func() time.Time { return time.Unix(0, 0) }})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	handler := hub.Upgrade(func(r *http.Request) (string, bool) {
		if id := r.Header.Get("X-User"); id != "" {
			return id, true
		}
		return "", false
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/" + projectID + "/" + docID
	ws, _, err := websocket.DefaultDialer.Dial(url, http.Header{"X-User": []string{"alice"}})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	if _, msg, err := ws.ReadMessage(); err != nil {
		t.Fatalf("read snapshot: %v", err)
	} else {
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
		if snap.Document.Name != "Alpha" || snap.ReviewNumber != 7 {
			t.Fatalf("snapshot mismatch: %+v", snap)
		}
		if len(snap.Members) != 1 || snap.Members[0].UserID != "alice" {
			t.Fatalf("snapshot roster mismatch: %+v", snap.Members)
		}
	}
}

func TestHubRejectsNonMember(t *testing.T) {
	res := newStubResolver()
	projectID := "11111111-1111-1111-1111-111111111111"
	diagramID := "22222222-2222-2222-2222-222222222222"
	res.seed(projectID, "alice", domain.DiagramDocument{ID: &diagramID, Name: "Alpha"})

	hub := NewHub(res, Config{FreshnessWindow: time.Second, Now: func() time.Time { return time.Unix(0, 0) }})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := httptest.NewServer(hub.Upgrade(func(r *http.Request) (string, bool) {
		if id := r.Header.Get("X-User"); id != "" {
			return id, true
		}
		return "", false
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/" + projectID + "/" + diagramID
	_, resp, err := websocket.DefaultDialer.Dial(url, http.Header{"X-User": []string{"bob"}})
	if err == nil {
		t.Fatalf("non-member dial must fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("non-member dial must return 401, got %v", resp)
	}
}

func TestHubEmitsDiagramChanged(t *testing.T) {
	res := newStubResolver()
	projectID := "11111111-1111-1111-1111-111111111111"
	diagramID := "22222222-2222-2222-2222-222222222222"
	res.seed(projectID, "alice", domain.DiagramDocument{ID: &diagramID, Name: "Alpha"})
	res.seed(projectID, "bruno", domain.DiagramDocument{ID: &diagramID, Name: "Alpha"})

	hub := NewHub(res, Config{FreshnessWindow: 2 * time.Second, Now: func() time.Time { return time.Unix(0, 0) }})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	handler := hub.Upgrade(func(r *http.Request) (string, bool) {
		if id := r.Header.Get("X-User"); id != "" {
			return id, true
		}
		return "", false
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/" + projectID + "/" + diagramID
	aliceWS, _, err := websocket.DefaultDialer.Dial(url, http.Header{"X-User": []string{"alice"}})
	if err != nil {
		t.Fatalf("alice dial: %v", err)
	}
	defer aliceWS.Close()
	brunoWS, _, err := websocket.DefaultDialer.Dial(url, http.Header{"X-User": []string{"bruno"}})
	if err != nil {
		t.Fatalf("bruno dial: %v", err)
	}
	defer brunoWS.Close()

	// Drain per-peer snapshot envelopes. Alice also receives a
	// presence.join broadcast for bruno appearing.
	for _, c := range []struct {
		name string
		ws   *websocket.Conn
		want int // envelopes expected before broadcast
	}{
		{name: "alice", ws: aliceWS, want: 2}, // snapshot + presence.join(bruno)
		{name: "bruno", ws: brunoWS, want: 1}, // snapshot
	} {
		for i := 0; i < c.want; i++ {
			if _, _, err := c.ws.ReadMessage(); err != nil {
				t.Fatalf("%s drain %d: %v", c.name, i, err)
			}
		}
	}

	hub.BroadcastDiagramChanged(projectID, diagramID, domain.DiagramChangedEvent{
		ActorID: "alice", ReviewNumber: 8, Version: 1, Kind: "working-document",
	})

	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatalf("diagram.changed envelope not received")
		default:
		}
		_, msg, err := brunoWS.ReadMessage()
		if err != nil {
			t.Fatalf("read changed: %v", err)
		}
		var env domain.Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("envelope malformed: %v", err)
		}
		if env.Type != domain.EnvelopeDiagramChange {
			continue
		}
		raw, _ := json.Marshal(env.Payload)
		var evt domain.DiagramChangedEvent
		if err := json.Unmarshal(raw, &evt); err != nil {
			t.Fatalf("event payload malformed: %v", err)
		}
		if evt.Kind != "working-document" || evt.ReviewNumber != 8 {
			t.Fatalf("event mismatch: %+v", evt)
		}
		return
	}
}

func TestHubOriginGateRefusesUnknownOrigin(t *testing.T) {
	res := newStubResolver()
	projectID := "11111111-1111-1111-1111-111111111111"
	diagramID := "22222222-2222-2222-2222-222222222222"
	res.seed(projectID, "alice", domain.DiagramDocument{ID: &diagramID, Name: "Alpha"})

	hub := NewHub(res, Config{FreshnessWindow: time.Second, Now: func() time.Time { return time.Unix(0, 0) }})
	hub.SetHubOptions(HubOptions{
		AllowOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == "https://allowed.example" },
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := httptest.NewServer(hub.Upgrade(func(r *http.Request) (string, bool) {
		if id := r.Header.Get("X-User"); id != "" {
			return id, true
		}
		return "", false
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/" + projectID + "/" + diagramID
	hdr := http.Header{"X-User": []string{"alice"}, "Origin": []string{"https://evil.example"}}
	_, resp, err := websocket.DefaultDialer.Dial(url, hdr)
	if err == nil {
		t.Fatalf("evil origin must refuse upgrade")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("evil origin must return 403, got %v", resp)
	}
}
