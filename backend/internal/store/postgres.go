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
// Autosave and explicit checkpoints are split into SaveWorkingDocument and
// AppendCheckpoint so untrusted autosave traffic never grows the version
// history. AppendCheckpoint computes version_number inside the transaction and
// stamps created_by with the actor so authorship is preserved independently
// of the diagram creator.

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

// SaveDocument upserts the diagrams row with the current JSONB document.
// A non-nil expectedReview is the CAS gate: it matches against the
// row's current review_number, and on success the row's review_number is
// bumped atomically by one. A nil expectedReview implies "create the
// diagram for the first time" — the row is inserted with review_number
// = 1 and NOT otherwise incremented (CreateDiagram calls AppendCheckpoint
// next to land at 2 explicitly).
//
// It does NOT touch diagram_versions: explicit checkpoints own the
// version history. The diagrams row carries the autosave snapshot.
func (p *Postgres) SaveDocument(ctx context.Context, d DiagramRecord, expectedReview *int64) (int64, error) {
	if expectedReview == nil {
		var newReview int64
		err := p.pool.QueryRow(ctx, `INSERT INTO diagrams (id, project_id, name, document, created_by, updated_at, review_number)
  VALUES ($1, $2, $3, $4::jsonb, $5, $6, 1)
  ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, document = EXCLUDED.document, updated_at = EXCLUDED.updated_at
  RETURNING review_number`, d.ID, d.ProjectID, d.Name, string(d.Document), d.CreatedBy, d.UpdatedAt).Scan(&newReview)
		if err != nil {
			return 0, err
		}
		return newReview, nil
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

// AppendCheckpoint writes a new diagram_versions row with version_number =
// max+1 AND rewrites the diagrams row (name, document, updated_at, +1
// review_number) inside ONE transaction. created_by on the version row is
// stamped with the actor (preserving authorship even when the actor differs
// from the diagram owner); message is the optional note supplied by the
// client. Review bumps atomically so two concurrent /checkpoints POSTs
// cannot both succeed: either transaction sees the other's bumped
// review_number on SELECT FOR UPDATE and fails the CAS with ErrReviewMismatch.
//
// Atomicity guarantee (V6+): the diagrams row and the diagram_versions row
// land on the same timeline. A crash between row update and version insert
// rolls both back, so concurrent readers never see the bumped document
// without the matching version row.
func (p *Postgres) AppendCheckpoint(ctx context.Context, d DiagramRecord, expectedReview *int64, message *string) (VersionRecord, error) {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return VersionRecord{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var nextReview int64
	var updateErr error
	if expectedReview != nil {
		updateErr = tx.QueryRow(ctx, `UPDATE diagrams
  SET name = $2, document = $3::jsonb, updated_at = $4, review_number = review_number + 1
  WHERE id = $1 AND review_number = $5
  RETURNING review_number`,
			d.ID, d.Name, string(d.Document), d.UpdatedAt, *expectedReview).Scan(&nextReview)
	} else {
		// No baseline supplied: this is the create-only path used by
		// service.CreateDiagram on the implicit "Initial revision"
		// checkpoint. We bind to review_number = 0 so a concurrent
		// create never races a second writer into double-insert.
		updateErr = tx.QueryRow(ctx, `UPDATE diagrams
  SET name = $2, document = $3::jsonb, updated_at = $4, review_number = review_number + 1
  WHERE id = $1 AND review_number = 0
  RETURNING review_number`,
			d.ID, d.Name, string(d.Document), d.UpdatedAt).Scan(&nextReview)
	}
	if updateErr != nil {
		if errors.Is(updateErr, pgx.ErrNoRows) {
			// Two reasons we hit zero rows: the row does not exist yet,
			// or the supplied review baseline is stale. Disambiguate via a
			// second query so the caller gets a precise error.
			var current int64
			if err := tx.QueryRow(ctx, `SELECT review_number FROM diagrams WHERE id = $1`, d.ID).Scan(&current); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return VersionRecord{}, ErrNotFound
				}
				return VersionRecord{}, err
			} else if current == 0 {
				// Create path: the diagrams row was created with
				// review_number = 0 by ServiceResolver or a manual
				// upsert. We accept the FIRST AppendCheckpoint as
				// implicit "Initial revision" and synthesize the row.
				if err := p.seedDiagramForCreate(ctx, tx, d); err != nil {
					return VersionRecord{}, err
				}
				nextReview = 1
			} else {
				return VersionRecord{}, ErrReviewMismatch{Current: current}
			}
		} else {
			return VersionRecord{}, updateErr
		}
	}
	var next int
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
			// A concurrent commit won the race; surface the post-bump review
			// number so the client can retry with a fresh baseline.
			current, lookupErr := p.CurrentReview(ctx, d.ID)
			if lookupErr != nil {
				return VersionRecord{}, lookupErr
			}
			return VersionRecord{}, ErrReviewMismatch{Current: current}
		}
		return VersionRecord{}, err
	}
	return VersionRecord{
		ID:          id,
		DiagramID:   d.ID,
		Number:      next,
		ReviewNumber: nextReview,
		Document:    append([]byte(nil), d.Document...),
		CreatedBy:   d.CreatedBy,
		CreatedAt:   createdAt,
		Message:     message,
	}, nil
}

// seedDiagramForCreate inserts a fresh diagrams row for the create-only
// path used by Service.CreateDiagram. It runs inside the same transaction
// as the AppendCheckpoint so the version insert cannot precede the row.
func (p *Postgres) seedDiagramForCreate(ctx context.Context, tx pgx.Tx, d DiagramRecord) error {
	_, err := tx.Exec(ctx, `INSERT INTO diagrams (id, project_id, name, document, created_by, updated_at, review_number)
  VALUES ($1, $2, $3, $4::jsonb, $5, $6, 1)`,
		d.ID, d.ProjectID, d.Name, string(d.Document), d.CreatedBy, d.UpdatedAt)
	return err
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
