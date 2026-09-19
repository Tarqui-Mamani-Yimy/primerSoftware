-- V4: bootstrap a personal project and starter diagram for lonely users.
--
-- Go-owned English tables only (users, projects, project_memberships,
-- diagrams, diagram_versions from V1); the Spanish base.sql tables are
-- untouched (disjoint names). This file sorts after V1–V3 via the numeric
-- filename convention, so the tables already exist whether the database
-- started from base.sql tables alone (V1 creates them earlier in the same
-- Up chain) or had V1–V3 applied long ago (only V4 is pending).
--
-- Fully rerun-safe: project/diagram/version ids are derived
-- deterministically from the user id (md5(...)::uuid), the project insert
-- is guarded by NOT EXISTS over project_memberships, later inserts guard
-- on the target row itself (later statements in the same transaction see
-- the membership rows inserted above, so they must not re-check lonely
-- status), and every INSERT uses ON CONFLICT DO NOTHING. A second apply
-- inserts zero rows, and a partially applied first run heals on retry.
INSERT INTO projects (id, name, description, access_code, owner_id)
SELECT
  md5(u.id::text || ':personal-project')::uuid,
  'My first project',
  'Personal starter project created automatically.',
  substring(md5(u.id::text || ':personal-access') from 1 for 12),
  u.id
FROM users u
WHERE NOT EXISTS (SELECT 1 FROM project_memberships pm WHERE pm.user_id = u.id)
ON CONFLICT DO NOTHING;

INSERT INTO project_memberships (project_id, user_id, role)
SELECT
  md5(u.id::text || ':personal-project')::uuid,
  u.id,
  'OWNER'
FROM users u
WHERE EXISTS (SELECT 1 FROM projects p WHERE p.id = md5(u.id::text || ':personal-project')::uuid)
  AND NOT EXISTS (SELECT 1 FROM project_memberships pm WHERE pm.user_id = u.id)
ON CONFLICT DO NOTHING;

-- Empty-diagram contract mirrors the V1 pattern: schemaVersion 1 with the
-- document id equal to the diagram row id, no classes or relationships.
INSERT INTO diagrams (id, project_id, name, document, created_by)
SELECT
  md5(u.id::text || ':personal-diagram')::uuid,
  md5(u.id::text || ':personal-project')::uuid,
  'First diagram',
  jsonb_build_object(
    'schemaVersion', 1,
    'id', (md5(u.id::text || ':personal-diagram')::uuid)::text,
    'name', 'First diagram',
    'classes', '[]'::jsonb,
    'relationships', '[]'::jsonb
  ),
  u.id
FROM users u
WHERE EXISTS (SELECT 1 FROM projects p WHERE p.id = md5(u.id::text || ':personal-project')::uuid)
  AND NOT EXISTS (SELECT 1 FROM diagrams d WHERE d.id = md5(u.id::text || ':personal-diagram')::uuid)
ON CONFLICT DO NOTHING;

INSERT INTO diagram_versions (id, diagram_id, version_number, document, created_by)
SELECT
  md5(u.id::text || ':personal-diagram-v1')::uuid,
  md5(u.id::text || ':personal-diagram')::uuid,
  1,
  jsonb_build_object(
    'schemaVersion', 1,
    'id', (md5(u.id::text || ':personal-diagram')::uuid)::text,
    'name', 'First diagram',
    'classes', '[]'::jsonb,
    'relationships', '[]'::jsonb
  ),
  u.id
FROM users u
WHERE EXISTS (SELECT 1 FROM diagrams d WHERE d.id = md5(u.id::text || ':personal-diagram')::uuid)
  AND NOT EXISTS (
    SELECT 1 FROM diagram_versions dv
    WHERE dv.diagram_id = md5(u.id::text || ':personal-diagram')::uuid
      AND dv.version_number = 1
  )
ON CONFLICT DO NOTHING;
