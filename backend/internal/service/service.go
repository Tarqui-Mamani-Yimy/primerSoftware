// Package service ports ProjectService and DiagramService: membership gating,
// project assignment with live diagram counts, classroom-code project creation
// and joining, diagram CRUD with forced schemaVersion/id normalization,
// checkpoint-only version history (autosave no longer appends), restore-as-new-
// checkpoint semantics, and optimistic-concurrency 409 that surfaces the
// server's current review number so clients can retry without guessing.
//
// Versioning follows an explicit-checkpoint model: autosave writes the
// working document (PUT) and POST /checkpoints is the only path that grows
// the diagram_versions history. CreatedBy is stamped on the version row so
// authorship stays attached to the user that pressed "Create checkpoint".
//
// Realtime collaboration: the service also feeds a presence hub. Every
// successful autosave or /checkpoints POST emits a DiagramChanged event
// (actor + bumped review number) so connected collaborators see the new
// baseline without polling, and dashboards stay in sync with the live
// read_state and active members of each diagram room. Authorization is
// enforced at the hub boundary; this package never trusts a caller to have
// already passed membership.
package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

// DiagramPresenceBroadcaster is the integration point between the service
// and the realtime hub. Implementations project the actor + the new review
// number onto every other connected member of the same diagram. A nil
// broadcaster is treated as a no-op so cold-start unit tests do not have to
// build a hub to pass the type check.
type DiagramPresenceBroadcaster interface {
	BroadcastDiagramChanged(projectID, diagramID string, evt domain.DiagramChangedEvent)
}

// ConflictError is the optimistic-concurrency envelope: the client sent a
// baseline review number (or version) that no longer matches the server.
// The HTTP layer maps it to 409 carrying the document the server holds so
// the client can merge or replace its state and retry.
type ConflictError struct {
	Current  domain.DiagramDocument
	Expected int
	ExpectedReview int64
	ActualReview   int64
}

func (e ConflictError) Error() string { return "Diagram was changed by another collaborator" }

// ValidationError mirrors IllegalArgumentException from semantic validation
// (and bean-validation failures): the HTTP layer maps it to 400.
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

// Service bundles the ported service layer over a Store.
type Service struct {
	store     store.Store
	broadcast DiagramPresenceBroadcaster
	jhipGen   ArtifactGenerator
}

// Store returns the underlying store the service is bound to. Realtime
// callers (the WS upgrade handler) read it to authorize the membership
// gate and to hydrate the initial document snapshot, both of which are
// store-backed operations the realtime layer is not allowed to re-implement.
func (s *Service) Store() store.Store { return s.store }

// New returns a Service over s with no broadcaster (autosave/checkpoint
// callers do not need real-time integration in unit tests).
func New(s store.Store) *Service { return &Service{store: s} }

// NewWithBroadcaster returns a Service that fans out DiagramChanged events
// after every successful write. The broadcaster may be nil, in which case
// the service runs in a polling-only mode.
func NewWithBroadcaster(s store.Store, broadcaster DiagramPresenceBroadcaster) *Service {
	if broadcaster == nil {
		return New(s)
	}
	return &Service{store: s, broadcast: broadcaster}
}

// AttachBroadcaster installs the broadcaster atomically after the Service
// is constructed. Production main uses it to rewire the Service once the
// realtime Hub is built (so the Service can stay decoupled from the
// realtime package at construction time).
func (s *Service) AttachBroadcaster(b DiagramPresenceBroadcaster) {
	s.broadcast = b
}

// emitChanged is the typed hook called after every successful write. A nil
// broadcaster is intentionally swallowed (unit-test runs) and a non-nil
// panic from a buggy hub is recorded but never propagates back to the
// caller so the database write stays authoritative.
func (s *Service) emitChanged(projectID, diagramID string, evt domain.DiagramChangedEvent) {
	if s.broadcast == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("service: broadcaster panic for %s/%s: %v", projectID, diagramID, r)
		}
	}()
	s.broadcast.BroadcastDiagramChanged(projectID, diagramID, evt)
}

