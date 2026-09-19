// Package service ports ProjectService and DiagramService: membership gating,
// project assignment with live diagram counts, classroom-code project creation
// and joining, diagram CRUD with forced schemaVersion/id normalization,
// checkpoint-only version history (autosave no longer appends), and
// restore-as-new-checkpoint semantics with optimistic-concurrency 409.
//
// Versioning follows an explicit-checkpoint model: autosave writes the
// working document (PUT), and POST /checkpoints is the only path that grows
// the diagram_versions history. CreatedBy is stamped on the version row so
// authorship stays attached to the user that pressed "Create checkpoint".
package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/auth"
	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/store"
)

// CredentialsError mirrors AuthService.InvalidCredentialsException: the HTTP
// layer maps it to 401 {"message":"Invalid email or password"}.
type CredentialsError struct{}

func (CredentialsError) Error() string { return "Invalid email or password" }

// ForbiddenError mirrors ProjectService.AccessDeniedException: the HTTP layer
// maps it to 403 {"message":"Project membership required"}.
type ForbiddenError struct{}

func (ForbiddenError) Error() string { return "Project membership required" }

// NotFoundError mirrors NoSuchElementException: the HTTP layer maps it to 404
// carrying the message ("Diagram not found", "Version not found").
type NotFoundError struct{ Message string }

func (e NotFoundError) Error() string { return e.Message }

// ConflictError is the optimistic-concurrency envelope: the client sent a
// baseline version that no longer matches the diagrams row. The HTTP layer
// maps it to 409 with the document the server holds so the client can merge
// or replace its state.
type ConflictError struct {
	Current  domain.DiagramDocument
	Expected int
}

func (e ConflictError) Error() string { return "Diagram was changed by another collaborator" }

// ValidationError mirrors IllegalArgumentException from semantic validation
// (and bean-validation failures): the HTTP layer maps it to 400.
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

// Service bundles the ported service layer over a Store.
type Service struct {
	store store.Store
}

// New returns a Service over s.
func New(s store.Store) *Service { return &Service{store: s} }

// Login ports AuthService.login: case-insensitive email lookup, BCrypt check,
// opaque token issuance with 8h expiry. Unknown users and bad passwords both
// yield CredentialsError (no oracle).
func (s *Service) Login(ctx context.Context, email, password string) (domain.LoginResponse, error) {
	if errs := domain.ValidateLoginInput(email, password); len(errs) > 0 {
		return domain.LoginResponse{}, ValidationError{Message: strings.Join(errs, "; ")}
	}
	user, err := s.store.FindUserByEmail(ctx, email)
	if err != nil || !auth.CheckPassword(user.PasswordHash, password) {
		return domain.LoginResponse{}, CredentialsError{}
	}
	raw := auth.NewRawToken()
	now := time.Now().UTC()
	if err := s.store.CreateToken(ctx, store.TokenRecord{
		ID: store.NewUUID(), UserID: user.ID,
		Hash: auth.HashToken(raw), ExpiresAt: now.Add(auth.TokenTTL),
	}); err != nil {
		return domain.LoginResponse{}, err
	}
	return domain.LoginResponse{
		AccessToken: raw, UserID: user.ID,
		DisplayName: user.DisplayName, Email: user.Email,
	}, nil
}

// Authenticate ports AuthService.userFor: stored-hash lookup with revoked_at
// null and expiry checks.
func (s *Service) Authenticate(ctx context.Context, raw string) (string, error) {
	tok, err := s.store.FindToken(ctx, auth.HashToken(raw))
	if err != nil {
		return "", CredentialsError{}
	}
	if tok.RevokedAt != nil || !tok.ExpiresAt.After(time.Now().UTC()) {
		return "", CredentialsError{}
	}
	return tok.UserID, nil
}

// AssignedProjects ports ProjectService.assigned().
func (s *Service) AssignedProjects(ctx context.Context, userID string) ([]domain.ProjectResponse, error) {
	rows, err := s.store.AssignedProjects(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ProjectResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.ProjectResponse{
			ID: r.ID, Name: r.Name, Description: r.Description,
			Role: r.Role, DiagramCount: r.DiagramCount,
		})
	}
	return out, nil
}

// accessCodeAlphabet omits characters that are easy to confuse when read aloud
// or copied by hand (I, L, O, U, 0, 1); accessCodeLength matches the
// six-character codes seeded in V1, and accessCodeAttempts bounds the retry
// loop that keeps a rare collision off the 500 path.
const (
	accessCodeAlphabet = "ABCDEFGHJKMNPQRSTVWXYZ23456789"
	accessCodeLength   = 6
	accessCodeAttempts = 5
)

