package realtime

import (
	"sort"
	"sync"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
)

// Room is one diagram-presence room. It owns its peer roster and pushes
// outgoing envelopes onto each peer's outbound channel. The Hub mu guards
// the rooms map; once a Room is created, the room itself guards its peers
// (peersMu) so cross-room broadcasts do not serialize on the hub's lock.
type Room struct {
	projectID string
	diagramID string

	peersMu sync.RWMutex
	peers   map[*Client]struct{}
}

func keyOf(projectID, diagramID string) string {
	return projectID + "\x00" + diagramID
}

func newRoom(projectID, diagramID string) *Room {
	return &Room{
		projectID: projectID,
		diagramID: diagramID,
		peers:     make(map[*Client]struct{}),
	}
}

// addPeer registers c in the room. Caller (handleRegister) is responsible
// for sending the welcome snapshot; adding a peer must not block on outbound
// channels.
func (r *Room) addPeer(c *Client) {
	r.peersMu.Lock()
	r.peers[c] = struct{}{}
	r.peersMu.Unlock()
}

// removePeer detaches c from the room. Returns true if the room has no
// remaining peers so the hub can mark it for cleanup.
func (r *Room) removePeer(c *Client) bool {
	r.peersMu.Lock()
	delete(r.peers, c)
	remaining := len(r.peers)
	r.peersMu.Unlock()
	return remaining == 0
}

// count returns the live peer count, used by tests to assert sweep
// behaviour without depending on the public roster shape.
func (r *Room) count() int {
	r.peersMu.RLock()
	defer r.peersMu.RUnlock()
	return len(r.peers)
}

// roster returns a stable userID-sorted view of the room's members. The
// lastSeenOf callback is invoked under the read-lock so concurrent
// heartbeats can advance the per-client timestamp without blocking the
// snapshot producer for long.
func (r *Room) roster(lastSeenOf func(*Client) time.Time) []domain.PresenceMember {
	r.peersMu.RLock()
	defer r.peersMu.RUnlock()
	out := make([]domain.PresenceMember, 0, len(r.peers))
	for p := range r.peers {
		out = append(out, domain.PresenceMember{
			UserID:      p.userID,
			DisplayName: p.displayName,
			LastSeen:    domain.FormatInstant(lastSeenOf(p)),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UserID < out[j].UserID })
	return out
}

// sweepStalePeers evicts peers whose last_heartbeat is older than cutoff.
// Returns the evicted clients so the hub can disconnect them.
func (r *Room) sweepStalePeers(cutoff time.Time, lastSeenOf func(*Client) time.Time) []*Client {
	r.peersMu.Lock()
	defer r.peersMu.Unlock()
	evicted := make([]*Client, 0)
	for p := range r.peers {
		if lastSeenOf(p).Before(cutoff) {
			delete(r.peers, p)
			evicted = append(evicted, p)
		}
	}
	return evicted
}

// broadcastEnvelope fans out a single envelope to every peer in the room
// except the originator (from may be nil for hub-origin emissions such
// as diagram.changed). The originator's userID is matched so duplicate
// echoes never return to the producer.
func (r *Room) broadcastEnvelope(from *Client, env domain.Envelope) {
	r.peersMu.RLock()
	defer r.peersMu.RUnlock()
	for peer := range r.peers {
		if from != nil && peer.userID == from.userID {
			continue
		}
		peer.enqueue(env)
	}
}
