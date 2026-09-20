# Task: project-local-postgres-artifact

Feature document for the end-to-end correction of the generated JHipster artifact:
the generated Java backend must be runnable against a local PostgreSQL started
with Docker (compose + init SQL), and every generated entity must expose real
service/DTO/REST layers (no missing association links, no hardcoded frontend
data). Go stays the runtime; the generated Java project is independent per
request and full builds are run by the user.

## Objective

The ZIP produced by `POST /api/projects/{id}/diagrams/{id}/artifact` must
contain, **derived from the same UML document** and consistent with the Java
JPA artifact:

1. `database/<project-slug>.sql` — real PostgreSQL DDL (tables, PK/FK, unique,
   nullability, join tables) for every generateable UML entity/attribute/
   relationship **and** every JHipster-internal table the shipped changelogs
   create (user management), with seed data required for login.
2. `compose.yml` — `postgres:16-alpine`, `container_name <slug>-postgres`,
   `devuser`/`devpassword`, `POSTGRES_DB` = same slug, host port 5432, volume
   `./database/postgres_data`, exact bind mount of
   `./database/<project-slug>.sql` to
   `/docker-entrypoint-initdb.d/init.sql` (runs once on first boot).
3. Configured Java project: same database/credentials; Liquibase **disabled**
   (the init SQL already created the schema) while the changelogs stay for
   traceability; documented run mode.
4. Association classes produce **real links to both association ends**; every
   diagram entity gets `service * with serviceImpl` + `dto * with mapstruct`,
   so service interfaces/impls, DTOs, and REST resources are generated.
5. Manifest gains derived fields: `sqlFileName`, `databaseName`,
   `runCommands` — computed from the request (never hardcoded).

## Context and constraints

- Pinned generator `generator-jhipster@9.4.0` via `pnpm dlx --ignore-scripts`
  (authored installs are pnpm-only; never a global or shell install).
- Go runtime (backend/internal/{jdlgen,jhipster,service,httpapi}) stays; the
  generated Java is an independent per-request project.
- No AI/API keys, no CRDT/OT, no meetings, no push, no foreign files
  (`backend/internal/realtime/ticket.go`, `backend/credenciales.md`,
  `backend/database/`, `frontend/*`, other `odd/tasks/*` are owned by the user
  and must not be touched). No frontend/mobile edits beyond the manifest
  contract documented here.
- Local conventional commits, no `Co-Authored-By`.
- The user runs complete builds/tests; the agent may run focused pinned
  generation via pnpm and static validations to test the ZIP.
- Skill: `work-unit-commits` (loaded).

## Decisions

