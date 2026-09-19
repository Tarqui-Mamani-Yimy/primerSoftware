-- V1: create the Go-owned schema and development seed.
-- Replicates the Java Flyway V1 authority (tables, indexes, dev seeds) without
-- hibernate ddl-auto: the schema only ever changes through these versioned
-- files. Every statement is idempotent so V1 can apply over a pre-existing
-- database (for example one provisioned by base.sql) without 42P07/23505:
-- tables and indexes use IF NOT EXISTS and the fixed-ID dev seed rows skip on
-- conflict. Pre-existing tables with a colliding name (base.sql's
-- refresh_tokens) are preserved and never dropped; the Go refresh-token store
-- is served by the Go-owned table created in the isolated uml_architect
-- database (see the runbook).
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  display_name VARCHAR(150) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS projects (
  id UUID PRIMARY KEY,
  name VARCHAR(150) NOT NULL,
  description TEXT,
  access_code VARCHAR(16) NOT NULL UNIQUE,
  owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS project_memberships (
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role VARCHAR(20) NOT NULL CHECK (role IN ('OWNER', 'COLLABORATOR')),
  joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_project_memberships_user ON project_memberships(user_id);
CREATE TABLE IF NOT EXISTS diagrams (
  id UUID PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  document JSONB NOT NULL,
  created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_diagrams_project ON diagrams(project_id, updated_at DESC);
CREATE TABLE IF NOT EXISTS diagram_versions (
  id UUID PRIMARY KEY,
  diagram_id UUID NOT NULL REFERENCES diagrams(id) ON DELETE CASCADE,
  version_number INTEGER NOT NULL CHECK (version_number > 0),
  document JSONB NOT NULL,
  created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uq_diagram_versions_number UNIQUE (diagram_id, version_number)
);
CREATE INDEX IF NOT EXISTS idx_diagram_versions_diagram_created ON diagram_versions(diagram_id, version_number DESC);
CREATE INDEX IF NOT EXISTS idx_diagram_versions_document_gin ON diagram_versions USING GIN(document);
CREATE TABLE IF NOT EXISTS refresh_tokens (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash VARCHAR(64) NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- The Go-owned refresh_tokens index is guarded: a database that already
-- carries base.sql's same-named table (different columns) keeps its table
-- untouched and this index stays unbuilt there. Go-owned databases get the
-- index as part of the schema.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'refresh_tokens' AND column_name = 'user_id'
  ) THEN
    CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
  END IF;
END $$;

-- Development-only identities. Password for every account is: Password123!
-- ON CONFLICT makes the seed safe to re-run over a database that already
-- holds these fixed IDs or unique keys.
INSERT INTO users (id, display_name, email, password_hash) VALUES
 ('11111111-1111-1111-1111-111111111111','Ana Fernández','ana@example.com','$2y$10$0po1d5ULCpVopbelx2gsj.6UfMTCpVQka2C1LubEfBhi.aK.9.VmO'),
 ('22222222-2222-2222-2222-222222222222','Bruno Quispe','bruno@example.com','$2y$10$0po1d5ULCpVopbelx2gsj.6UfMTCpVQka2C1LubEfBhi.aK.9.VmO'),
 ('33333333-3333-3333-3333-333333333333','Camila Rojas','camila@example.com','$2y$10$0po1d5ULCpVopbelx2gsj.6UfMTCpVQka2C1LubEfBhi.aK.9.VmO')
ON CONFLICT (email) DO NOTHING;
INSERT INTO projects (id,name,description,access_code,owner_id) VALUES
 ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','Sistema de Ventas','Diagrama de clases para ventas y facturación','X7K2P9','11111111-1111-1111-1111-111111111111'),
 ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','Gestión de Biblioteca','Modelo para préstamos y catálogo','L4M8QW','33333333-3333-3333-3333-333333333333')
ON CONFLICT (access_code) DO NOTHING;
INSERT INTO project_memberships(project_id,user_id,role) VALUES
 ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','11111111-1111-1111-1111-111111111111','OWNER'),
 ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','22222222-2222-2222-2222-222222222222','COLLABORATOR'),
 ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','33333333-3333-3333-3333-333333333333','OWNER'),
 ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111','COLLABORATOR')
ON CONFLICT (project_id, user_id) DO NOTHING;
INSERT INTO diagrams(id,project_id,name,document,created_by) VALUES
 ('cccccccc-cccc-cccc-cccc-cccccccccccc','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','Diagrama principal', '{"schemaVersion":1,"id":"cccccccc-cccc-cccc-cccc-cccccccccccc","name":"Diagrama principal","classes":[],"relationships":[]}'::jsonb, '11111111-1111-1111-1111-111111111111')
ON CONFLICT (id) DO NOTHING;
INSERT INTO diagram_versions(id,diagram_id,version_number,document,created_by) SELECT
 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', id, 1, document, created_by FROM diagrams WHERE id='cccccccc-cccc-cccc-cccc-cccccccccccc'
ON CONFLICT (id) DO NOTHING;