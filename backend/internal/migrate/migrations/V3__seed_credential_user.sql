-- V3: seed the credential user and align the development password hashes.
--
-- Go-owned English schema only (users table from V1); the Spanish base.sql
-- tables are untouched. This migration always runs after V1/V2 in
-- migrate.Up order, so the users table (with its UNIQUE email constraint)
-- already exists — whether the database started from base.sql tables alone
-- or had V1/V2 applied before.
--
-- Fully idempotent, safe to rerun: the INSERT skips on email conflict and
-- the UPDATE rewrites the same hash value every time, so already-aligned
-- rows and user-edited diagrams are never disturbed.
INSERT INTO users (id, display_name, email, password_hash) VALUES
  ('44444444-4444-4444-4444-444444444444', 'Seed User', 'yimyt771@gmail.com', '$2b$12$HWqim2KP4GxJWv1kOb79tudNBV0Y94xoM2y5JRa32DSGPigvenb1C')
ON CONFLICT (email) DO NOTHING;

UPDATE users
SET password_hash = '$2b$12$HWqim2KP4GxJWv1kOb79tudNBV0Y94xoM2y5JRa32DSGPigvenb1C'
WHERE email IN ('ana@example.com', 'bruno@example.com', 'camila@example.com', 'yimyt771@gmail.com');
