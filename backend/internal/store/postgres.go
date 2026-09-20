// Postgres Store implementation over pgx (GOBE-02).
//
// Every query mirrors the Spring Data derived query it replaces:
//   - findByEmailIgnoreCase -> LOWER(email) lookup
//   - findByTokenHashAndRevokedAtIsNull (+ expiry in service, like userFor)
//   - findByUserId memberships joined to projects + countByProjectId
//   - findByAccessCodeIgnoreCase -> UPPER(access_code) lookup
//   - create() project + OWNER membership, join() COLLABORATOR membership
//   - findByProjectIdOrderByUpdatedAtDesc
//   - findByIdAndProjectId
//   - findFirstByDiagramIdOrderByVersionNumberDesc for max+1 numbering
//   - findByDiagramIdOrderByVersionNumberDesc / findByDiagramIdAndVersionNumber
//
// Autosave and explicit checkpoints are split into SaveDocument and
// AppendCheckpoint so untrusted autosave traffic never grows the version
// history. Versioning controls live entirely in AppendCheckpoint: it
// owns the diagrams row lifecycle together with the diagram_versions
// insert so the two land atomically inside one SERIALIZABLE tx.

package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (p *Postgres) FindProjectByAccessCode(ctx context.Context, accessCode string) (ProjectRecord, error) {
	var r ProjectRecord
	err := p.pool.QueryRow(ctx, `SELECT id, name, description, access_code, owner_id, created_at
  FROM projects WHERE UPPER(access_code) = UPPER($1)`, accessCode).
		Scan(&r.ID, &r.Name, &r.Description, &r.AccessCode, &r.OwnerID, &r.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectRecord{}, ErrNotFound
		}
		return ProjectRecord{}, err
	}
	return r, nil
}

// CreateProject inserts the project and its OWNER membership in one
// transaction, mirroring @Transactional create(). An access-code collision is
// reported as ErrAccessCodeTaken so the service can retry with a fresh code.
func (p *Postgres) CreateProject(ctx context.Context, project ProjectRecord) (ProjectRecord, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return ProjectRecord{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO projects (id, name, description, access_code, owner_id)
  VALUES ($1, $2, $3, $4, $5)`,
		project.ID, project.Name, project.Description, project.AccessCode, project.OwnerID); err != nil {
		if isAccessCodeConflict(err) {
			return ProjectRecord{}, ErrAccessCodeTaken
		}
		return ProjectRecord{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO project_memberships (project_id, user_id, role)
  VALUES ($1, $2, 'OWNER') ON CONFLICT (project_id, user_id) DO NOTHING`,
		project.ID, project.OwnerID); err != nil {
		return ProjectRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProjectRecord{}, err
	}
	return project, nil
}

