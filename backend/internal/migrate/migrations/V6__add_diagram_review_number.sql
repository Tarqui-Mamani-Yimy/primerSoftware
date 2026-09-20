-- V6: per-diagram "work counter" used for optimistic concurrency on autosave
-- and explicit checkpoints.
--
-- `review_number` is independent of the explicit-checkpoint
-- `version_number` so concurrent autosaves never silently overwrite
-- each other: when a client sends a stale `review_number` the save
-- surfaces a 409 carrying the live value, and the client retries with
-- the new baseline.
--
-- The counter must exist on TWO tables so the working document row and
-- the snapshot rows land on the same timeline: `diagram_versions.review_number`
-- is stamped inside the same transaction as the version insert, and
-- `diagrams.review_number` is bumped on every successful save (autosave
-- or explicit checkpoint).
--
-- Backfill order matters: any statement that reads
-- `diagram_versions.review_number` MUST run AFTER
-- `ALTER TABLE diagram_versions ADD COLUMN review_number`. Earlier
-- revisions of this file started by adding the diagrams column and
-- then tried to `SELECT MAX(v.review_number)` while that column was
-- still missing, so V6 was a no-op or failed on a fresh DB.
--
-- Pre-V6 schema had no counter columns. After V6:
--   diagrams.review_number          : NOT NULL DEFAULT 1
--   diagram_versions.review_number  : NOT NULL DEFAULT 1
-- Existing rows are seeded so the working row points one past the
-- latest snapshot at migration time. Idempotent: every ALTER/UPDATE is
-- guarded so re-running V6 is safe.

-- 1. diagram_versions.review_number first.
ALTER TABLE diagram_versions
  ADD COLUMN IF NOT EXISTS review_number BIGINT;

-- 2. ROW_NUMBER backfill so each version row gets a deterministic
--    per-diagram review number, then collapse leftover NULLs to 1.
WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY diagram_id ORDER BY version_number) AS rn
  FROM diagram_versions
  WHERE review_number IS NULL
)
UPDATE diagram_versions v
  SET review_number = ranked.rn
  FROM ranked
  WHERE v.id = ranked.id;

UPDATE diagram_versions
  SET review_number = 1
  WHERE review_number IS NULL;

ALTER TABLE diagram_versions
  ALTER COLUMN review_number SET NOT NULL,
  ALTER COLUMN review_number SET DEFAULT 1;

-- 3. diagrams.review_number second: seeded from MAX(version_row review)
--    so the working row points one past the latest snapshot, then NULLs
--    collapse to 1 (brand-new diagram that never reached a checkpoint).
ALTER TABLE diagrams
  ADD COLUMN IF NOT EXISTS review_number BIGINT;

UPDATE diagrams d
  SET review_number = COALESCE((
    SELECT MAX(v.review_number)
    FROM diagram_versions v
    WHERE v.diagram_id = d.id
  ), 1)
  WHERE d.review_number IS NULL;

UPDATE diagrams
  SET review_number = 1
  WHERE review_number IS NULL;

ALTER TABLE diagrams
  ALTER COLUMN review_number SET NOT NULL,
  ALTER COLUMN review_number SET DEFAULT 1;

-- 4. Driver-friendly index for "latest review for this diagram".
CREATE INDEX IF NOT EXISTS idx_diagram_versions_diagram_review
  ON diagram_versions(diagram_id, review_number DESC);