| ID | Decision | Rationale |
| --- | --- | --- |
| D-1 | Slug = kebab of `Options.BaseName` (camelCase boundary split + lowercase): `MyProject` → `my-project`; validated so the identifier is safe for JDBC URL, `POSTGRES_DB`, and file names. | Requirement example; single source of truth is the configured `baseName`. |
| D-2 | SQL file is rendered by a new `sqlgen` step from the **same** jdlgen intermediate model (entities, fields, relationships incl. association-class links) so JPA ↔ DDL stay 1:1. Naming mirrors JHipster's Liquibase conventions (probe T-1, cross-checked in e2e T-8): snake_case tables/columns, identity PK, `fk_<table>__<column>`, `ux_<table>__<column>` for OneToOne, join tables `rel_<owner>__<collection-field>` with PK `(<owner>_id, <field>_id)` and `fk_rel_<join>__<col>` constraints. Blob columns get a `<field>_content_type varchar(255)` sibling; the ID sequence `sequence_generator` starts at 1050 increment 50. | Consistency with the generated JPA artifact; no mock DDL. The join-table name uses the **collection field**, not the target entity (observed: `Course{tags}` → `rel_course__tags`). |
| D-3 | The SQL also creates JHipster-internal tables (user management: `jhi_user`, `jhi_authority`, `jhi_user_authority`, + audit if shipped) and seeds the authorities/admin users the generated changelogs would, so `spring.liquibase.enabled: false` keeps login working. | Hold Liquibase off (init SQL already applied the schema) but keep the changelog files for traceability. |
| D-4 | Patch the generated `src/main/resources/config/application-{dev,prod}.yml`: datasource URL/username/password = the compose DB, and Liquibase disabled. The `liquibase:`/`contexts:` anchor tolerates the generator's human comment lines (regex, comments preserved); a missing anchor fails generation loudly. Run mode: `docker compose up -d` then `./mvnw` (dev, default) or `./mvnw -Pprod`. `application.yml` also disables Spring's own docker-compose lifecycle so it cannot compete for port 5432. | Same credentials/db on both sides; one documented command set. Layout verified against the real 9.4.0 output (probe T-1 + e2e T-8). |
| D-5 | A UML class with `isAssociationClass=true` referencing attached relationship R emits, in addition to R itself, `ManyToOne` links from the association entity to **both** of R's endpoints (field per endpoint), with warnings. | JHipster has no association-class construct; real JPA/FK links to the ends are required. |
| D-6 | Application block gains `service * with serviceImpl` and `dto * with mapstruct` **after** the `entities` line (JDL option keywords in the app block), verified to parse/generate by probe T-1. | Real services, DTOs, and REST resources for every diagram entity. |
| D-7 | Manifest adds `sqlFileName`, `databaseName`, `runCommands`; `Files` includes the new SQL and compose (both physically present when listed). | Frontend can render derived data; manifest only lists real files. |
| D-8 | No `skipUserManagement` (keeps JHipster's user/Auth entities + login) — SQL mirrors their tables + seed. | Avoid breaking JWT login of the generated app. |
| D-9 | `listGenerated`/`buildZip` order stays deterministic; database/ + compose.yml written before file listing so the manifest is truthful. | Existing provenance discipline. |

## Tasks

### Backend

- [x] T-1 (probe) Focused pinned generation in `/tmp/opencode/probe-mar24` (+ `/tmp/opencode/probe-types`): JDL with app block + `service * with serviceImpl` + `dto * with mapstruct`, OneToOne/OneToMany/ManyToMany incl. association-class shape, user management present. Record: config YAML keys for datasource/liquibase; master.xml changelog list; table/column/FK/join-table naming and id type; internal user tables + seed data; generated java file list (domain/service/service-impl/dto/rest); pom mapstruct handling. **Ground truth captured and applied.**
- [x] T-2 Slug derivation + validation: `sqlgen.Slugify` + unit tests (`MyProject`→`my-project`, `UMLArchitect`→`uml-architect`, `Order2`→`order2`); invalid `baseName` already rejected by `ValidateOptions` (400).
- [x] T-3 `sqlgen` package: `RenderSQL(model)` renders `database/<slug>.sql` DDL from the same intermediate model incl. association-class links (D-5) and JHipster internal tables/seeds (D-3), idempotent (IF NOT EXISTS / ON CONFLICT DO NOTHING); 6 tests assert 1:1 tables/columns/types/FKs/join tables/kebab outputs.
- [x] T-4 Compose + config patching: `RenderCompose(slug)` (D-2 fields) and `PatchApplicationConfig` (D-4), deterministic, written before listing; tests cover patching and missing-anchor failure.
- [x] T-5 jdlgen association-class link emission (D-5) with report warnings; goldens/tests updated (`TestExportSynthesizesAssociationLinks`, collision → `student2`).
- [x] T-6 jdlgen app block options (D-6) for service/dto; goldens/tests updated.
- [x] T-7 Manifest/endpoint contract (D-7): `sqlFileName`, `databaseName`, `runCommands` fields flow through `Manifest`; unit tests assert values + new zip entries.
- [x] T-8 Work-unit commits (pending split below); static validation green (gofmt/vet/`go test ./internal/...` = 230); focused pnpm e2e (`JHIPSTER_E2E=1`) green against the real generator: JDL parses, join-table/FK/unique names match the real changelogs, blob `_content_type` matches, internal tables present, config patching works on real yml.
- [x] T-9 This doc updated with resolved evidence; Spanish report follows.

Regression caught by the e2e and fixed (worth recording):
- `renderRelationship` emitted the cardinality keyword twice (`relationship ManyToMany { ManyToMany A{...} to B{...} }`) — the real generator rejects it with `MismatchedTokenException`. Goldens had been regenerated with the bug; e2e caught it, unit tests and goldens corrected to the grammar `A{...} to B{...}`.
- The real `application-dev.yml` carries a comment between `liquibase:` and `contexts:`; the exact-string anchor failed → `patchLiquibase` regex tolerates comment lines (preserved verbatim).

### Web (none)

- No frontend/mobile source changes. Allowed: only the manifest JSON contract fields documented here so a later frontend change can consume them.

## Pending checks (user runs)

- `go test ./internal/... -count=1` (agent runs focused subsets) — user runs the full suite/CI.
- Unzip the generated ZIP, `docker compose up -d`, `./mvnw` (dev) — user runs; agent validates statically + via focused generation.

## Progress

- T-1..T-9 done (task active since 2026-09-20 after `real-jhipster-backend-generation.md`, closed, commits 08283d9 / 7fcddc6 / 9c72833). JDL refactor (build model + association links + app options) and sqlgen + jhipster provisioning implemented; focused e2e against the real generator green. Remaining: work-unit commits (WU-1 jdlgen / WU-2 sqlgen / WU-3 jhipster + e2e), user-side full build/test, and the Spanish close report with DB commands.
- **Verification so far (agent-run, focused):** `go test ./internal/... -count=1` → 230 passed (incl. 68 jdlgen, 6 sqlgen, 10 jhipster + 1 env-gated e2e); `gofmt -l` clean on touched packages; `go vet` clean. Full build/CI and `docker compose up -d` + boot are user-run checks.
- **Close commands for the user (final report basis):** unzip the artifact → `cd <baseName>` → `docker compose up -d` (first boot runs database/<slug>.sql) → `./mvnw` (dev) or `./mvnw -Pprod`; reset with `docker compose down -v` (re-runs init SQL on next up). Credentials: `admin`/`admin`, `user`/`user`.

## Re-correction pass (2026-09-20)

Re-verified against the current tree as source of truth. Three defects found and fixed:

1. **ManyToMany autorreferenciales sin join table** — `sqlgen.RenderSQL` skipped self-associations (`Src == Dst`), emitting only a comment. JHipster 9.4.0 requires a real join table for `ManyToMany Order{orders} to Order{orders}`. Removed the skip; the join table is now emitted with the same shape as any other ManyToMany (`rel_order__orders`, composite PK, two FKs to the same table). Test `TestRenderSQLStructure` extended with the self-association assertions.

2. **Comandos de manifest sin `cd` al ZIP root** — The ZIP root contains a single `<baseName>/` directory, but `runCommands` returned `docker compose up -d` directly, so the user would land in the wrong directory. Added `cd <baseName>` as the first command (both maven and gradle). Updated `TestRunCommandsPerBuildTool` and the manifest assertion in `TestGenerateProducesZipWithManifest`.

3. **Reset bind mount postgres_data documentado** — The compose file uses a bind mount (`./database/postgres_data`), not a named volume. `docker compose down -v` removes the local directory and re-runs the init SQL on the next `up`. The compose comment already documented this; the task doc now records the exact close/reset commands.