// Login ports AuthService.login.
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

// Authenticate ports AuthService.userFor.
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

const (
	accessCodeAlphabet = "ABCDEFGHJKMNPQRSTVWXYZ23456789"
	accessCodeLength   = 6
	accessCodeAttempts = 5
)

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

// CreateProject ports ProjectService.create.
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

// JoinProject ports the join-by-code flow.
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
	doc := domain.DiagramDocument{
		SchemaVersion: 1, ID: &id,
		Version:      raw.Version,
		ReviewNumber:  raw.ReviewNumber,
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

// hydrateReview mirrors the live checkpoint count AND the per-diagram
// review number into the response document so clients can echo them
// back on the next PUT or /checkpoints POST.
//
// The stored JSON document carries the pre-bump review number for
// audit purposes; the wire is the live one. Both fields are always
// overridden with the server-stored values so concurrent reads and
// writes converge on a single timeline.
func (s *Service) hydrateReview(ctx context.Context, doc *domain.DiagramDocument) error {
	if doc.ID == nil {
		return nil
	}
	review, err := s.store.CurrentReview(ctx, *doc.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return err
	}
	doc.ReviewNumber = review
	version, err := s.store.CurrentVersion(ctx, *doc.ID)
	if err != nil {
		return err
	}
	doc.Version = version
	return nil
}

func toDocument(payload []byte) (domain.DiagramDocument, error) {
	var doc domain.DiagramDocument
	if err := json.Unmarshal(payload, &doc); err != nil {
		return domain.DiagramDocument{}, err
	}
	return doc, nil
}

func hydrateCheckpointVersion(ctx context.Context, st store.Store, doc *domain.DiagramDocument, v *store.VersionRecord) error {
	if v != nil {
		doc.Version = v.Number
		doc.ReviewNumber = v.ReviewNumber
		return nil
	}
	// Fallback when the store returns only the diagrams row: pull the
	// live counters so the response carries the bumped baseline.
	review, err := st.CurrentReview(ctx, *doc.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return err
	}
	version, err := st.CurrentVersion(ctx, *doc.ID)
	if err != nil {
		return err
	}
	doc.Version = version
	doc.ReviewNumber = review
	return nil
}

// CreateDiagram ports DiagramService.create: fresh id, manual seed of the
// diagrams row (review_number = 1) and an implicit "Initial revision"
// checkpoint (review_number = 2) so the new diagram has a stable starting
// point. Both happen inside AppendCheckpoint's serialized transaction so
// concurrent calls cannot both see a missing row and double-insert.
//
// SaveDocument is autosave-only; the create path never touches it. The
// HTTP layer maps success to 201.
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
	diagram := store.DiagramRecord{
		ID: id, ProjectID: projectID, Name: doc.Name,
		Document: payload, CreatedBy: userID, UpdatedAt: now,
	}
	if err := requireNotExists(ctx, s.store, "create diagram", projectID, id); err != nil {
		return domain.DiagramDocument{}, err
	}
	version, err := s.store.AppendCheckpoint(ctx, diagram, nil, strPtr("Initial revision"))
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	if err := hydrateCheckpointVersion(ctx, s.store, &doc, &version); err != nil {
		return domain.DiagramDocument{}, err
	}
	s.emitChanged(projectID, id, domain.DiagramChangedEvent{
		ActorID:      userID,
		ReviewNumber: version.ReviewNumber,
		Version:      version.Number,
		Kind:         "checkpoint",
		Message:      "Initial revision",
	})
	return doc, nil
}

