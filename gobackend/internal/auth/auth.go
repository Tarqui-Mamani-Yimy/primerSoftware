// Package auth ports AuthService's opaque-token mechanics: raw tokens are
// uuid.uuid, only the SHA-256 hex digest is stored, passwords are checked
// with BCrypt, and tokens live for 8 hours unless revoked.
package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// TokenTTL mirrors Duration.ofHours(8) in AuthService.login.
const TokenTTL = 8 * time.Hour

// NewRawToken returns a fresh opaque token in uuid.uuid form, mirroring
// UUID.randomUUID() + "." + UUID.randomUUID().
func NewRawToken() string {
	return store.NewUUID() + "." + store.NewUUID()
}

// HashToken returns the lowercase SHA-256 hex digest of a raw token, mirroring
// AuthService.hash.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// CheckPassword reports whether password matches a BCrypt hash. It accepts the
// $2y$ hashes shipped by the V1 seed migration.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
