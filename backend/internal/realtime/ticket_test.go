package realtime

import (
	"strings"
	"testing"
	"time"
)

func TestTicketRoundTrip(t *testing.T) {
	fixed := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	tk := NewTicketSigner([]byte("k"), func() time.Time { return fixed })
	raw, err := tk.Issue("p1", "d1", "u1", MaxTicketTTL)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	got, err := tk.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.ProjectID != "p1" || got.DiagramID != "d1" || got.UserID != "u1" {
		t.Fatalf("decoded mismatch: %+v", got)
	}
	if !got.ExpiresAt.Equal(fixed.Add(MaxTicketTTL)) {
		t.Fatalf("expiresAt mismatched: got %s want %s", got.ExpiresAt, fixed.Add(MaxTicketTTL))
	}
}

func TestTicketRejectsPastExp(t *testing.T) {
	base := time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC)
	tk := NewTicketSigner([]byte("k"), func() time.Time { return base })
	raw, err := tk.Issue("p1", "d1", "u1", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	// Move clock forward past TTL.
	later := NewTicketSigner([]byte("k"), func() time.Time { return base.Add(2 * time.Minute) })
	if _, err := later.Parse(raw); err == nil {
		t.Fatalf("past-exp ticket must be rejected")
	}
}

func TestTicketRejectsSignatureMismatch(t *testing.T) {
	fixed := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	tk1 := NewTicketSigner([]byte("k1"), func() time.Time { return fixed })
	tk2 := NewTicketSigner([]byte("k2"), func() time.Time { return fixed })
	raw, err := tk1.Issue("p1", "d1", "u1", MaxTicketTTL)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := tk2.Parse(raw); err == nil {
		t.Fatalf("cross-secret ticket must be rejected")
	}
}

func TestTicketRejectsTamperedBody(t *testing.T) {
	fixed := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	tk := NewTicketSigner([]byte("k"), func() time.Time { return fixed })
	raw, err := tk.Issue("p1", "d1", "u1", MaxTicketTTL)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	parts := strings.Split(raw, ".")
	parts[3] = "evil" // user id
	tampered := strings.Join(parts, ".")
	if _, err := tk.Parse(tampered); err == nil {
		t.Fatalf("tampered ticket must be rejected")
	}
}

func TestTicketRejectsMalformed(t *testing.T) {
	tk := NewTicketSigner([]byte("k"), nil)
	for _, raw := range []string{"", "one", "a.b.c", "a.b.c.d.e.f", "abc.d.e.f"} {
		if _, err := tk.Parse(raw); err == nil {
			t.Fatalf("malformed ticket %q must be rejected", raw)
		}
	}
}

func TestTicketIssuerClampsTTL(t *testing.T) {
	tk := NewTicketSigner([]byte("k"), nil)
	if _, err := tk.Issue("p1", "d1", "u1", 0); err != nil {
		t.Fatalf("zero TTL must default, got: %v", err)
	}
	if _, err := tk.Issue("p1", "d1", "u1", 24*time.Hour); err != nil {
		t.Fatalf("huge TTL must clamp, got: %v", err)
	}
	if _, err := tk.Issue("", "d1", "u1", time.Minute); err == nil {
		t.Fatalf("missing project must error")
	}
}