// requireNotExists guards CreateDiagram against updating a row that the
// store already sees. Without this check, AppendCheckpoint(nil) returns
// ErrReviewMismatch{Current: live} for an existing diagram and our
// would-be-create collides with the bumped state. A NotFound lookup here
// is the only "ok" outcome; anything else is an error the caller can
// surface as 409 or 500.
func requireNotExists(ctx context.Context, st store.Store, op, projectID, diagramID string) error {
	_, err := st.FindDiagram(ctx, projectID, diagramID)
	if err == nil {
		return ErrCreateCollision{Op: op, DiagramID: diagramID}
	}
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	return err
}

// ErrCreateCollision is returned when CreateDiagram finds an existing
// row for the requested id. AppendCheckpoint(nil) would otherwise refuse
// with ErrReviewMismatch, which is the wrong status code for a create
// path; the router maps ErrCreateCollision to 409 too.
type ErrCreateCollision struct {
	Op       string
	DiagramID string
}

func (e ErrCreateCollision) Error() string {
	return e.Op + ": diagram " + e.DiagramID + " already exists"
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
	if err := s.hydrateReview(ctx, &doc); err != nil {
		return domain.DiagramDocument{}, err
	}
	return doc, nil
}

// ListDiagrams ports DiagramService.list (updatedAt desc). Each summary
// carries the current explicit checkpoint count (so the list row can show
// "v3 · r12" without an extra round-trip) and the live review number.
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
			UpdatedAt:    domain.FormatInstant(r.UpdatedAt),
			Version:      version,
			ReviewNumber: r.ReviewNumber,
		})
	}
	return out, nil
}

// UpdateDiagram is the autosave path: it writes the new working document
// using a CAS gate on the per-diagram review number, never appending a
// version row. baseline resolution is the contract's center:
//
//   - body.reviewNumber > 0 OR header > 0 → positive baseline. The
//     UPDATE WHERE review_number = $baseline is the SINGLE atomic
//     decision. A stale positive baseline returns 409 carrying the live
//     server document so the client can retry with the bumped value.
//     There is NO pre-read CAS that returns 409 before the UPDATE runs;
//     the service trusts the UPDATE's atomic answer.
//
//   - body.reviewNumber == 0 AND header == 0 / absent → LEGACY WRITE.
//     The service reads the LIVE review number ONCE and uses it as the
//     CAS baseline. The SQL UPDATE always has a useful WHERE clause and
//     the save never 409s. Two concurrent legacy writers race for the
//     same baseline; one wins and bumps, the other loses and retries.
//     The losing retry is a SECOND legacy write with the new live value.
//
// ifMatch pins the explicit-checkpoint counter; if a positive check
// mismatches the live current_version the service refuses BEFORE the
// UPDATE so a concurrent checkpoint cannot be silently absorbed.
func (s *Service) UpdateDiagram(ctx context.Context, projectID, diagramID, userID string, raw domain.DiagramDocument, ifMatch *int, ifReview *int64) (domain.DiagramDocument, error) {
	if err := s.requireMember(ctx, projectID, userID); err != nil {
		return domain.DiagramDocument{}, err
	}
	currentVersion, err := s.store.CurrentVersion(ctx, diagramID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return domain.DiagramDocument{}, err
	}
	currentReview, err := s.store.CurrentReview(ctx, diagramID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.DiagramDocument{}, NotFoundError{Message: "Diagram not found"}
		}
		return domain.DiagramDocument{}, err
	}
	if ifMatch != nil && currentVersion != -1 && *ifMatch != currentVersion {
		serverDoc, gerr := s.GetDiagram(ctx, projectID, diagramID, userID)
		if gerr != nil {
			return domain.DiagramDocument{}, gerr
		}
		return domain.DiagramDocument{}, ConflictError{
			Current: serverDoc, Expected: raw.Version, ExpectedReview: ifReviewAsRaw(ifReview), ActualReview: currentReview,
		}
	}
	resolvedReview := int64(0)
	if ifReview != nil {
		resolvedReview = *ifReview
	}
	if resolvedReview == 0 && raw.ReviewNumber > 0 {
		resolvedReview = raw.ReviewNumber
	}
	baseline := &currentReview
	if resolvedReview > 0 {
		baseline = &resolvedReview
	}
	doc, payload, err := normalize(raw, diagramID)
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	now := time.Now().UTC()
	bump, err := s.store.SaveDocument(ctx,
		store.DiagramRecord{ID: diagramID, ProjectID: projectID, Name: doc.Name, Document: payload, CreatedBy: userID, UpdatedAt: now},
		baseline,
	)
	if err != nil {
		if current, ok := store.AsReviewMismatch(err); ok {
			if current == 0 {
				return domain.DiagramDocument{}, NotFoundError{Message: "Diagram not found"}
			}
			serverDoc, gerr := s.GetDiagram(ctx, projectID, diagramID, userID)
			if gerr != nil {
				return domain.DiagramDocument{}, gerr
			}
			return domain.DiagramDocument{}, ConflictError{
				Current: serverDoc, Expected: raw.Version, ExpectedReview: *baseline, ActualReview: current,
			}
		}
		return domain.DiagramDocument{}, err
	}
	doc.ReviewNumber = bump
	doc.Version = currentVersion
	s.emitChanged(projectID, diagramID, domain.DiagramChangedEvent{
		ActorID:      userID,
		ReviewNumber: bump,
		Version:      doc.Version,
		Kind:         "working-document",
	})
	return doc, nil
}

