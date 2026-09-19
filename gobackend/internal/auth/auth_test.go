package auth_test

import (
	"strings"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/auth"
)

func TestHashTokenIsStableSHA256Hex(t *testing.T) {
	h1 := auth.HashToken("token-a")
	h2 := auth.HashToken("token-a")
	if h1 != h2 {
		t.Fatalf("hash must be deterministic, got %q and %q", h1, h2)
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %q", h1)
	}
	for _, c := range h1 {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			t.Fatalf("expected lowercase hex digest, got %q", h1)
		}
	}
	if auth.HashToken("token-a") == auth.HashToken("token-b") {
		t.Errorf("different tokens must hash differently")
	}
}

func TestNewRawTokenHasUUIDDotUUIDShape(t *testing.T) {
	raw := auth.NewRawToken()
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		t.Fatalf("expected uuid.uuid shape, got %q", raw)
	}
	for _, p := range parts {
		if len(p) != 36 || strings.Count(p, "-") != 4 {
			t.Fatalf("expected UUID part, got %q in %q", p, raw)
		}
	}
	if auth.NewRawToken() == raw {
		t.Errorf("tokens must be unique")
	}
}

func TestCheckPasswordAcceptsSeedHash(t *testing.T) {
	// BCrypt hash shipped by V1__create_diagram_storage.sql for Password123!.
	const seed = "$2y$10$0po1d5ULCpVopbelx2gsj.6UfMTCpVQka2C1LubEfBhi.aK.9.VmO"
	if !auth.CheckPassword(seed, "Password123!") {
		t.Errorf("seed password must verify against the V1 hash")
	}
	if auth.CheckPassword(seed, "wrong") {
		t.Errorf("wrong password must not verify")
	}
}
