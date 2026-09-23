package realtime

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"
	"sync"
	"time"
)

// TicketSigner issues and verifies ONE-SHOT short-lived tickets for
// WebSocket handshakes. The HTTP router emits a ticket from a bearer-gated
// REST route (`POST /api/v1/projects/{p}/diagrams/{d}/realtime-tickets`)
// so the browser WebSocket can connect without writing a custom
// `Authorization` header that JS cannot set.
//
// Wire format (URL-safe base64, no padding):
//
//	<random_tokenID>.<exp_sec_be8>.<projectId>.<diagramId>.<userID>.<sig16>
//
// The signature is HMAC-SHA-256(secret, "<tokenID>.<exp>.<p>.<d>.<u>")
// truncated to 16 bytes. The tokenID is random per ticket so a replay
// across two upgrades of the same (project, diagram, user) cannot bond the
// same signature; Parse keeps an in-memory set of already-consumed
// tokenIDs and rejects any second use as ErrTicketInvalid.
//
// Tickets cap at MaxTicketTTL = 5 minutes. Past-exp tickets are
// rejected. The MembershipResolver is the real authorization gate; the
// ticket is just a way to open the socket browser-side.
type TicketSigner struct {
	secret []byte
	now    func() time.Time

	consumedMu sync.Mutex
	consumed   map[string]time.Time // tokenID -> consumed-at (TTL'd)
}

// NewTicketSigner returns a signer bound to secret. Secret length is
// not validated: the HMAC absorbs whatever bytes it gets, and tests can
// use any opaque string.
func NewTicketSigner(secret []byte, now func() time.Time) *TicketSigner {
	if now == nil {
		now = time.Now
	}
	s := &TicketSigner{
		secret:   secret,
		now:      now,
		consumed: make(map[string]time.Time),
	}
	go s.gcLoop()
	return s
}

// gcLoop prunes expired entries from the `consumed` map so the set size
// stays bounded by the issue rate, not the running total.
func (s *TicketSigner) gcLoop() {
	t := time.NewTicker(MaxTicketTTL)
	defer t.Stop()
	for range t.C {
		cutoff := s.now().Add(-MaxTicketTTL)
		s.consumedMu.Lock()
		for k, ts := range s.consumed {
			if ts.Before(cutoff) {
				delete(s.consumed, k)
			}
		}
		s.consumedMu.Unlock()
	}
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

// Issue signs a ticket whose first segment is a random tokenID so the
// signature cannot be replayed across two valid (project, diagram, user)
// triples. TTL is clamped to MaxTicketTTL.
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
	idBytes, err := randomTokenBytes(ticketRandomIDBytes)
	if err != nil {
		return "", err
	}
	tokenID := base64.RawURLEncoding.EncodeToString(idBytes)
	body := formatTicketBody(tokenID, exp, projectID, diagramID, userID)
	sig := v.sign(body)[:ticketSigLen]
	return body + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// Parse decodes a wire ticket and atomically checks it has not been seen
// before; the second Parse of the same tokenID fails with ErrTicketInvalid
// even before the TTL elapses. This makes the ticket one-shot end-to-end.
func (v *TicketSigner) Parse(raw string) (Ticket, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 6 {
		return Ticket{}, ErrTicketInvalid
	}
	if _, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil || len(parts[0]) < 8 {
		return Ticket{}, ErrTicketInvalid
	}
	exp, err := parseExp(parts[1])
	if err != nil {
		return Ticket{}, ErrTicketInvalid
	}
	if v.now().Unix() >= exp {
		return Ticket{}, ErrTicketInvalid
	}
	body := parts[0] + "." + parts[1] + "." + parts[2] + "." + parts[3] + "." + parts[4]
	want := v.sign(body)
	got, err := base64.RawURLEncoding.DecodeString(parts[5])
	if err != nil {
		return Ticket{}, ErrTicketInvalid
	}
	if !hmac.Equal(want[:ticketSigLen], got) {
		return Ticket{}, ErrTicketInvalid
	}
	v.consumedMu.Lock()
	if _, seen := v.consumed[parts[0]]; seen {
		v.consumedMu.Unlock()
		return Ticket{}, ErrTicketInvalid
	}
	v.consumed[parts[0]] = v.now()
	v.consumedMu.Unlock()
	return Ticket{
		ProjectID: parts[2],
		DiagramID: parts[3],
		UserID:    parts[4],
		ExpiresAt: time.Unix(exp, 0),
	}, nil
}

const (
	MaxTicketTTL        = 5 * time.Minute
	ticketSigLen        = 16
	ticketRandomIDBytes = 12
)

func formatTicketBody(tokenID string, exp int64, projectID, diagramID, userID string) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(exp))
	expStr := base64.RawURLEncoding.EncodeToString(buf[:])
	return tokenID + "." + expStr + "." + projectID + "." + diagramID + "." + userID
}

func randomTokenBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
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
