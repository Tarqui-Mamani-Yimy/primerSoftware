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
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projects[id] = projectRow{id: id, name: name, description: description}
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

func (m *MemoryStore) SaveDiagram(_ context.Context, d DiagramRecord, v VersionRecord) (VersionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d.UpdatedAt = time.Now().UTC()
	m.diagrams[d.ID] = d
	next := 1
	for _, existing := range m.versions[d.ID] {
		if existing.Number >= next {
			next = existing.Number + 1
		}
	}
	v.Number = next
	v.DiagramID = d.ID
	v.Document = append([]byte(nil), d.Document...)
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now().UTC()
	}
	m.versions[d.ID] = append(m.versions[d.ID], v)
	return v, nil
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
