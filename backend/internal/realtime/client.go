package realtime

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/gorilla/websocket"
)

// Client is one upgraded WebSocket connection attached to a diagram-presence
// room. The connection's read pump (per-client goroutine launched by the
// hub on upgrade) drives heartbeats and surfaces envelope errors; the
// write pump is the only writer to the underlying websocket connection,
// delivering envelopes queued by the hub from the parallel broadcast path.
//
// All fields are mutable under c.mu; the read loop, the write loop, and
// the hub's broadcasts may touch them concurrently.
type Client struct {
	conn  *websocket.Conn
	mu    sync.Mutex
	queue []domain.Envelope

	projectID, diagramID string
	userID, displayName  string
	lastSeen             time.Time

	// Snapshot fields are populated by the Hub membership gate at join
	// time so the freshly-connected peer reconciliates without an extra
	// REST GET. They are immutable for the lifetime of the client.
	snapshotDocument domain.DiagramDocument
	snapshotVersion  int
	snapshotReview   int64

	// Closed signals the write pump that the connection has been torn
	// down. use io.EOF semantics so multiple closes are safe.
	closed bool
}

// lastSeenOf returns the per-client lastSeen timestamp. It is package-level
// so the hub's sweep loop and room roster helpers can call it without
// exposing Client internals.
func lastSeenOf(c *Client) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastSeen
}

// touch advances the per-client heartbeat clock; the read loop calls it
// whenever an inbound presence.heartbeat envelope arrives.
func (c *Client) touch(ts time.Time) {
	c.mu.Lock()
	c.lastSeen = ts
	c.mu.Unlock()
}

// enqueue pushes env onto the outbound queue. The size is bounded so a
// slow consumer cannot grow memory unbounded; when the queue is full the
// oldest entry is dropped to make room for the new one. Drop-on-overflow
// is acceptable for presence events because the next graph change or the
// next heartbeat refreshes the client's UI.
func (c *Client) enqueue(env domain.Envelope) {
	c.mu.Lock()
	if len(c.queue) >= outboundQueueCap {
		// Drop the oldest entry, append the new one.
		c.queue = append(c.queue[:0], c.queue[1:]...)
	}
	c.queue = append(c.queue, env)
	c.mu.Unlock()
}

// drainQueue pulls the current outbound queue and resets it; called by
// the write pump before encoding.
func (c *Client) drainQueue() []domain.Envelope {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.queue) == 0 {
		return nil
	}
	out := c.queue
	c.queue = nil
	return out
}

// tryClose is the best-effort connection tear-down used by the sweep loop
// when a peer is evicted. Idempotent — repeated calls are no-ops.
func (c *Client) tryClose() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// outboundQueueCap bounds the per-client outbound queue. 64 envelopes is
// roughly an hour of presence deltas in a small room; when a client falls
// behind we drop the oldest entries and rely on the eventual snapshot
// rebuild on reconnect.
const outboundQueueCap = 64

// enqueueError pushes an error envelope to the per-client outbound queue
// so the read/write pump can surface server-side failures to dashboards
// without disrupting the broadcast path. Used by Handle when the
// membership gate refuses a Join after the WebSocket upgrade.
func (c *Client) enqueueError(code, message string) {
	c.enqueue(domain.Envelope{
		Type:       domain.EnvelopeError,
		ServerTime: domain.FormatInstant(time.Now()),
		Payload: domain.RealtimeError{
			Code:    code,
			Message: message,
		},
	})
}

// writePump is the per-client goroutine that drains the outbound queue and
// JSON-encodes the envelopes onto the websocket connection. It exits when
// the connection closes or an unrecoverable encode/write error occurs.
// The pump is fan-in: many hub broadcasts may append envelopes
// concurrently, only the write pump serializes them onto the wire.
func (c *Client) writePump(done chan<- struct{}) {
	defer close(done)
	for {
		envs := c.drainQueue()
		if len(envs) == 0 {
			if c.isClosed() {
				return
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if err := c.writeEnvelopes(envs); err != nil {
			return
		}
	}
}

func (c *Client) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// writeEnvelopes JSON-encodes and writes every envelope in envs as a
// separate websocket frame. We don't batch to keep the per-event
// granularity the protocol relies on.
func (c *Client) writeEnvelopes(envs []domain.Envelope) error {
	conn := c.conn
	if conn == nil {
		return io.EOF
	}
	for _, env := range envs {
		payload, err := json.Marshal(env)
		if err != nil {
			return err
		}
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return err
		}
	}
	return nil
}