// JoinProject is idempotent: ON CONFLICT DO NOTHING keeps an existing role
// (an owner who re-enters their own code stays OWNER), and the returned row is
// the same shape the dashboard list uses.
func (p *Postgres) JoinProject(ctx context.Context, projectID, userID string) (AssignedProject, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return AssignedProject{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO project_memberships (project_id, user_id, role)
  VALUES ($1, $2, 'COLLABORATOR') ON CONFLICT (project_id, user_id) DO NOTHING`,
		projectID, userID); err != nil {
		return AssignedProject{}, err
	}
	var r AssignedProject
	err = tx.QueryRow(ctx, `SELECT p.id, p.name, p.description, m.role,
    (SELECT COUNT(*) FROM diagrams d WHERE d.project_id = p.id)
  FROM project_memberships m JOIN projects p ON p.id = m.project_id
  WHERE m.project_id = $1 AND m.user_id = $2`, projectID, userID).
		Scan(&r.ID, &r.Name, &r.Description, &r.Role, &r.DiagramCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AssignedProject{}, ErrNotFound
		}
		return AssignedProject{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AssignedProject{}, err
	}
	return r, nil
}

// isAccessCodeConflict reports whether err is a unique violation on
// projects.access_code, as opposed to any other constraint.
func isAccessCodeConflict(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "access_code")
}

// SaveDocument is the autosave-only path: a strict-CAS UPDATE under a
// positive expectedReview baseline. CreateDiagram no longer calls this;
// it forwards to AppendCheckpoint instead. A nil expectedReview is
// rejected so the autosave layer cannot accidentally recreate a
// diagram row that never existed.
//
// The diagrams row carries the autosave snapshot. diagram_versions is
// untouched; explicit checkpoints own the version history.
func (p *Postgres) SaveDocument(ctx context.Context, d DiagramRecord, expectedReview *int64) (int64, error) {
	if expectedReview == nil {
		return 0, errors.New("store: SaveDocument requires a non-nil expectedReview; AppendCheckpoint owns creates")
	}
	var newReview int64
	err := p.pool.QueryRow(ctx, `UPDATE diagrams
  SET name = $3, document = $4::jsonb, updated_at = $5, review_number = review_number + 1
  WHERE id = $1 AND project_id = $2 AND review_number = $6
  RETURNING review_number`,
		d.ID, d.ProjectID, d.Name, string(d.Document), d.UpdatedAt, *expectedReview).Scan(&newReview)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			current, lookupErr := p.CurrentReview(ctx, d.ID)
			if lookupErr != nil {
				return 0, lookupErr
			}
			return 0, ErrReviewMismatch{Current: current}
		}
		return 0, err
	}
	return newReview, nil
}

// AppendCheckpoint is the create-or-bump path for the diagram version
// timeline. It owns the diagrams row lifecycle together with the
// diagram_versions insert so the two land atomically:
//
//   - nil baseline (Service.CreateDiagram): synthesises a fresh row
//     when none exists, or bumps an existing row by one. The freshly
//     inserted diagrams row starts at review_number = 1; the version
//     row is stamped with review_number = 2 so concurrent readers can
//     detect create-vs-bump without ambiguity.
//   - non-nil baseline (Service.CreateCheckpoint / RestoreDiagram /
//     internal append paths): strict-CAS UPDATE. Stale baseline
//     returns ErrReviewMismatch with the live current_review, never
//     the bumped value.
//
// created_by on the version row is stamped with the actor (preserving
// authorship even when the actor differs from the diagram owner);
// message is the optional note supplied by the client. SELECT FOR
// UPDATE locks the diagrams row inside the tx so two concurrent writers
// cannot both see the same current_review and both satisfy CAS.
func (p *Postgres) AppendCheckpoint(ctx context.Context, d DiagramRecord, expectedReview *int64, message *string) (VersionRecord, error) {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return VersionRecord{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current int64
	var rowExists bool
	if err := tx.QueryRow(ctx, `SELECT review_number FROM diagrams WHERE id = $1 FOR UPDATE`, d.ID).Scan(&current); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return VersionRecord{}, err
		}
		rowExists = false
	} else {
		rowExists = true
	}

	var nextReview int64
	switch {
	case expectedReview == nil && !rowExists:
		if _, err := tx.Exec(ctx, `INSERT INTO diagrams
  (id, project_id, name, document, created_by, updated_at, review_number)
  VALUES ($1, $2, $3, $4::jsonb, $5, $6, 1)`,
			d.ID, d.ProjectID, d.Name, string(d.Document), d.CreatedBy, d.UpdatedAt); err != nil {
			return VersionRecord{}, err
		}
		nextReview = 1
	case expectedReview != nil && rowExists && current == *expectedReview:
		if err := tx.QueryRow(ctx, `UPDATE diagrams
  SET name = $2, document = $3::jsonb, updated_at = $4, review_number = review_number + 1
  WHERE id = $1
  RETURNING review_number`,
			d.ID, d.Name, string(d.Document), d.UpdatedAt).Scan(&nextReview); err != nil {
			return VersionRecord{}, err
		}
	case expectedReview == nil && rowExists:
		// nil baseline but the row already exists: a previous call
		// could have left a row. Surface ErrReviewMismatch with the
		// live current_review so the service can retry.
		return VersionRecord{}, ErrReviewMismatch{Current: current}
	case expectedReview != nil && !rowExists:
		return VersionRecord{}, ErrReviewMismatch{Current: 0}
	case expectedReview != nil && rowExists && current != *expectedReview:
		return VersionRecord{}, ErrReviewMismatch{Current: current}
	}

	next := 1
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version_number), 0) + 1
  FROM diagram_versions WHERE diagram_id = $1`, d.ID).Scan(&next); err != nil {
		return VersionRecord{}, err
	}
	id := NewUUID()
	createdAt := time.Now().UTC()
	if _, err := tx.Exec(ctx, `INSERT INTO diagram_versions
  (id, diagram_id, version_number, review_number, document, created_by, created_at, message)
  VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8)`,
		id, d.ID, next, nextReview, string(d.Document), d.CreatedBy, createdAt, message); err != nil {
		return VersionRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		if isSerializationConflict(err) {
			live, lookupErr := p.CurrentReview(ctx, d.ID)
			if lookupErr != nil {
				return VersionRecord{}, lookupErr
			}
			return VersionRecord{}, ErrReviewMismatch{Current: live}
		}
		return VersionRecord{}, err
	}
	return VersionRecord{
		ID:           id,
		DiagramID:    d.ID,
		Number:       next,
		ReviewNumber: nextReview,
		Document:     append([]byte(nil), d.Document...),
		CreatedBy:    d.CreatedBy,
		CreatedAt:    createdAt,
		Message:      message,
	}, nil
}

