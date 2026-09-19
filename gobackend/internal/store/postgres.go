// Postgres Store implementation over pgx (GOBE-02).
//
// Every query mirrors the Spring Data derived query it replaces:
//   - findByEmailIgnoreCase -> LOWER(email) lookup
//   - findByTokenHashAndRevokedAtIsNull (+ expiry in service, like userFor)
//   - findByUserId memberships joined to projects + countByProjectId
//   - findByProjectIdOrderByUpdatedAtDesc
//   - findByIdAndProjectId
//   - findFirstByDiagramIdOrderByVersionNumberDesc for max+1 numbering
//   - findByDiagramIdOrderByVersionNumberDesc / findByDiagramIdAndVersionNumber
//
// SaveDiagram runs the diagram upsert and the version insert in one
// transaction, mirroring @Transactional save()/restore().

package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres is a Store backed by PostgreSQL with JSONB documents.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres returns a Store over pool.
func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

// OpenPool connects using a postgres:// (or postgresql://) URL.
func OpenPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

func (p *Postgres) FindUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := p.pool.QueryRow(ctx, `SELECT id, display_name, email, password_hash
  FROM users WHERE LOWER(email) = LOWER($1)`, email).Scan(&u.ID, &u.DisplayName, &u.Email, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (p *Postgres) CreateToken(ctx context.Context, tok TokenRecord) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at)
  VALUES ($1, $2, $3, $4, NULL)`, tok.ID, tok.UserID, tok.Hash, tok.ExpiresAt)
	return err
}

func (p *Postgres) FindToken(ctx context.Context, hash string) (TokenRecord, error) {
	var tok TokenRecord
	err := p.pool.QueryRow(ctx, `SELECT id, user_id, token_hash, expires_at, revoked_at
  FROM refresh_tokens WHERE token_hash = $1 AND revoked_at IS NULL`, hash).
		Scan(&tok.ID, &tok.UserID, &tok.Hash, &tok.ExpiresAt, &tok.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenRecord{}, ErrNotFound
		}
		return TokenRecord{}, err
	}
	return tok, nil
}

func (p *Postgres) AssignedProjects(ctx context.Context, userID string) ([]AssignedProject, error) {
	rows, err := p.pool.Query(ctx, `SELECT p.id, p.name, p.description, m.role,
    (SELECT COUNT(*) FROM diagrams d WHERE d.project_id = p.id)
  FROM project_memberships m JOIN projects p ON p.id = m.project_id
  WHERE m.user_id = $1 ORDER BY p.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AssignedProject
	for rows.Next() {
		var r AssignedProject
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Role, &r.DiagramCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (p *Postgres) IsMember(ctx context.Context, projectID, userID string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(ctx, `SELECT EXISTS(
  SELECT 1 FROM project_memberships WHERE project_id = $1 AND user_id = $2)`, projectID, userID).Scan(&exists)
	return exists, err
}

func (p *Postgres) SaveDiagram(ctx context.Context, d DiagramRecord, v VersionRecord) (VersionRecord, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return VersionRecord{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO diagrams (id, project_id, name, document, created_by, updated_at)
  VALUES ($1, $2, $3, $4::jsonb, $5, $6)
  ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, document = EXCLUDED.document, updated_at = EXCLUDED.updated_at`,
		d.ID, d.ProjectID, d.Name, string(d.Document), d.CreatedBy, d.UpdatedAt); err != nil {
		return VersionRecord{}, err
	}
	var next int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version_number), 0) + 1
  FROM diagram_versions WHERE diagram_id = $1`, d.ID).Scan(&next); err != nil {
		return VersionRecord{}, err
	}
	v.Number = next
	v.DiagramID = d.ID
	v.Document = append([]byte(nil), d.Document...)
	if _, err := tx.Exec(ctx, `INSERT INTO diagram_versions (id, diagram_id, version_number, document, created_by, created_at)
  VALUES ($1, $2, $3, $4::jsonb, $5, $6)`,
		v.ID, v.DiagramID, v.Number, string(v.Document), d.CreatedBy, v.CreatedAt); err != nil {
		return VersionRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return VersionRecord{}, err
	}
	return v, nil
}

func (p *Postgres) FindDiagram(ctx context.Context, projectID, diagramID string) (DiagramRecord, error) {
	var d DiagramRecord
	var payload string
	err := p.pool.QueryRow(ctx, `SELECT id, project_id, name, document::text, updated_at
  FROM diagrams WHERE id = $1 AND project_id = $2`, diagramID, projectID).
		Scan(&d.ID, &d.ProjectID, &d.Name, &payload, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DiagramRecord{}, ErrNotFound
		}
		return DiagramRecord{}, err
	}
	d.Document = []byte(payload)
	return d, nil
}

func (p *Postgres) ListDiagrams(ctx context.Context, projectID string) ([]DiagramRecord, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, project_id, name, updated_at
  FROM diagrams WHERE project_id = $1 ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DiagramRecord
	for rows.Next() {
		var d DiagramRecord
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.Name, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (p *Postgres) ListVersions(ctx context.Context, diagramID string) ([]VersionRecord, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, diagram_id, version_number, document::text, created_at
  FROM diagram_versions WHERE diagram_id = $1 ORDER BY version_number DESC`, diagramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VersionRecord
	for rows.Next() {
		var v VersionRecord
		var payload string
		if err := rows.Scan(&v.ID, &v.DiagramID, &v.Number, &payload, &v.CreatedAt); err != nil {
			return nil, err
		}
		v.Document = []byte(payload)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (p *Postgres) FindVersion(ctx context.Context, diagramID string, number int) (VersionRecord, error) {
	var v VersionRecord
	var payload string
	err := p.pool.QueryRow(ctx, `SELECT id, diagram_id, version_number, document::text, created_at
  FROM diagram_versions WHERE diagram_id = $1 AND version_number = $2`, diagramID, number).
		Scan(&v.ID, &v.DiagramID, &v.Number, &payload, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return VersionRecord{}, ErrNotFound
		}
		return VersionRecord{}, err
	}
	v.Document = []byte(payload)
	return v, nil
}
