// In-memory Store for hermetic unit tests (no database required).

package store

import (
	"context"
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
		if parts[1] != userID {
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

// SaveWorkingDocument mirrors the Postgres implementation: it upserts the
// diagrams row without writing to diagram_versions, since autosave must not
// pollute the explicit-checkpoint history.
func (m *MemoryStore) SaveWorkingDocument(_ context.Context, d DiagramRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = time.Now().UTC()
	}
	m.diagrams[d.ID] = d
	return nil
}

// AppendCheckpoint mirrors the Postgres implementation: it assigns
// version_number = max+1, stamps CreatedBy with the actor (authorship), and
// stores the optional Message verbatim.
func (m *MemoryStore) AppendCheckpoint(_ context.Context, d DiagramRecord, message *string) (VersionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	next := 1
	for _, existing := range m.versions[d.ID] {
		if existing.Number >= next {
			next = existing.Number + 1
		}
	}
	current, ok := m.diagrams[d.ID]
	if !ok || current.ID == "" {
		m.diagrams[d.ID] = d
	} else if d.CreatedBy != "" && current.CreatedBy == "" {
		current.CreatedBy = d.CreatedBy
		m.diagrams[d.ID] = current
	}
	v := VersionRecord{
		ID:        NewUUID(),
		DiagramID: d.ID,
		Number:    next,
		Document:  append([]byte(nil), d.Document...),
		CreatedBy: d.CreatedBy,
		CreatedAt: time.Now().UTC(),
		Message:   message,
	}
	m.versions[d.ID] = append(m.versions[d.ID], v)
	return v, nil
}

// CurrentVersion mirrors Postgres: the highest version_number for a diagram
// (0 when no checkpoint row exists yet). Tests use it to assert the Version
// the service mirrors onto DiagramDocument after save/restore.
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