// ifReviewAsRaw is a tiny helper that lets a 0-default header explain the
// ConflictError's ExpectedReview field. Returns 0 when the header was
// unused; never dereferences nil.
func ifReviewAsRaw(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// CreateCheckpoint is the explicit-save path: it bumps the per-diagram review
// counter, assigns a fresh diagram_versions row with version_number = max+1,
// stamps the actor as created_by, and stores the optional message supplied by
// the client. It is the only call that grows diagram_versions. The CAS
// contract mirrors UpdateDiagram: a positive baseline (header or body) is a
// real CAS; zero or absent falls through to the live baseline so legacy
// checkpoints never 409.
func (s *Service) CreateCheckpoint(ctx context.Context, projectID, diagramID, userID string, raw domain.DiagramDocument, message *string, ifMatch *int, ifReview *int64) (domain.DiagramVersion, error) {
	if err := s.requireMember(ctx, projectID, userID); err != nil {
		return domain.DiagramVersion{}, err
	}
	if _, err := s.store.FindDiagram(ctx, projectID, diagramID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.DiagramVersion{}, NotFoundError{Message: "Diagram not found"}
		}
		return domain.DiagramVersion{}, err
	}
	currentVersion, err := s.store.CurrentVersion(ctx, diagramID)
	if err != nil {
		return domain.DiagramVersion{}, err
	}
	currentReview, err := s.store.CurrentReview(ctx, diagramID)
	if err != nil {
		return domain.DiagramVersion{}, err
	}
	if ifMatch != nil && *ifMatch != currentVersion {
		serverDoc, err := s.GetDiagram(ctx, projectID, diagramID, userID)
		if err != nil {
			return domain.DiagramVersion{}, err
		}
		return domain.DiagramVersion{}, ConflictError{
			Current: serverDoc, Expected: raw.Version, ExpectedReview: ifReviewAsRaw(ifReview), ActualReview: currentReview,
		}
	}
	resolvedReview := int64(0)
	if ifReview != nil {
		resolvedReview = *ifReview
	}
	if resolvedReview == 0 && raw.ReviewNumber > 0 {
		resolvedReview = raw.ReviewNumber
	}
	baseline := &currentReview
	if resolvedReview > 0 {
		baseline = &resolvedReview
	}
	if resolvedReview > 0 && resolvedReview != currentReview {
		serverDoc, err := s.GetDiagram(ctx, projectID, diagramID, userID)
		if err != nil {
			return domain.DiagramVersion{}, err
		}
		return domain.DiagramVersion{}, ConflictError{
			Current: serverDoc, Expected: raw.Version, ExpectedReview: resolvedReview, ActualReview: currentReview,
		}
	}
	doc, payload, err := normalize(raw, diagramID)
	if err != nil {
		return domain.DiagramVersion{}, err
	}
	now := time.Now().UTC()
	diagram := store.DiagramRecord{ID: diagramID, ProjectID: projectID, Name: doc.Name, Document: payload, CreatedBy: userID, UpdatedAt: now}
	version, err := s.store.AppendCheckpoint(ctx, diagram, baseline, message)
	if err != nil {
		if current, ok := store.AsReviewMismatch(err); ok {
			serverDoc, gerr := s.GetDiagram(ctx, projectID, diagramID, userID)
			if gerr != nil {
				return domain.DiagramVersion{}, gerr
			}
			return domain.DiagramVersion{}, ConflictError{
				Current: serverDoc, Expected: raw.Version, ExpectedReview: *baseline, ActualReview: current,
			}
		}
		if errors.Is(err, store.ErrNotFound) {
			return domain.DiagramVersion{}, NotFoundError{Message: "Diagram not found"}
		}
		return domain.DiagramVersion{}, err
	}
	reflected, err := toDocument(version.Document)
	if err != nil {
		return domain.DiagramVersion{}, err
	}
	reflected.Version = version.Number
	reflected.ReviewNumber = version.ReviewNumber
	s.emitChanged(projectID, diagramID, domain.DiagramChangedEvent{
		ActorID:      userID,
		ReviewNumber: version.ReviewNumber,
		Version:      version.Number,
		Kind:         "checkpoint",
		Message:      derefMessage(message),
	})
	return domain.DiagramVersion{
		ID: version.ID, VersionNumber: version.Number,
		ReviewNumber: version.ReviewNumber,
		CreatedAt:    domain.FormatInstant(version.CreatedAt),
		CreatedBy:    version.CreatedBy,
		Message:      version.Message,
		Document:     reflected,
	}, nil
}

