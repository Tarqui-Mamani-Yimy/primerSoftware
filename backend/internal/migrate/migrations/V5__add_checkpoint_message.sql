-- V5: add per-checkpoint authorship note.
--
-- Version history now records the username and optional commit-style message
-- for every explicit checkpoint. The autosave rows in diagram_versions were
-- already populated by V1/V2/V4; the new column defaults to NULL so those
-- pre-existing rows need no rewrite. Subsequent explicit checkpoint writes
-- (created by Go) populate both created_by and message.
--
-- Fully rerun-safe: ADD COLUMN IF NOT EXISTS is a no-op on databases where
-- the column is already present (e.g. a developer who manually applied this
-- migration before re-running the full chain).
ALTER TABLE diagram_versions
  ADD COLUMN IF NOT EXISTS message TEXT;
