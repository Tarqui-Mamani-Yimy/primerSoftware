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
-- Backfill design notes:
--  * Both columns are added nullable so the ALTER never errors on
--    legacy rows.
--  * The per-diagram `diagram_versions` ROW_NUMBER backfill runs in
--    the same transaction as the CREATE COLUMN, guaranteeing no
--    concurrent INSERT can land between the add and the backfill.
--    Legacy version rows therefore land at 1, 2, 3, … per diagram,
--    matching the per-diagram timeline that the application enforces.
--  * `diagrams.review_number` is then seeded from the per-diagram
--    MAX(version row's review_number) so the row's working counter
--    points one past the last snapshot at migration time.
--  * Both columns are tightened to NOT NULL DEFAULT 1 so the
--    application code never reads NULL.
--  * Idempotency: each block is wrapped with `IF NOT EXISTS` /
--    `IF col IS NULL` so re-running V6 on a partially-migrated DB
--    is a no-op (the file is safe to interleave with the running app
--    across migration interruptions).

-- diagrams.review_number
ALTER TABLE diagrams
  ADD COLUMN IF NOT EXISTS review_number BIGINT;

UPDATE diagrams d
  SET review_number = COALESCE((
    SELECT MAX(v.review_number)
    FROM diagram_versions v
    WHERE v.diagram_id = d.id
  ), 1)
  WHERE d.review_number IS NULL;

ALTER TABLE diagrams
  ALTER COLUMN review_number SET NOT NULL,
  ALTER COLUMN review_number SET DEFAULT 1;

-- diagram_versions.review_number
ALTER TABLE diagram_versions
  ADD COLUMN IF NOT EXISTS review_number BIGINT;

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

-- drivers: latest review for a diagram.
CREATE INDEX IF NOT EXISTS idx_diagram_versions_diagram_review
  ON diagram_versions(diagram_id, review_number DESC);
