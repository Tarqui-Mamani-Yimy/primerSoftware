// Package store defines the persistence boundary ported from the Java Spring
// Data repositories (UserRepository, RefreshTokenRepository,
// ProjectRepository, ProjectMembershipRepository, DiagramRepository,
// DiagramVersionRepository).
package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned when a row or membership does not exist. Callers map
// it to 404 (diagrams/versions) or 401/403 (auth/membership).
var ErrNotFound = errors.New("not found")

// User mirrors UserEntity.
type User struct {
	ID           string
	DisplayName  string
	Email        string
	PasswordHash string
}

// TokenRecord mirrors RefreshTokenEntity.
type TokenRecord struct {
	ID        string
	UserID    string
	Hash      string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// AssignedProject is one membership row joined to its project plus the live
// diagram count, mirroring ProjectService.assigned().
type AssignedProject struct {
	ID           string
	Name         string
	Description  *string
	Role         string
	DiagramCount int
}

// DiagramRecord mirrors DiagramEntity. CreatedBy is the acting user: it is
// stored on insert and left untouched on conflict updates (matching the JPA
// merge of a loaded entity, which never rewrites created_by).
type DiagramRecord struct {
	ID        string
	ProjectID string
	Name      string
	Document  []byte
	CreatedBy string
	UpdatedAt time.Time
}

// VersionRecord mirrors DiagramVersionEntity.
type VersionRecord struct {
	ID        string
	DiagramID string
	Number    int
	Document  []byte
	CreatedAt time.Time
}

// Store is the full persistence surface used by the service layer.
type Store interface {
	FindUserByEmail(ctx context.Context, email string) (User, error)
	CreateToken(ctx context.Context, tok TokenRecord) error
	FindToken(ctx context.Context, hash string) (TokenRecord, error)
	AssignedProjects(ctx context.Context, userID string) ([]AssignedProject, error)
	IsMember(ctx context.Context, projectID, userID string) (bool, error)
	// SaveDiagram creates or updates d and appends a version row carrying the
	// same document, assigning version numbers as max+1. Implementations must
	// apply both writes atomically, mirroring @Transactional save().
	SaveDiagram(ctx context.Context, d DiagramRecord, v VersionRecord) (VersionRecord, error)
	FindDiagram(ctx context.Context, projectID, diagramID string) (DiagramRecord, error)
	ListDiagrams(ctx context.Context, projectID string) ([]DiagramRecord, error)
	ListVersions(ctx context.Context, diagramID string) ([]VersionRecord, error)
	FindVersion(ctx context.Context, diagramID string, number int) (VersionRecord, error)
}

// NewUUID returns a random UUID string in 8-4-4-4-12 lowercase hex form,
// matching java.util.UUID.randomUUID().toString().
func NewUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("store: crypto/rand failed: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	hexed := hex.EncodeToString(b[:])
	return hexed[0:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:32]
}