// newAccessCode returns a uniformly random classroom code. crypto/rand.Int
// avoids the modulo bias a plain byte%len scheme would introduce.
func newAccessCode() (string, error) {
	limit := big.NewInt(int64(len(accessCodeAlphabet)))
	code := make([]byte, accessCodeLength)
	for i := range code {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		code[i] = accessCodeAlphabet[n.Int64()]
	}
	return string(code), nil
}

// CreateProject ports ProjectService.create: the creator becomes OWNER and the
// row receives a generated, unique access code. Generation retries on the rare
// collision (store.ErrAccessCodeTaken) so the unique index never surfaces as a
// 500. projects.description stays nullable.
func (s *Service) CreateProject(ctx context.Context, userID, name string, description *string) (domain.ProjectCreatedResponse, error) {
	if errs := domain.ValidateProjectInput(name); len(errs) > 0 {
		return domain.ProjectCreatedResponse{}, ValidationError{Message: strings.Join(errs, "; ")}
	}
	trimmed := strings.TrimSpace(name)
	for attempt := 0; attempt < accessCodeAttempts; attempt++ {
		code, err := newAccessCode()
		if err != nil {
			return domain.ProjectCreatedResponse{}, err
		}
		created, err := s.store.CreateProject(ctx, store.ProjectRecord{
			ID: store.NewUUID(), Name: trimmed, Description: description,
			AccessCode: code, OwnerID: userID, CreatedAt: time.Now().UTC(),
		})
		if errors.Is(err, store.ErrAccessCodeTaken) {
			continue
		}
		if err != nil {
			return domain.ProjectCreatedResponse{}, err
		}
		return domain.ProjectCreatedResponse{
			ID: created.ID, Name: created.Name, Description: created.Description,
			Role: "OWNER", DiagramCount: 0, AccessCode: created.AccessCode,
		}, nil
	}
	return domain.ProjectCreatedResponse{},
		fmt.Errorf("service: could not generate a unique access code after %d attempts", accessCodeAttempts)
}

// JoinProject ports the join-by-code flow: the code is matched
// case-insensitively, membership is idempotent (a second join keeps the role
// already held), and the response is the same list item the dashboard renders.
func (s *Service) JoinProject(ctx context.Context, userID, accessCode string) (domain.ProjectResponse, error) {
	if errs := domain.ValidateAccessCodeInput(accessCode); len(errs) > 0 {
		return domain.ProjectResponse{}, ValidationError{Message: strings.Join(errs, "; ")}
	}
	project, err := s.store.FindProjectByAccessCode(ctx, domain.NormalizeAccessCode(accessCode))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.ProjectResponse{}, NotFoundError{Message: "Project not found"}
		}
		return domain.ProjectResponse{}, err
	}
	joined, err := s.store.JoinProject(ctx, project.ID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.ProjectResponse{}, NotFoundError{Message: "Project not found"}
		}
		return domain.ProjectResponse{}, err
	}
	return domain.ProjectResponse{
		ID: joined.ID, Name: joined.Name, Description: joined.Description,
		Role: joined.Role, DiagramCount: joined.DiagramCount,
	}, nil
}

func (s *Service) requireMember(ctx context.Context, projectID, userID string) error {
	ok, err := s.store.IsMember(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ForbiddenError{}
	}
	return nil
}

func normalize(raw domain.DiagramDocument, id string) (domain.DiagramDocument, []byte, error) {
	if errs := domain.ValidateDiagramInput(raw); len(errs) > 0 {
		return domain.DiagramDocument{}, nil, ValidationError{Message: strings.Join(errs, "; ")}
	}
	// save() forces schemaVersion 1 and the path/generated id, ignoring input.
	doc := domain.DiagramDocument{
		SchemaVersion: 1, ID: &id, Version: raw.Version,
		Name:          raw.Name,
		Classes:       raw.Classes,
		Relationships: raw.Relationships,
	}
	if doc.Classes == nil {
		doc.Classes = []domain.UmlClass{}
	}
	if doc.Relationships == nil {
		doc.Relationships = []domain.Relationship{}
	}
	payload, err := json.Marshal(doc)
	if err != nil {
		return domain.DiagramDocument{}, nil, err
	}
	return doc, payload, nil
}

// hydrateVersion mirrors the current version into the document the service
// hands back to a client. It is called whenever the response document is
// built, so clients always see the integer they must echo on the next PUT or
// POST /checkpoints call.
func (s *Service) hydrateVersion(ctx context.Context, doc *domain.DiagramDocument) error {
	if doc.ID == nil {
		return nil
	}
	current, err := s.store.CurrentVersion(ctx, *doc.ID)
	if err != nil {
		return err
	}
	doc.Version = current
	return nil
}

