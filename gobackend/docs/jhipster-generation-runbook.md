# Generate a downloadable JHipster project from a UML diagram (GEN-02)

From one UML diagram, prepare everything needed to generate and download an
independent JHipster Java Spring Boot/JPA + PostgreSQL project. The Go backend
remains the `ai-uml-architect` runtime and is never modified by this flow.

> Status note: JHipster is NOT installed where the bundle is prepared
> (`jhipster: command not found`). This runbook therefore covers preparation
> here plus the exact generation steps for a machine that has JHipster.

## Quick path

1. Prepare the bundle (Go toolchain only, no installs):
   ```bash
   ./gobackend/tools/prepare-jhipster-bundle.sh /tmp/shop.json shop ./shop-jhipster
   ```
2. Review `shop-jhipster/report.json` — resolve every warning (renames,
   skipped inheritance, unknown types) before generating.
3. Copy `shop-jhipster/` to a machine with Node.js LTS, Java 17+, and network
   access, then run the ordered commands:
   ```bash
   cd shop-jhipster
   PINNED_JHIPSTER="$(npm view generator-jhipster version)" ./generate.sh
   ```
   (`generate.sh` runs: `npm install -g generator-jhipster@<pinned>` →
   `jhipster --version` → `jhipster jdl model.jdl --force` →
   `./mvnw -Pprod verify`.)
4. Verify the expected artifacts listed below, then configure the generated
   project's OWN PostgreSQL database.

## Details

| Topic | Decision |
|-------|----------|
| Runtime boundary | Go is the production runtime. The generated Spring Boot project is a separate, downloadable artifact and is NEVER pointed at the Go runtime database. |
| Database rule | The generated app uses JHipster Liquibase changelogs (default). The Go backend uses versioned SQL migrations and never uses `ddl-auto`. Neither side shares a database. |
| UML → JDL mapping | Classes become entities; attribute types map via a fixed table (`Money`→`BigDecimal`, `string`→`String`, `int`→`Integer`, `bool`→`Boolean`, `date`→`LocalDate`, `datetime`→`ZonedDateTime`, `uuid`→`UUID`, `text`→`TextBlob`, `blob`→`Blob`, …). Unknown types fall back to `String` with a warning. |
| Relationship cardinalities | Derived from multiplicities: `*`/`0..*`/`1..*`/bounds above 1 count as many; missing or `1`/`0..1` count as one. `association`, `aggregation`, and `composition` all emit JDL cardinalities; aggregation/composition ownership and cascade semantics flatten to plain associations (warned). |
| Warn-and-skip | `generalization`, `realization`, and `dependency` are omitted from `model.jdl` and recorded in `report.json`. Inheritance must be remodeled in the generated project by hand. |
| Enums | A class with an `Enum` stereotype is skipped with a warning: the UML Attribute carries `id`/`name`/`type`/`visibility` only, so there is no value list to emit a JDL enum from. |
| Always dropped | Methods, member visibility, canvas layout (`x`/`y`/`width`), package names, table bindings, and relationship labels have no JDL equivalent; each occurrence is listed in `report.json`. |
| Identifier safety | Entity and field names are sanitized to legal Java identifiers (reserved words, leading digits, illegal characters) with deterministic `_2`, `_3` collision suffixes. Every rename is a warning. |
| `.yo-rc.json` seed | Pinned monolith seed: `applicationType: monolith`, `databaseType: sql`, `devDatabaseType`/`prodDatabaseType: postgresql`, `buildTool: maven`. `baseName`/`packageName` derive from the requested app name. |

## Expected JHipster artifacts

| Artifact | Where |
|----------|-------|
| JPA entities (`@Entity`) | `src/main/java/<package>/domain/` |
| Spring Data repositories | `src/main/java/<package>/repository/` |
| REST resources + DTOs/mappers | `src/main/java/<package>/web/rest/` |
| Liquibase changelogs (PostgreSQL) | `src/main/resources/config/liquibase/changelog/` |
| Production datasource config | `src/main/resources/config/application-prod.yml` |

## Checklist

- [ ] `report.json` reviewed: renames accepted, skipped inheritance remodeled, unknown types corrected.
- [ ] `PINNED_JHIPSTER` set to an exact generator version (reproducible output).
- [ ] `jhipster jdl model.jdl --force` completed without entity errors.
- [ ] `./mvnw -Pprod verify` passed on the generation machine.
- [ ] Generated app points at its OWN PostgreSQL database, never the Go runtime database.
- [ ] Go runtime untouched: no server/API changes, no `ddl-auto` anywhere.

## Next step

Hand the bundle directory to the requester as the downloadable artifact; keep
`model.jdl` + `report.json` alongside the diagram for auditability.
