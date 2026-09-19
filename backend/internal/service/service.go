// Package service ports ProjectService and DiagramService: membership gating,
// project assignment with live diagram counts, classroom-code project creation
// and joining, diagram CRUD with forced schemaVersion/id normalization,
// version appends with max+1 numbering, and restore-as-new-write semantics.
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
	if errs := domain.ValidateDocument(raw); len(errs) > 0 {
		return domain.DiagramDocument{}, nil, ValidationError{Message: strings.Join(errs, "; ")}
	}
	// save() forces schemaVersion 1 and the path/generated id, ignoring input.
	doc := domain.DiagramDocument{
		SchemaVersion: 1, ID: &id, Name: raw.Name,
		Classes: raw.Classes, Relationships: raw.Relationships,
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

func toDocument(payload []byte) (domain.DiagramDocument, error) {
	var doc domain.DiagramDocument
	if err := json.Unmarshal(payload, &doc); err != nil {
		return domain.DiagramDocument{}, err
	}
	return doc, nil
}

// CreateDiagram ports DiagramService.create: membership gate, fresh id,
// version 1 row in the same transaction. The HTTP layer maps success to 201.
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
	_, err = s.store.SaveDiagram(ctx,
		store.DiagramRecord{ID: id, ProjectID: projectID, Name: doc.Name, Document: payload, CreatedBy: userID, UpdatedAt: now},
		store.VersionRecord{ID: store.NewUUID(), CreatedAt: now},
	)
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	return doc, nil
}

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
	return toDocument(row.Document)
}

// ListDiagrams ports DiagramService.list (updatedAt desc).
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
		out = append(out, domain.DiagramSummary{
			ID: r.ID, Name: r.Name, UpdatedAt: domain.FormatInstant(r.UpdatedAt),
		})
	}
	return out, nil
}

// UpdateDiagram ports DiagramService.update: membership gate, rewrite plus a
// new version row.
func (s *Service) UpdateDiagram(ctx context.Context, projectID, diagramID, userID string, raw domain.DiagramDocument) (domain.DiagramDocument, error) {
	if err := s.requireMember(ctx, projectID, userID); err != nil {
		return domain.DiagramDocument{}, err
	}
	if _, err := s.store.FindDiagram(ctx, projectID, diagramID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.DiagramDocument{}, NotFoundError{Message: "Diagram not found"}
		}
		return domain.DiagramDocument{}, err
	}
	doc, payload, err := normalize(raw, diagramID)
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	now := time.Now().UTC()
	_, err = s.store.SaveDiagram(ctx,
		store.DiagramRecord{ID: diagramID, ProjectID: projectID, Name: doc.Name, Document: payload, CreatedBy: userID, UpdatedAt: now},
		store.VersionRecord{ID: store.NewUUID(), CreatedAt: now},
	)
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	return doc, nil
}

// ListVersions ports DiagramService.versions (versionNumber desc). The leading
// get() preserves the membership gate and the 404 for missing diagrams.
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
		out = append(out, domain.DiagramVersion{
			ID: r.ID, VersionNumber: r.Number,
			CreatedAt: domain.FormatInstant(r.CreatedAt), Document: doc,
		})
	}
	return out, nil
}

// RestoreDiagram ports DiagramService.restore: the old version payload becomes
// a NEW write plus a new version row.
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
	return s.UpdateDiagram(ctx, projectID, diagramID, userID, snapshot)
}
