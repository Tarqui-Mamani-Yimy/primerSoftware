package realtime

import (
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
)

// handleRegister attaches c to the room, sends the freshly-connected peer
// an initial snapshot envelope (with the pre-join roster) and broadcasts a
// presence.join to existing peers. The room map is updated under the hub's
// mutex; the per-room roster mutations run under the room's own lock.
func (h *Hub) handleRegister(c *Client) {
	roomKey := keyOf(c.projectID, c.diagramID)
	h.mu.Lock()
	room, ok := h.rooms[roomKey]
	if !ok {
		room = newRoom(c.projectID, c.diagramID)
		h.rooms[roomKey] = room
	}
	// Re-opening a room we had marked for deletion after the previous
	// sweep: clear the deletion marker so the next sweep does not free
	// the room while we still have peers.
	delete(h.pendingDelete, roomKey)
	h.mu.Unlock()

	room.addPeer(c)
	h.byClient[c] = true

	// Send the freshly-joined peer the initial snapshot. The roster
	// intentionally excludes themselves (collected before addPeer
	// is visible above) — though since we already added them, the roster
	// includes them too. We send the post-add roster so the new peer
	// sees their own row, matching what other dashboards render.
	roster := room.roster(func(p *Client) time.Time { return p.lastSeen })
	c.enqueue(domain.Envelope{
		Type:       domain.EnvelopeSnapshot,
		ServerTime: domain.FormatInstant(h.now()),
		Payload: domain.PresenceSnapshot{
			ProjectID:    c.projectID,
			DiagramID:    c.diagramID,
			Members:      roster,
			Document:     c.snapshotDocument,
			Version:      c.snapshotVersion,
			ReviewNumber: c.snapshotReview,
		},
	})

	// Broadcast presence.join to everyone else in the room.
	room.broadcastEnvelope(c, domain.Envelope{
		Type:       domain.EnvelopePresenceJoin,
		ServerTime: domain.FormatInstant(h.now()),
		Payload: domain.PresenceDelta{
			ProjectID: c.projectID,
			DiagramID: c.diagramID,
			Member: domain.PresenceMember{
				UserID:      c.userID,
				DisplayName: c.displayName,
				LastSeen:    domain.FormatInstant(h.now()),
			},
		},
	})
}

// handleLeave detaches c from its room and fans out presence.leave to the
// remaining peers. When the room empties it is marked for deletion; the
// next sweep that finds it older than freshnessWindow frees the map entry
// to give a rapidly-reconnecting client a short grace window.
func (h *Hub) handleLeave(c *Client) {
	roomKey := keyOf(c.projectID, c.diagramID)
	h.mu.RLock()
	room, ok := h.rooms[roomKey]
	h.mu.RUnlock()
	if !ok {
		delete(h.byClient, c)
		return
	}
	empty := room.removePeer(c)
	delete(h.byClient, c)

	room.broadcastEnvelope(c, domain.Envelope{
		Type:       domain.EnvelopePresenceLeave,
		ServerTime: domain.FormatInstant(h.now()),
		Payload: domain.PresenceDelta{
			ProjectID: c.projectID,
			DiagramID: c.diagramID,
			Member: domain.PresenceMember{
				UserID:      c.userID,
				DisplayName: c.displayName,
				LastSeen:    domain.FormatInstant(h.now()),
			},
		},
	})

	h.mu.Lock()
	if empty {
		h.pendingDelete[roomKey] = h.now()
	}
	h.mu.Unlock()
}

// fanout emits a diagram.changed envelope to every peer in the target room.
// Unknown rooms are dropped (no peers yet = nothing to broadcast). The
// originator (the actor's userID matched against the message envelope) is
// filtered downstream by the per-room broadcast.
func (h *Hub) fanout(req broadcastRequest) {
	h.mu.RLock()
	room, ok := h.rooms[keyOf(req.projectID, req.diagramID)]
	h.mu.RUnlock()
	if !ok {
		return
	}
	origin := &Client{userID: req.event.ActorID}
	room.broadcastEnvelope(origin, domain.Envelope{
		Type:       domain.EnvelopeDiagramChange,
		ServerTime: domain.FormatInstant(h.now()),
		Payload:    req.event,
	})
}

// sweep is the periodic eviction loop. It runs every freshnessWindow/6 so a
// stale peer is detected within at most 60/6 = 10s in production before
// forced eviction. Two passes happen here:
//  1. Iterate every room and evict peers whose last_heartbeat is older than
//     now - freshnessWindow.
//  2. For each room whose last peer left earlier than now - freshnessWindow
//     (and was marked for deletion in handleLeave) remove the room map
//     entry so memory usage does not grow with short-lived diagrams.
func (h *Hub) sweep() {
	cutoff := h.now().Add(-h.freshnessWindow)
	h.mu.RLock()
	rooms := make([]*Room, 0, len(h.rooms))
	for _, r := range h.rooms {
		rooms = append(rooms, r)
	}
	h.mu.RUnlock()
	for _, r := range rooms {
		evicted := r.sweepStalePeers(cutoff, func(p *Client) time.Time { return p.lastSeen })
		for _, p := range evicted {
			// best-effort close; the write pump will detect this and shut down.
			p.tryClose()
		}
	}

	h.mu.Lock()
	for key, ts := range h.pendingDelete {
		if ts.Before(cutoff) {
			if r, ok := h.rooms[key]; ok && r.count() == 0 {
				delete(h.rooms, key)
			}
			delete(h.pendingDelete, key)
		}
	}
	h.mu.Unlock()
}
