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

// ErrReviewMismatch is returned by SaveDocument and AppendCheckpoint when the
// supplied expectedReview baseline no longer matches the row's current
// review_number. The accompanying int64 is the row's current value so the
// service can surface a 409 with the server's review number.
type ErrReviewMismatch struct {
	Current int64
}

func (e ErrReviewMismatch) Error() string { return "review number out of date" }

// IsReviewMismatch reports whether err originated as ErrReviewMismatch.
func IsReviewMismatch(err error) bool {
	var target ErrReviewMismatch
	return errors.As(err, &target)
}

// AsReviewMismatch extracts the current review number from err if it is a
// review-mismatch error. Returns 0, false otherwise.
func AsReviewMismatch(err error) (int64, bool) {
	var target ErrReviewMismatch
	if errors.As(err, &target) {
		return target.Current, true
	}
	return 0, false
}

// ErrNotFound is returned when a row or membership does not exist. Callers map
// it to 404 (diagrams/versions) or 401/403 (auth/membership).
var ErrNotFound = errors.New("not found")

// ErrAccessCodeTaken reports that a generated access code is already in use.
// Project creation retries with a fresh code when it sees this error, so the
// unique index on projects.access_code is never surfaced as a 500.
var ErrAccessCodeTaken = errors.New("access code already taken")

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

// ProjectRecord mirrors ProjectEntity: the project row plus the classroom
// access code collaborators use to join.
type ProjectRecord struct {
	ID          string
	Name        string
	Description *string
	AccessCode  string
	OwnerID     string
	CreatedAt   time.Time
}

// DiagramRecord mirrors DiagramEntity. CreatedBy is the acting user: it is
// stored on insert and left untouched on conflict updates (matching the JPA
// merge of a loaded entity, which never rewrites created_by). ReviewNumber
// is the monotonic work counter that bumps atomically on every successful
// save or explicit checkpoint.
type DiagramRecord struct {
	ID          string
	ProjectID   string
	Name        string
	Document    []byte
	CreatedBy   string
	UpdatedAt   time.Time
	ReviewNumber int64
}

// VersionRecord mirrors DiagramVersionEntity. CreatedBy records the acting
// user (annotation column on the table) and Message is an optional per-checkpoint
// note supplied by the client; both preserve authorship of explicit saves.
// ReviewNumber is the diagrams row's work counter at the moment of the
// checkpoint append, so the version-history UI can correlate the entry with
// concurrent autosaves and reject duplicate replays during restore.
type VersionRecord struct {
	ID           string
	DiagramID    string
	Number       int
	ReviewNumber int64
	Document     []byte
	CreatedBy    string
	CreatedAt    time.Time
	Message      *string
}

// Store is the full persistence surface used by the service layer.
type Store interface {
	FindUserByEmail(ctx context.Context, email string) (User, error)
	CreateToken(ctx context.Context, tok TokenRecord) error
	FindToken(ctx context.Context, hash string) (TokenRecord, error)
	AssignedProjects(ctx context.Context, userID string) ([]AssignedProject, error)
	IsMember(ctx context.Context, projectID, userID string) (bool, error)
	// FindProjectByAccessCode looks a project up by its classroom code,
	// ignoring case and surrounding whitespace-normalized caller input.
	FindProjectByAccessCode(ctx context.Context, accessCode string) (ProjectRecord, error)
	// CreateProject inserts p and its OWNER membership atomically. It returns
	// ErrAccessCodeTaken when the access code already exists.
	CreateProject(ctx context.Context, p ProjectRecord) (ProjectRecord, error)
	// JoinProject inserts a COLLABORATOR membership when absent and returns the
	// membership joined to its project and live diagram count. Joining twice is
	// a no-op that returns the role already held.
	JoinProject(ctx context.Context, projectID, userID string) (AssignedProject, error)
	// SaveDocument upserts the diagram row (the latest autosaved state) with
	// a CAS gate on the work counter. When expectedReview is nil, the row is
	// upserted unconditionally (CreateDiagram) and the bumped review_number
	// is returned. When expectedReview points to a stale value the call
	// returns ErrReviewMismatch with the row's current review_number, never
	// the bumped value. It does NOT write to diagram_versions; explicit
	// checkpoints own the version history.
	SaveDocument(ctx context.Context, d DiagramRecord, expectedReview *int64) (int64, error)
	// AppendCheckpoint inserts a new diagram_versions row copying d's current
	// document, assigns version_number = max+1, stamps created_by with
	// d.CreatedBy, optionally stores Message, and bumps the diagrams row's
	// review_number atomically inside the same transaction. CAS uses the
	// supplied expectedReview baseline (nil = skip).
	AppendCheckpoint(ctx context.Context, d DiagramRecord, expectedReview *int64, message *string) (VersionRecord, error)
	// CurrentReview returns the diagrams row's current review_number (0
	// when no row exists yet). Service uses it to hydrate the response.
	CurrentReview(ctx context.Context, diagramID string) (int64, error)
	// CurrentVersion returns the highest checkpoint number for a diagram
	// (0 when no checkpoint row exists yet). It is the value GetDiagram
	// echoes on the response so clients know what to send on the next PUT.
	CurrentVersion(ctx context.Context, diagramID string) (int, error)
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