func toDocument(payload []byte) (domain.DiagramDocument, error) {
	var doc domain.DiagramDocument
	if err := json.Unmarshal(payload, &doc); err != nil {
		return domain.DiagramDocument{}, err
	}
	return doc, nil
}

// CreateDiagram ports DiagramService.create: membership gate, fresh id, a
// version-1 row as the implicit first checkpoint so the new diagram has a
// stable starting point in the history. The HTTP layer maps success to 201.
func (s *Service) CreateDiagram(ctx context.Context, projectID, userID string, raw domain.DiagramDocument) (domain.DiagramDocument, error) {
	if err := s.requireMember(ctx, projectID, userID); err != nil {
		return domain.DiagramDocument{}, err
	}
	id := store.NewUUID()
	doc, payload, err := normalize(raw, id)
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	now := time.Now().UTC()
	if err := s.store.SaveWorkingDocument(ctx,
		store.DiagramRecord{ID: id, ProjectID: projectID, Name: doc.Name, Document: payload, CreatedBy: userID, UpdatedAt: now},
	); err != nil {
		return domain.DiagramDocument{}, err
	}
	if _, err := s.store.AppendCheckpoint(ctx,
		store.DiagramRecord{ID: id, ProjectID: projectID, Name: doc.Name, Document: payload, CreatedBy: userID, UpdatedAt: now},
		strPtr("Initial revision"),
	); err != nil {
		return domain.DiagramDocument{}, err
	}
	if err := s.hydrateVersion(ctx, &doc); err != nil {
		return domain.DiagramDocument{}, err
	}
	return doc, nil
}

func strPtr(s string) *string { return &s }

// GetDiagram ports DiagramService.get.
func (s *Service) GetDiagram(ctx context.Context, projectID, diagramID, userID string) (domain.DiagramDocument, error) {
	if err := s.requireMember(ctx, projectID, userID); err != nil {
		return domain.DiagramDocument{}, err
	}
	row, err := s.store.FindDiagram(ctx, projectID, diagramID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.DiagramDocument{}, NotFoundError{Message: "Diagram not found"}
		}
		return domain.DiagramDocument{}, err
	}
	doc, err := toDocument(row.Document)
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	if err := s.hydrateVersion(ctx, &doc); err != nil {
		return domain.DiagramDocument{}, err
	}
	return doc, nil
}

// ListDiagrams ports DiagramService.list (updatedAt desc). Each summary
// carries a current version number so the CLI / mobile list rows can render
// "v3" alongside the diagram name without an extra round-trip.
func (s *Service) ListDiagrams(ctx context.Context, projectID, userID string) ([]domain.DiagramSummary, error) {
	if err := s.requireMember(ctx, projectID, userID); err != nil {
		return nil, err
	}
	rows, err := s.store.ListDiagrams(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.DiagramSummary, 0, len(rows))
	for _, r := range rows {
		version, err := s.store.CurrentVersion(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.DiagramSummary{
			ID: r.ID, Name: r.Name,
			UpdatedAt: domain.FormatInstant(r.UpdatedAt),
			Version:   version,
		})
	}
	return out, nil
}

// UpdateDiagram is the autosave path: it writes the new working document
// without appending a version row. The client may include Version in the body
// (or an If-Match header that the HTTP layer converts to it) to enable
// optimistic concurrency; a stale baseline yields a 409 carrying the server's
// current document so the client can reload and retry. The returned
// document mirrors the unchanged current version.
func (s *Service) UpdateDiagram(ctx context.Context, projectID, diagramID, userID string, raw domain.DiagramDocument, ifMatch *int) (domain.DiagramDocument, error) {
	if err := s.requireMember(ctx, projectID, userID); err != nil {
		return domain.DiagramDocument{}, err
	}
	if _, err := s.store.FindDiagram(ctx, projectID, diagramID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.DiagramDocument{}, NotFoundError{Message: "Diagram not found"}
		}
		return domain.DiagramDocument{}, err
	}
	current, err := s.store.CurrentVersion(ctx, diagramID)
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	effectiveBaseline := raw.Version
	if ifMatch != nil {
		effectiveBaseline = *ifMatch
	}
	if effectiveBaseline != current {
		existing, err := s.loadDiagramDocument(ctx, projectID, diagramID, userID)
		if err != nil {
			return domain.DiagramDocument{}, err
		}
		return domain.DiagramDocument{}, ConflictError{Current: existing, Expected: effectiveBaseline}
	}
	doc, payload, err := normalize(raw, diagramID)
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	if err := s.store.SaveWorkingDocument(ctx,
		store.DiagramRecord{ID: diagramID, ProjectID: projectID, Name: doc.Name, Document: payload, CreatedBy: userID, UpdatedAt: time.Now().UTC()},
	); err != nil {
		return domain.DiagramDocument{}, err
	}
	doc.Version = current
	return doc, nil
}

