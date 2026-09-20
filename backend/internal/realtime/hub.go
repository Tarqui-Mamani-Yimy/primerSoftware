// Package realtime implements the diagram-presence hub on top of
// gorilla/websocket. Authorization is enforced at join time via a
// MembershipResolver supplied by the caller (typically the service layer's
// Store-backed helpers), so the hub never trusts a request to have passed
// HTTP auth: every WebSocket upgrade is gated by a fresh check on the project
// membership table.
//
// Design constraints (from realtime-diagram-collaboration tracker):
//   - No CRDT/OT. The hub only mirrors state changes — clients keep their
//     own working copy and surface conflict-banner envelopes.
//   - Authorization is per-diagram. A user has access only to the WS room
//     behind a diagram they belong to; presence is never leaked across
//     projects or non-member diagrams.
//   - Heartbeat-driven liveness. A peer that does not ack a
//     presence.heartbeat before freshnessWindow (default 60s) is evicted
//     and a presence.leave envelope is broadcast to the remaining peers.
//
// The hub speaks the JSON envelopes declared in internal/domain. Envelopes
// are typed so clients can switch on the `type` field without parsing every
// field up-front; payloads are `any` to allow the same wrapper to carry the
// closed set of typed shapes.
package realtime

import (
	"context"
	"sync"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
)

// MembershipResolver is the boundary the hub needs to authorize a
// WebSocket connection. Production callers wire it to the store-backed
// helpers so the gate always reflects the latest membership state; unit
// tests inject a stub resolver to exercise the join and broadcast paths
// hermetically.
type MembershipResolver interface {
	// DisplayName returns the human-facing name for userID. The hub uses
	// it to label presence.roster rows; an empty string produces
	// "anonymous" so dashboards have something to render.
	DisplayName(userID string) string
	// IsMember reports whether userID is a member of projectID. The hub
	// reads it on every Join; a false answer closes the WebSocket with
	// 1008 (policy violation) so the client UI can read the close code
	// without parsing the JSON body.
	IsMember(projectID, userID string) bool
	// LatestDocument returns a complete snapshot of the diagram's
	// working state plus the latest version/review counters. It is called
	// on Join so the freshly connected peer reconciles without an extra
	// REST GET. Implementations should be cheap; the hub calls it once
	// per upgrade.
	LatestDocument(ctx context.Context, projectID, diagramID string) (domain.DiagramDocument, int, int64, error)
}

// Hub is the central fan-out point for diagram-presence events. There is
// exactly one Hub per process; it owns the room registry, the broadcast
// channels, and the sweep loop that evicts stale peers.
type Hub struct {
	resolver MembershipResolver

	mu            sync.RWMutex
	rooms         map[string]*Room
	byClient      map[*Client]bool
	pendingDelete map[string]time.Time

	events   chan broadcastRequest
	register chan *Client
	leave    chan *Client

	freshnessWindow time.Duration
	now             func() time.Time

	// upgradeOpts captures the configurable Origin allow-list and the
	// optional ticket verifier so production can swap them without
	// touching the upgrade wiring in main.
	upgradeOpts HubOptions

	wg     sync.WaitGroup
	stop   chan struct{}
	closed atomicBool
}

// atomicBool is the in-package minimal CAS boolean the hub uses for
// lifecycle gating. Kept internal because only the hub owns it.
type atomicBool struct {
	mu sync.Mutex
	v  bool
}

func (a *atomicBool) set(value bool) {
	a.mu.Lock()
	a.v = value
	a.mu.Unlock()
}
func (a *atomicBool) get() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.v
}

// broadcastRequest is the hub-facing envelope broadcast. projectID and
// diagramID identify the target room; Event is the typed payload the hub
// copies into the envelope sent to every other peer.
type broadcastRequest struct {
	projectID string
	diagramID string
	event     domain.DiagramChangedEvent
}

// Config groups tuning knobs the production main wires; tests override.
type Config struct {
	// FreshnessWindow bounds heartbeat liveness. A peer whose
	// last_heartbeat is older than this is considered offline and is
	// evicted at the next sweep tick. Zero defaults to 60s.
	FreshnessWindow time.Duration
	// Now makes wall-clock injectable; zero defaults to time.Now. Tests
	// freeze time to deterministically assert liveness boundaries.
	Now func() time.Time
}

// NewHub returns a Hub wired to the resolver with the given config knobs.
func NewHub(resolver MembershipResolver, cfg Config) *Hub {
	if cfg.FreshnessWindow <= 0 {
		cfg.FreshnessWindow = 60 * time.Second
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	h := &Hub{
		resolver:        resolver,
		rooms:           make(map[string]*Room),
		byClient:        make(map[*Client]bool),
		pendingDelete:   make(map[string]time.Time),
		events:          make(chan broadcastRequest, 64),
		register:        make(chan *Client, 32),
		leave:           make(chan *Client, 32),
		freshnessWindow: cfg.FreshnessWindow,
		now:             cfg.Now,
		stop:            make(chan struct{}),
	}
	return h
}

// Run starts the hub goroutine: it processes broadcasts, register/leave
// events, and runs the sweep tick that evicts stale peers. ctx cancel
// finishes in-flight channels and drains the sweep.
func (h *Hub) Run(ctx context.Context) {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		ticker := time.NewTicker(h.freshnessWindow / 6)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				h.closed.set(true)
				close(h.stop)
				return
			case req := <-h.events:
				h.fanout(req)
			case c := <-h.register:
				h.handleRegister(c)
			case c := <-h.leave:
				h.handleLeave(c)
			case <-ticker.C:
				h.sweep()
			case <-h.stop:
				return
			}
		}
	}()
}

// Close drains in-flight broadcasts and tears down the sweep loop.
func (h *Hub) Close() {
	h.closed.set(true)
	select {
	case <-h.stop:
		return
	default:
	}
}

// Join attaches a freshly upgraded WebSocket connection to the room behind
// (projectID, diagramID) for userID/displayName. The membership gate is
// enforced synchronously; Join returns false when the user is not a
// member, true on attach.
func (h *Hub) Join(ctx context.Context, c *Client) bool {
	if h.closed.get() {
		return false
	}
	if !h.resolver.IsMember(c.projectID, c.userID) {
		return false
	}
	doc, version, review, err := h.resolver.LatestDocument(ctx, c.projectID, c.diagramID)
	if err != nil {
		return false
	}
	c.snapshotDocument = doc
	c.snapshotVersion = version
	c.snapshotReview = review
	select {
	case h.register <- c:
		return true
	default:
		return false
	}
}

// BroadcastDiagramChanged fans out a domain.DiagramChangedEvent to the room
// behind (projectID, diagramID). It is invoked by the service after every
// successful autosave or explicit checkpoint; the hub drops the event for
// unknown rooms.
func (h *Hub) BroadcastDiagramChanged(projectID, diagramID string, evt domain.DiagramChangedEvent) {
	if h.closed.get() {
		return
	}
	select {
	case h.events <- broadcastRequest{projectID: projectID, diagramID: diagramID, event: evt}:
	default:
		// Channel full — drop the event so a slow consumer never blocks
		// the service's HTTP write path.
	}
}

// DispatchLeave signals the hub that a client's read loop finished. It is
// non-blocking: when the leave channel is full, the client is detached
// synchronously.
func (h *Hub) DispatchLeave(c *Client) {
	select {
	case h.leave <- c:
	default:
		h.handleLeave(c)
	}
}