func derefMessage(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ListVersions ports DiagramService.versions (versionNumber desc). The
// membership gate and the 404 for missing diagrams are inherited from
// GetDiagram. Each row carries the actor and the optional message so the
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
		doc.ReviewNumber = r.ReviewNumber
		out = append(out, domain.DiagramVersion{
			ID: r.ID, VersionNumber: r.Number,
			ReviewNumber: r.ReviewNumber,
			CreatedAt:    domain.FormatInstant(r.CreatedAt),
			CreatedBy:    r.CreatedBy,
			Message:      r.Message,
			Document:     doc,
		})
	}
	return out, nil
}

// RestoreDiagram ports DiagramService.restore: the old version payload
// becomes the new working document and a new explicit checkpoint records the
// restore as a deliberate save, carrying "Restored v{N}" so the history shows
// what produced the change. The safety baseline is the LATEST review number
// because RestoreDiagram is a privileged action that should pick up the live
// server state, not the browser's possibly-stale copy.
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
	liveVersion, err := s.store.CurrentVersion(ctx, diagramID)
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	snapshot.Version = liveVersion
	live, err := s.store.CurrentReview(ctx, diagramID)
	if err != nil {
		return domain.DiagramDocument{}, err
	}
	snapshot.ReviewNumber = live
	message := fmt.Sprintf("Restored v%d", number)
	// Pass nil for ifMatch (no version check; restore picks up the live
	// baseline) and &live for ifReview so the bump is gated by the live
	// review number rather than the version row number.
	if _, err := s.CreateCheckpoint(ctx, projectID, diagramID, userID, snapshot, &message, nil, &live); err != nil {
		return domain.DiagramDocument{}, err
	}
	return s.GetDiagram(ctx, projectID, diagramID, userID)
}
