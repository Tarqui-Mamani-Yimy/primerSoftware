// In-memory Store for hermetic unit tests (no database required).
//
// The memory store mirrors the Postgres store's optimistic-concurrency
// contract so service-level tests reproduce the same 409 surface that
// production code does. The mutex holdings are kept short; each public
// method locks once.

package store

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryStore is a mutex-guarded Store implementation for tests.
type MemoryStore struct {
	mu          sync.Mutex
	users       map[string]User
	projects    map[string]projectRow
	memberships map[string]string // projectID + "\x00" + userID -> role
	tokens      map[string]TokenRecord
	diagrams    map[string]DiagramRecord
	versions    map[string][]VersionRecord // diagramID -> rows (ascending)
}

type projectRow struct {
	id          string
	name        string
	description *string
	accessCode  string
	ownerID     string
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:       make(map[string]User),
		projects:    make(map[string]projectRow),
		memberships: make(map[string]string),
		tokens:      make(map[string]TokenRecord),
		diagrams:    make(map[string]DiagramRecord),
		versions:    make(map[string][]VersionRecord),
	}
}

// SeedUser inserts a user fixture.
func (m *MemoryStore) SeedUser(u User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.ID] = u
}

// SeedProject inserts a project fixture.
func (m *MemoryStore) SeedProject(id, name string, description *string) {
	m.SeedProjectWithCode(id, name, description, "", "")
}

// SeedProjectWithCode inserts a project fixture carrying a classroom access
// code and owner, for create/join coverage. A non-empty owner also gets the
// OWNER membership row, mirroring CreateProject.
func (m *MemoryStore) SeedProjectWithCode(id, name string, description *string, accessCode, ownerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projects[id] = projectRow{
		id: id, name: name, description: description, accessCode: accessCode, ownerID: ownerID,
	}
	if ownerID != "" {
		m.memberships[id+"\x00"+ownerID] = "OWNER"
	}
}

// SeedMember inserts a membership fixture.
func (m *MemoryStore) SeedMember(projectID, userID, role string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memberships[projectID+"\x00"+userID] = role
}

// SeedDiagramWithReview primes the diagram row with a known review_number
// baseline (default 0) so optimistic-concurrency tests can target predictable
// numbers rather than chasing bumped values.
func (m *MemoryStore) SeedDiagramWithReview(d DiagramRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.diagrams[d.ID] = d
}

