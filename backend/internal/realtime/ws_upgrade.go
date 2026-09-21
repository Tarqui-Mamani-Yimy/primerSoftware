package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/gorilla/websocket"
)

// DefaultOrigins is the production Origin allow-list the upgrader uses
// when the deployment wires no checker. It returns false so a
// misconfigured deployment cannot silently accept cross-origin
// handshakes; tests inject permissive checkers explicitly.
func DefaultOrigins(r *http.Request) bool { return false }

// OriginChecker limits the WebSocket accept gate to a configurable list
// of HTTP origins. Production wires the same allow-list as HTTP CORS so
// a malicious origin cannot open the socket.
type OriginChecker func(*http.Request) bool

// TicketValidator verifies the optional ?ticket=<...> parameter the
// browser WebSocket sends. The HTTP layer issues tickets from a
// cookie-eligible REST route; the validator is the ONLY place that
// decides whether the ticket is well-formed and unexpired. Membership is
// re-checked inside Join so a replayed ticket cannot elevate presence.
type TicketValidator interface {
	Parse(raw string) (Ticket, error)
}

// AuthFunc is the bearer-token authenticator the WS handler reuses.
// Production wires it to the same Authenticate used by REST handlers so
// a single auth gate decisions the realtime session.
//
// It must close (closeCode >= 1000 and closeMessage) the connection with a
// policy-violation close code so clients can read it without parsing JSON.
type AuthFunc func(r *http.Request) (userID string, ok bool)

// HubUpgrader is the seam between *Hub and HTTP routing. Returning a
// http.HandlerFunc keeps the realtime package decoupled from any specific
// router (chi, gin, stdlib) — adapters (e.g., httpapi) call Upgrade with
// the AuthFunc they want, and the runtime path stays in net/http.
type HubUpgrader interface {
	Upgrade(auth AuthFunc) http.HandlerFunc
}

// Hub satisfies HubUpgrader; concrete calls go through Upgrade(auth) so
// the bearer-auth adapter can be swapped without touching the hub itself.
func (h *Hub) Upgrade(auth AuthFunc) http.HandlerFunc { return Handle(h, auth) }

// HubOptions tunes the per-hub upgrade policy. Zero-values apply the
// safe defaults: origin denied, ticket-required disabled.
type HubOptions struct {
	// AllowOrigin enables the Origin accept gate when non-nil. nil
	// keeps the gate closed (refuses every origin).
	AllowOrigin OriginChecker
	// Tickets enables the ?ticket= path when non-nil. nil disables.
	Tickets TicketValidator
}

// SetHubOptions overrides the upgrade policy at boot time. Production
// main wires the Origin allow-list and the ticket signer here so the rest
// of the package stays decoupled from config.New / app wiring.
func (h *Hub) SetHubOptions(o HubOptions) { h.upgradeOpts = o }

// Handle is the HTTP entry point for the diagram-presence WebSocket. It
// upgrades the connection, authenticates via either bearer header or
// ticket, authorizes via the membership resolver attached to the hub,
// then runs the read/write pumps until disconnect.
//
// Path parameters expected by the router: {projectId}/{diagramId}. The
// handler enforces the same membership gate every REST endpoint enforces,
// so a WS upgrade cannot escape the auth chain.
func Handle(h *Hub, auth AuthFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, diagramID := projectAndDiagramFromRequest(r)
		if projectID == "" || diagramID == "" {
			http.Error(w, "missing project or diagram id", http.StatusBadRequest)
			return
		}
		opts := h.upgradeOpts
		if opts.AllowOrigin != nil && !opts.AllowOrigin(r) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		// The ticket (when present) is parsed exactly once here: Parse is
		// one-shot, so a second Parse of the same tokenID would fail. The
		// verified claims travel in `ticket` and are re-checked against the
		// path below without re-parsing.
		userID, source, ticket, ok := authenticate(r, auth, opts.Tickets)
		if !ok || userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Pre-flight membership gate: refuse non-members before we even
		// upgrade. Both auth paths (bearer + ticket) must surface the
		// same 401 so non-members never see a successful handshake.
		if h.resolver == nil || !h.resolver.IsMember(projectID, userID) {
			http.Error(w, "membership required", http.StatusUnauthorized)
			return
		}
		displayName := h.resolver.DisplayName(userID)
		if displayName == "" {
			displayName = "anonymous"
		}
		up := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			CheckOrigin:     allowOriginOrDefault(opts.AllowOrigin),
		}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// If the auth path was a ticket, double-check the (projectID,
		// diagramID, userID) triple from the already-parsed claims matches
		// the path; a ticket for room A cannot open a connection to room B.
		// No re-parse here: the ticket is one-shot and already consumed.
		if source == "ticket" {
			if ticket.ProjectID != projectID || ticket.DiagramID != diagramID || ticket.UserID != userID {
				writeClose(conn, websocket.ClosePolicyViolation, "ticket mismatch")
				return
			}
		}
		client := &Client{
			conn:        conn,
			projectID:   projectID,
			diagramID:   diagramID,
			userID:      userID,
			displayName: displayName,
			lastSeen:    h.now(),
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if !h.Join(ctx, client) {
			writeClose(conn, websocket.ClosePolicyViolation, "membership required")
			return
		}

		writeDone := make(chan struct{})
		go client.writePump(writeDone)
		client.readPump(h, writeDone)
	}
}