// CurrentReview returns the diagrams row's current review_number (0 when no
// row exists yet). Service uses it to hydrate DiagramDocument.ReviewNumber on
// the response and to translate a 409 into "the server saw N as the latest
// baseline".
func (p *Postgres) CurrentReview(ctx context.Context, diagramID string) (int64, error) {
	var v int64
	err := p.pool.QueryRow(ctx, `SELECT review_number FROM diagrams WHERE id = $1`, diagramID).Scan(&v)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return v, nil
}

// CurrentVersion returns the highest current version_number written for a
// diagram (0 when no checkpoint row exists yet). It is the value the service
// mirrors into DiagramDocument.Version so clients can include it in their
// next PUT and POST /checkpoints call.
func (p *Postgres) CurrentVersion(ctx context.Context, diagramID string) (int, error) {
	var v int
	err := p.pool.QueryRow(ctx, `SELECT COALESCE(MAX(version_number), 0)
  FROM diagram_versions WHERE diagram_id = $1`, diagramID).Scan(&v)
	return v, err
}

func (p *Postgres) FindDiagram(ctx context.Context, projectID, diagramID string) (DiagramRecord, error) {
	var d DiagramRecord
	var payload string
	err := p.pool.QueryRow(ctx, `SELECT id, project_id, name, document::text, updated_at, review_number
  FROM diagrams WHERE id = $1 AND project_id = $2`, diagramID, projectID).
		Scan(&d.ID, &d.ProjectID, &d.Name, &payload, &d.UpdatedAt, &d.ReviewNumber)
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
	rows, err := p.pool.Query(ctx, `SELECT id, project_id, name, updated_at, review_number
  FROM diagrams WHERE project_id = $1 ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DiagramRecord
	for rows.Next() {
		var d DiagramRecord
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.Name, &d.UpdatedAt, &d.ReviewNumber); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (p *Postgres) ListVersions(ctx context.Context, diagramID string) ([]VersionRecord, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, diagram_id, version_number, review_number, document::text, created_by, created_at, message
  FROM diagram_versions WHERE diagram_id = $1 ORDER BY version_number DESC`, diagramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VersionRecord
	for rows.Next() {
		var v VersionRecord
		var payload string
		if err := rows.Scan(&v.ID, &v.DiagramID, &v.Number, &v.ReviewNumber, &payload, &v.CreatedBy, &v.CreatedAt, &v.Message); err != nil {
			return nil, err
		}
		v.Document = []byte(payload)
		out = append(out, v)
	}
	return out, rows.Err()
}

// isSerializationConflict reports err as a Postgres SQLSTATE 40001
// (serialization_failure) so callers can convert it to ErrReviewMismatch with
// the latest review_number look-up.
func isSerializationConflict(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "40001"
}

func (p *Postgres) FindVersion(ctx context.Context, diagramID string, number int) (VersionRecord, error) {
	var v VersionRecord
	var payload string
	err := p.pool.QueryRow(ctx, `SELECT id, diagram_id, version_number, review_number, document::text, created_by, created_at, message
  FROM diagram_versions WHERE diagram_id = $1 AND version_number = $2`, diagramID, number).
		Scan(&v.ID, &v.DiagramID, &v.Number, &v.ReviewNumber, &payload, &v.CreatedBy, &v.CreatedAt, &v.Message)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return VersionRecord{}, ErrNotFound
		}
		return VersionRecord{}, err
	}
	v.Document = []byte(payload)
	return v, nil
}