func (m *MemoryStore) FindUserByEmail(_ context.Context, email string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (m *MemoryStore) CreateToken(_ context.Context, tok TokenRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[tok.Hash] = tok
	return nil
}

func (m *MemoryStore) FindToken(_ context.Context, hash string) (TokenRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok, ok := m.tokens[hash]
	if !ok {
		return TokenRecord{}, ErrNotFound
	}
	return tok, nil
}

func (m *MemoryStore) AssignedProjects(_ context.Context, userID string) ([]AssignedProject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []AssignedProject
	for key, role := range m.memberships {
		parts := strings.SplitN(key, "\x00", 2)
		if len(parts) != 2 || parts[1] != userID {
			continue
		}
		p, ok := m.projects[parts[0]]
		if !ok {
			continue
		}
		count := 0
		for _, d := range m.diagrams {
			if d.ProjectID == p.id {
				count++
			}
		}
		out = append(out, AssignedProject{
			ID: p.id, Name: p.name, Description: p.description,
			Role: role, DiagramCount: count,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *MemoryStore) IsMember(_ context.Context, projectID, userID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.memberships[projectID+"\x00"+userID]
	return ok, nil
}

func (m *MemoryStore) FindProjectByAccessCode(_ context.Context, accessCode string) (ProjectRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.projects {
		if p.accessCode != "" && strings.EqualFold(p.accessCode, accessCode) {
			return ProjectRecord{
				ID: p.id, Name: p.name, Description: p.description,
				AccessCode: p.accessCode, OwnerID: p.ownerID,
			}, nil
		}
	}
	return ProjectRecord{}, ErrNotFound
}

func (m *MemoryStore) CreateProject(_ context.Context, p ProjectRecord) (ProjectRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.projects {
		if existing.accessCode != "" && strings.EqualFold(existing.accessCode, p.AccessCode) {
			return ProjectRecord{}, ErrAccessCodeTaken
		}
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	m.projects[p.ID] = projectRow{
		id: p.ID, name: p.Name, description: p.Description,
		accessCode: p.AccessCode, ownerID: p.OwnerID,
	}
	m.memberships[p.ID+"\x00"+p.OwnerID] = "OWNER"
	return p, nil
}

func (m *MemoryStore) JoinProject(_ context.Context, projectID, userID string) (AssignedProject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[projectID]
	if !ok {
		return AssignedProject{}, ErrNotFound
	}
	key := projectID + "\x00" + userID
	role, ok := m.memberships[key]
	if !ok {
		role = "COLLABORATOR"
		m.memberships[key] = role
	}
	count := 0
	for _, d := range m.diagrams {
		if d.ProjectID == p.id {
			count++
		}
	}
	return AssignedProject{
		ID: p.id, Name: p.name, Description: p.description,
		Role: role, DiagramCount: count,
	}, nil
}

// SaveDocument mirrors autosave-only: a strict-CAS UPDATE under a positive
// baseline. Create paths do NOT go here anymore; AppendCheckpoint owns the
// diagrams row lifecycle. A nil expectedReview is rejected so autosave can
// never recreate a diagram row that never existed.
func (m *MemoryStore) SaveDocument(_ context.Context, d DiagramRecord, expectedReview *int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if expectedReview == nil {
		return 0, errors.New("store: SaveDocument requires a non-nil expectedReview; AppendCheckpoint owns creates")
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = time.Now().UTC()
	}
	existing, ok := m.diagrams[d.ID]
	if !ok {
		return 0, ErrReviewMismatch{Current: 0}
	}
	if existing.ReviewNumber != *expectedReview {
		return existing.ReviewNumber, ErrReviewMismatch{Current: existing.ReviewNumber}
	}
	// preserve created_by; autosave never re-stamps the diagram creator.
	if existing.CreatedBy != "" {
		d.CreatedBy = existing.CreatedBy
	}
	d.ReviewNumber = existing.ReviewNumber + 1
	m.diagrams[d.ID] = d
	return d.ReviewNumber, nil
}

// AppendCheckpoint mirrors Postgres: create-or-bump with strict CAS or
// nil-baseline create. It owns the diagrams row lifecycle together with
// the diagram_versions insert under the same lock so concurrent writers
// serialize cleanly.
//
//   - nil baseline + row missing: synthesise diagrams row at
//     review_number = 1; version row also stamped at review_number = 1
//     so the implicit "Initial revision" lives at the same timeline as
//     a CREATE step that doubled as a checkpoint write.
//   - nil baseline + row present: ErrReviewMismatch with the live
//     current_review so the service can retry.
//   - positive baseline + row present + current==baseline: bump + 1 on
//     both the diagrams row and the version row.
//   - positive baseline + row absent: ErrReviewMismatch{Current: 0}.
// created_by is preserved across the bump so the diagram creator stays
// attributed to the original author.
func (m *MemoryStore) AppendCheckpoint(_ context.Context, d DiagramRecord, expectedReview *int64, message *string) (VersionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = time.Now().UTC()
	}
	current, ok := m.diagrams[d.ID]
	var nextReview int64
	switch {
	case expectedReview == nil && !ok:
		current = d
		current.ReviewNumber = 1
		m.diagrams[d.ID] = current
		nextReview = 1
	case expectedReview != nil && ok && current.ReviewNumber == *expectedReview:
		nextReview = current.ReviewNumber + 1
		if d.CreatedBy != "" && current.CreatedBy == "" {
			current.CreatedBy = d.CreatedBy
		}
		if d.Document != nil {
			current.Document = append([]byte(nil), d.Document...)
		}
		if d.Name != "" {
			current.Name = d.Name
		}
		if !d.UpdatedAt.IsZero() {
			current.UpdatedAt = d.UpdatedAt
		}
		current.ReviewNumber = nextReview
		m.diagrams[d.ID] = current
	case expectedReview == nil && ok:
		return VersionRecord{}, ErrReviewMismatch{Current: current.ReviewNumber}
	case expectedReview != nil && !ok:
		return VersionRecord{}, ErrReviewMismatch{Current: 0}
	case expectedReview != nil && ok && current.ReviewNumber != *expectedReview:
		return VersionRecord{}, ErrReviewMismatch{Current: current.ReviewNumber}
	}

	next := 1
	for _, existing := range m.versions[d.ID] {
		if existing.Number >= next {
			next = existing.Number + 1
		}
	}
	v := VersionRecord{
		ID:           NewUUID(),
		DiagramID:    d.ID,
		Number:       next,
		ReviewNumber: nextReview,
		Document:     append([]byte(nil), d.Document...),
		CreatedBy:    d.CreatedBy,
		CreatedAt:    time.Now().UTC(),
		Message:      message,
	}
	m.versions[d.ID] = append(m.versions[d.ID], v)
	return v, nil
}

// CurrentReview returns the diagrams row's current review_number (0 when no
// row exists yet).
func (m *MemoryStore) CurrentReview(_ context.Context, diagramID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.diagrams[diagramID]
	if !ok {
		return 0, nil
	}
	return d.ReviewNumber, nil
}

// CurrentVersion returns the highest version_number for a diagram (0 when no
// checkpoint row exists yet).
func (m *MemoryStore) CurrentVersion(_ context.Context, diagramID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	max := 0
	for _, v := range m.versions[diagramID] {
		if v.Number > max {
			max = v.Number
		}
	}
	return max, nil
}

func (m *MemoryStore) FindDiagram(_ context.Context, projectID, diagramID string) (DiagramRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.diagrams[diagramID]
	if !ok || d.ProjectID != projectID {
		return DiagramRecord{}, ErrNotFound
	}
	return d, nil
}

func (m *MemoryStore) ListDiagrams(_ context.Context, projectID string) ([]DiagramRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []DiagramRecord
	for _, d := range m.diagrams {
		if d.ProjectID == projectID {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func (m *MemoryStore) ListVersions(_ context.Context, diagramID string) ([]VersionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := append([]VersionRecord(nil), m.versions[diagramID]...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Number > rows[j].Number })
	return rows, nil
}

func (m *MemoryStore) FindVersion(_ context.Context, diagramID string, number int) (VersionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range m.versions[diagramID] {
		if v.Number == number {
			return v, nil
		}
	}
	return VersionRecord{}, ErrNotFound
}