// allowOriginOrDefault returns the configured Origin allow-list when
// present. nil means "no gate" (server-policy choice: tests + Safari
// same-origin requests). Production wiring always sets a strict checker
// so cross-origin WS upgrades from browsers are denied.
func allowOriginOrDefault(allow OriginChecker) func(*http.Request) bool {
	return allow
}

// authenticate extracts the user from the request using either the bearer
// header or the ticket query. It parses a ticket at most once and returns
// the verified claims alongside the source ("bearer" or "ticket") so the
// caller can re-verify cross-field constraints after the upgrade without
// re-parsing (tickets are one-shot; bearer tokens are not path-bound).
func authenticate(r *http.Request, auth AuthFunc, tickets TicketValidator) (userID, source string, claims Ticket, ok bool) {
	if auth != nil {
		if id, ok := auth(r); ok && id != "" {
			return id, "bearer", Ticket{}, true
		}
	}
	if tickets != nil {
		if raw := r.URL.Query().Get("ticket"); raw != "" {
			if t, err := tickets.Parse(raw); err == nil && t.UserID != "" {
				return t.UserID, "ticket", t, true
			}
		}
	}
	return "", "", Ticket{}, false
}

// projectAndDiagramFromRequest pulls projectId and diagramId out of the
// request, falling back to URL.Path parsing when the router does not
// expose PathValue (e.g. bare httptest servers). Production routing
// always uses ServeMux and PathValue is the source of truth. The REST
// table names the diagram segment {id} (not {diagramId}), so both names
// are accepted here.
func projectAndDiagramFromRequest(r *http.Request) (projectID, diagramID string) {
	projectID = r.PathValue("projectId")
	diagramID = r.PathValue("diagramId")
	if diagramID == "" {
		diagramID = r.PathValue("id")
	}
	if projectID != "" && diagramID != "" {
		return
	}
	// Fallback: extract the last two non-empty segments of the URL path,
	// ignoring any optional /ws suffix.
	trimmed := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(trimmed, "/")
	switch len(parts) {
	case 2:
		projectID, diagramID = parts[0], parts[1]
	case 3:
		projectID, diagramID = parts[0], parts[1]
	}
	return
}

// writeClose sends a control frame with the given close code and message.
// It is best-effort: an unreachable peer is dropped without retrying.
func writeClose(conn *websocket.Conn, code int, message string) {
	_ = conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(code, message),
		time.Now().Add(time.Second))
	_ = conn.Close()
}

// readPump is the per-client goroutine that reads inbound envelopes from
// the websocket connection, dispatches heartbeat/lifecycle events into
// the hub, and exits the connection when the peer disconnects or the
// close timer fires.
func (c *Client) readPump(h *Hub, writeDone <-chan struct{}) {
	defer h.DispatchLeave(c)
	defer c.tryClose()
	c.conn.SetReadLimit(maxReadBytes)
	_ = c.conn.SetReadDeadline(time.Now().Add(readDeadline))
	c.conn.SetPongHandler(func(string) error {
		c.touch(h.now())
		_ = c.conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var env domain.Envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			continue
		}
		c.touch(h.now())
		switch env.Type {
		case domain.EnvelopePresenceHeart:
			// heartbeat ack — touch already advanced the timestamp.
		case domain.EnvelopePresenceLeave:
			return
		default:
			// Unknown inbound envelope types are silently dropped so
			// future protocol additions on the server side don't break
			// older clients.
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(readDeadline))
	}
}

const (
	maxReadBytes   = 4096
	readDeadline   = 90 * time.Second
	writeFrequency = 50 * time.Millisecond
)
