package realtime

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"
	"time"
)

// TicketSigner issues and verifies short-lived, signed WS handshake
// tickets. The HTTP router emits a ticket from a cookie-eligible REST
// route (`POST /api/v1/projects/{}/diagrams/{}/realtime-tickets`) so the
// browser WebSocket can connect without sending the long-lived bearer in
// headers browsers cannot write from JavaScript.
//
// Wire format (URL-safe base64, no padding):
//
//	<exp_sec_be8>.<projectId=...>.<diagramId=...>.<userID=...>.<sig_truncated_16>
//
// The signature is HMAC-SHA-256(secret, "<exp>.<projectId>.<diagramId>.<userID>")
// truncated to 16 bytes to keep the URL short.
//
// This is NOT a replacement for auth. The ticket merely lets the browser
// open the socket; the hub's MembershipResolver still verifies project
// membership on every Join so a leaked or replayed ticket cannot grant
// presence to a non-member. Tickets cap at MaxTicketTTL to bound replay
// risk; the hub rejects past-expiry tickets at upgrade time.
type TicketSigner struct {
	secret []byte
	now    func() time.Time
}

// NewTicketSigner returns a signer bound to secret. Secret length is
// not validated: the HMAC absorbs whatever bytes it gets, and tests can
// use any opaque string.
func NewTicketSigner(secret []byte, now func() time.Time) *TicketSigner {
	if now == nil {
		now = time.Now
	}
	return &TicketSigner{secret: secret, now: now}
}

// ErrTicketInvalid is returned by Parse for any malformed, expired, or
// signature-mismatching ticket. Callers should map both branches of the
// upgrade gate (missing/invalid) to the same close code so clients cannot
// distinguish "you sent garbage" from "your ticket expired".
var ErrTicketInvalid = errors.New("realtime: ticket invalid")

// Ticket is the decoded form of a wire ticket.
type Ticket struct {
	UserID    string
	ProjectID string
	DiagramID string
	ExpiresAt time.Time
}

// Issue signs a ticket for the (project, diagram, user) triple with TTL.
// TTL is clamped to MaxTicketTTL at the boundary so the HTTP router
// cannot accidentally mint year-long tickets.
func (v *TicketSigner) Issue(projectID, diagramID, userID string, ttl time.Duration) (string, error) {
	if projectID == "" || diagramID == "" || userID == "" {
		return "", errors.New("realtime: ticket requires projectId, diagramId and userID")
	}
	if ttl <= 0 {
		ttl = MaxTicketTTL
	}
	if ttl > MaxTicketTTL {
		ttl = MaxTicketTTL
	}
	exp := v.now().Add(ttl).Unix()
	body := formatTicketBody(exp, projectID, diagramID, userID)
	sig := v.sign(body)[:ticketSigLen]
	return body + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// Parse decodes a wire ticket and returns the verified payload. Any
// malformed piece, expired timestamp, or signature mismatch collapses into
// ErrTicketInvalid.
func (v *TicketSigner) Parse(raw string) (Ticket, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 5 {
		return Ticket{}, ErrTicketInvalid
	}
	exp, err := parseExp(parts[0])
	if err != nil {
		return Ticket{}, ErrTicketInvalid
	}
	if v.now().Unix() >= exp {
		return Ticket{}, ErrTicketInvalid
	}
	body := parts[0] + "." + parts[1] + "." + parts[2] + "." + parts[3]
	want := v.sign(body)
	got, err := base64.RawURLEncoding.DecodeString(parts[4])
	if err != nil {
		return Ticket{}, ErrTicketInvalid
	}
	if !hmac.Equal(want[:16], got) {
		return Ticket{}, ErrTicketInvalid
	}
	return Ticket{
		ProjectID: parts[1],
		DiagramID: parts[2],
		UserID:    parts[3],
		ExpiresAt: time.Unix(exp, 0),
	}, nil
}

const MaxTicketTTL = 5 * time.Minute
const ticketSigLen = 16

func formatTicketBody(exp int64, projectID, diagramID, userID string) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(exp))
	expStr := base64.RawURLEncoding.EncodeToString(buf[:])
	return expStr + "." + projectID + "." + diagramID + "." + userID
}

func parseExp(s string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(raw) != 8 {
		return 0, ErrTicketInvalid
	}
	return int64(binary.BigEndian.Uint64(raw)), nil
}

func (v *TicketSigner) sign(body string) []byte {
	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(body))
	return mac.Sum(nil)
}
