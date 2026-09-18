# Relationship reconfiguration

## Objective
Allow an existing UML relationship to be reconfigured (type, source, destination, label, and cardinalities) while preventing invalid diagrams both in the UI and API.

## Problem and why
The UI already supports creating, linking, naming, and assigning cardinalities to relationships, but it cannot change a relationship's type or endpoints after creation. The backend accepts structurally invalid relationship documents.

## Scope and constraints
- Preserve the existing create/link/delete workflow and current persisted document format.
- Do not touch pre-existing changes: `backend/mvnw`, `frontend/pnpm-lock.yaml`, or `frontend/pnpm-workspace.yaml`.
- Use strict TDD when a runnable test harness is available; otherwise record the unavailable runner and run the narrowest static/build checks available.
- Route: delegated. Trigger evidence: implementation requires coordinated frontend and backend changes across multiple non-trivial files.

## Delivery
- Strategy: ask-on-risk
- Forecast: approximately 250 authored changed lines, excluding tests/configuration.

## Tasks
- [x] REL-1 Add UI controls and state flow to edit type and endpoints of an existing relationship, while reusing relationship validation. Acceptance: edited relationship updates, persists, and invalid relinks are rejected. Checks: frontend typecheck/build unavailable because the environment has node_modules but no `node`, `npm`, or `pnpm`; reviewed the TypeScript path structurally.
- [x] REL-2 Add backend document validation for relationship endpoints, type, duplicates, and multiplicity syntax. Acceptance: invalid API documents return 400 without persistence. Checks: `backend/./mvnw -q -Dtest=DiagramDocumentValidatorTest test` passed.
- [x] REL-3 Add focused regression tests for relationship reconfiguration and server validation. Acceptance: valid edits pass; invalid endpoints, duplicate relationships, invalid realization and multiplicity are rejected. Checks: `DiagramDocumentValidatorTest` covers valid documents plus missing endpoints, unsupported type, duplicates, invalid multiplicity, and invalid realization; passed. Full backend suite remains blocked by unavailable PostgreSQL/network socket access.

## Progress
- REL-1 completed: relationship inspector stages and saves edits to type, source, target, label, and multiplicities; CanvasView excludes the edited relationship while applying shared validation. Frontend runner unavailable (`node`, `npm`, and `pnpm` are absent).
- REL-2 completed: server-side semantic validation runs before persistence and `IllegalArgumentException` is returned as HTTP 400.
- REL-3 completed: focused validator unit tests pass. `backend/./mvnw -q test` still fails only in the pre-existing integration test setup because PostgreSQL socket access is not permitted.
- Work-unit commit: `f574d1b feat: reconfigure UML relationships`.
- RDD assessment: unavailable. `gentle-ai review assess` cannot create its temporary index below `.git` because the filesystem is read-only; user explicitly declined reporting the gentle-ai defect.
- Next step: run frontend lint/build and the full backend integration suite in an environment with Node and PostgreSQL.