// CreateCheckpoint is the explicit-save path: it first performs a guarded
// SaveWorkingDocument (If-Match/version guard for concurrent checkpoints)
// and then appends a new diagram_versions row stamping the actor as created_by.
// This is the only call that grows the version history.
func (s *Service) CreateCheckpoint(ctx context.Context, projectID, diagramID, userID string, raw domain.DiagramDocument, message *string) (domain.DiagramVersion, error) {
	if err := s.requireMember(ctx, projectID, userID); err != nil {
		return domain.DiagramVersion{}, err
	}
	if _, err := s.store.FindDiagram(ctx, projectID, diagramID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.DiagramVersion{}, NotFoundError{Message: "Diagram not found"}
		}
		return domain.DiagramVersion{}, err
	}
	current, err := s.store.CurrentVersion(ctx, diagramID)
	if err != nil {
		return domain.DiagramVersion{}, err
	}
	if raw.Version != current {
		existing, err := s.loadDiagramDocument(ctx, projectID, diagramID, userID)
		if err != nil {
			return domain.DiagramVersion{}, err
		}
		return domain.DiagramVersion{}, ConflictError{Current: existing, Expected: raw.Version}
	}
	doc, payload, err := normalize(raw, diagramID)
	if err != nil {
		return domain.DiagramVersion{}, err
	}
	diagram := store.DiagramRecord{ID: diagramID, ProjectID: projectID, Name: doc.Name, Document: payload, CreatedBy: userID, UpdatedAt: time.Now().UTC()}
	if err := s.store.SaveWorkingDocument(ctx, diagram); err != nil {
		return domain.DiagramVersion{}, err
	}
	version, err := s.store.AppendCheckpoint(ctx, diagram, message)
	if err != nil {
		return domain.DiagramVersion{}, err
	}
	reflected, err := toDocument(version.Document)
	if err != nil {
		return domain.DiagramVersion{}, err
	}
	reflected.Version = version.Number
	return domain.DiagramVersion{
		ID: version.ID, VersionNumber: version.Number,
		CreatedAt: domain.FormatInstant(version.CreatedAt),
		CreatedBy: version.CreatedBy,
		Message:   version.Message,
		Document:  reflected,
	}, nil
}

// loadDiagramDocument is a small helper the 409 path uses to fetch and
// hydrate the document the server is now holding.
func (s *Service) loadDiagramDocument(ctx context.Context, projectID, diagramID, userID string) (domain.DiagramDocument, error) {
	return s.GetDiagram(ctx, projectID, diagramID, userID)
}

// ListVersions ports DiagramService.versions (versionNumber desc). The leading
// get() preserves the membership gate and the 404 for missing diagrams. Each
// row carries the actor and the optional checkpoint message so the
// version-history UI can attribute authorship.
func (s *Service) ListVersions(ctx context.Context, projectID, diagramID, userID string) ([]domain.DiagramVersion, error) {
	if _, err := s.GetDiagram(ctx, projectID, diagramID, userID); err != nil {
		return nil, err
	}
	rows, err := s.store.ListVersions(ctx, diagramID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.DiagramVersion, 0, len(rows))
	for _, r := range rows {
		doc, err := toDocument(r.Document)
		if err != nil {
			return nil, err
		}
		doc.Version = r.Number
		out = append(out, domain.DiagramVersion{
			ID: r.ID, VersionNumber: r.Number,
			CreatedAt: domain.FormatInstant(r.CreatedAt),
			CreatedBy: r.CreatedBy,
			Message:   r.Message,
			Document:  doc,
		})
	}
	return out, nil
}

// RestoreDiagram ports DiagramService.restore: the old version payload
// becomes the new working document and a new explicit checkpoint records the
// restore as a deliberate save. The new version carries a "Restored v{N}"
// message so the history shows what produced the change.
func (s *Service) RestoreDiagram(ctx context.Context, projectID, diagramID, userID string, number int) (domain.DiagramDocument, error) {
	if _, err := s.GetDiagram(ctx, projectID, diagramID, userID); err != nil {
		return domain.DiagramDocument{}, err
	}
	row, err := s.store.FindVersion(ctx, diagramID, number)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.DiagramDocument{}, NotFoundError{Message: "Version not found"}
		}
		return domain.DiagramDocument{}, err
	}
	snapshot, err := toDocument(row.Document)
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	snapshot.Version = 0
	_, err = s.CreateCheckpoint(ctx, projectID, diagramID, userID, snapshot, strPtr(fmt.Sprintf("Restored v%d", number)))
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	return s.GetDiagram(ctx, projectID, diagramID, userID)
}
