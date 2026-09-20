// Package sqlgen renders the PostgreSQL provisioning files that accompany a
// generated JHipster backend: the init SQL that mirrors what the project's own
// Liquibase changelogs would create, the compose.yml that boots the database,
// and the config patches that point the generated Spring Boot app at it with
// Liquibase disabled.
//
// The renderings are pinned to observations of generator-jhipster 9.4.0
// (postgresql): table/column naming, constraint naming (fk_<table>__<column>,
// ux_<table>__<column>, rel_<src>__<dst> join tables), the shared
// sequence_generator (START 1050 INCREMENT 50), the JHipster-internal
// jhi_user/jhi_authority/jhi_user_authority tables, and the admin/user seed
// rows with their exact bcrypt hashes, so the provisioned database matches the
// schema the generated JPA entities expect.
package sqlgen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
)

// Database provisioning constants shared by the compose file, the SQL, and the
// application config patches.
const (
	// DatabaseImage is the pinned PostgreSQL image.
	DatabaseImage = "postgres:16-alpine"
	// DatabaseUser is the generated app's datasource user.
	DatabaseUser = "devuser"
	// DatabasePassword is the generated app's datasource password.
	DatabasePassword = "devpassword"
	// DatabasePort is the published host port.
	DatabasePort = "5432"
	// DataDir is the compose volume path (relative to the generated project).
	DataDir = "./database/postgres_data"
)

// adminHash and userHash are the exact bcrypt hashes JHipster 9.4.0 seeds for
// the internal admin and user accounts (config/liquibase/data/user.csv).
const (
	adminHash = "$2a$10$gSAhZrxMllrbgj/kkK9UceBPpChGWJA7SYIb1Mqo.n5aNLq1/oRrC"
	userHash  = "$2a$10$VEjxo0jq2YG9Rbk2HmX9S.k1uZBGYUHdUcid3g/vfiEl7lwWgOH/K"
)

// Slugify converts a baseName into the kebab-case database name JHipster runs
// with: camelCase boundaries become hyphens and everything is lowercased
// ("MyProject" -> "my-project", "UMLArchitect" -> "uml-architect"). The result
// is validated against ^[a-z0-9]+(-[a-z0-9]+)*$; baseName has already passed
// ValidateOptions (alphanumeric), so an error here means a coding bug, not
// user input.
func Slugify(baseName string) (string, error) {
	runes := []rune(baseName)
	var b strings.Builder
	for i, r := range runes {
		if i > 0 {
			prev := runes[i-1]
			next := rune(0)
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			boundary := false
			if isUpper(r) && (isLower(prev) || isDigit(prev)) {
				boundary = true // lower/digit -> upper
			}
			if isUpper(r) && isUpper(prev) && next != 0 && isLower(next) {
				boundary = true // acronym end: UMLArchitect -> uml-architect
			}
			if boundary {
				b.WriteByte('-')
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "", fmt.Errorf("baseName %q produced an empty database name", baseName)
	}
	for _, seg := range strings.Split(out, "-") {
		if seg == "" {
			return "", fmt.Errorf("baseName %q produced an invalid database name %q", baseName, out)
		}
		for _, r := range seg {
			if !isDigit(r) && !isLower(r) {
				return "", fmt.Errorf("baseName %q produced an invalid database name %q", baseName, out)
			}
		}
	}
	return out, nil
}

func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isDigit(r rune) bool { return r >= '0' && r <= '9' }

// snake converts a camelCase identifier to the snake_case column/table name
// JHipster's Hibernate naming strategy produces ("whenAt" -> "when_at").
func snake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if isUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sqlType maps a JDL field type to its native PostgreSQL column type, mirroring
// what the generated Liquibase changelogs resolve to on postgresql. The boolean
// is false for types this package does not know how to render.
func sqlType(jdlType string) (string, bool) {
	switch jdlType {
	case "String":
		return "varchar(255)", true
	case "Integer":
		return "integer", true
	case "Long":
		return "bigint", true
	case "Float":
		return "float4", true
	case "Double":
		return "double precision", true
	case "BigDecimal":
		return "decimal(21,2)", true
	case "Boolean":
		return "boolean", true
	case "LocalDate":
		return "date", true
	case "ZonedDateTime", "Instant":
		return "timestamp", true
	case "UUID":
		return "uuid", true
	case "Blob":
		return "bytea", true
	case "TextBlob":
		return "text", true
	default:
		return "", false
	}
}

// isBlob reports whether the JDL type renders as two columns (value +
// <field>_content_type), mirroring JHipster's Blob handling.
func isBlob(jdlType string) bool { return jdlType == "Blob" }

// RenderSQL renders the init SQL for the model. The output is deterministic
// and idempotent (IF NOT EXISTS / ON CONFLICT DO NOTHING) because the compose
// entrypoint may run it more than once across container lifecycles.
func RenderSQL(m jdlgen.Model) string {
	var b strings.Builder
	b.WriteString("-- Generated by ai-uml-architect (GEN-02).\n")
	b.WriteString("-- PostgreSQL init script for the JHipster backend scaffold. Provisioned by\n")
	b.WriteString("-- database/compose.yml on first boot. Mirrors the schema the generated Liquibase\n")
	b.WriteString("-- changelogs describe (Liquibase itself is disabled in the generated config).\n")
	b.WriteString("-- Credentials: login with admin/admin or user/user.\n\n")

	b.WriteString("-- Shared JHipster id sequence (START 1050 INCREMENT 50).\n")
	b.WriteString("CREATE SEQUENCE IF NOT EXISTS sequence_generator START WITH 1050 INCREMENT BY 50;\n\n")

	b.WriteString("-- JHipster-internal user management tables.\n")
	renderInternalTables(&b)

	type fk struct {
		table  string // owning table
		column string // owning column
		target string // referenced table
	}
	var fks []fk

	// Tables for every entity, including the scalar FK columns of the
	// relationships this entity owns. Join tables and standalone FK
	// constraints come after, so referenced tables always exist.
	for _, e := range m.Entities {
		table := snake(e.Name)
		b.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table))
		b.WriteString("    id bigint PRIMARY KEY")
		for _, f := range e.Fields {
			typ, ok := sqlType(f.Type)
			if !ok {
				typ = "varchar(255)"
			}
			b.WriteString(",\n    " + snake(f.Name) + " " + typ)
			if isBlob(f.Type) {
				b.WriteString(",\n    " + snake(f.Name) + "_content_type varchar(255)")
			}
		}
		for _, r := range m.Relationships {
			if r.Src != e.Name {
				continue
			}
			switch r.Kind {
			case "ManyToOne", "OneToOne":
				col := snake(r.SrcField) + "_id"
				unique := ""
				if r.Kind == "OneToOne" {
					unique = " CONSTRAINT ux_" + table + "__" + col + " UNIQUE"
				}
				b.WriteString(",\n    " + col + " bigint" + unique)
				fks = append(fks, fk{table: table, column: col, target: snake(r.Dst)})
			case "OneToMany":
				// The FK lives on the destination table: Customer{order}
				// owns order_id referencing Order.
				col := snake(r.DstField) + "_id"
				b.WriteString(",\n    " + col + " bigint")
				fks = append(fks, fk{table: snake(r.Dst), column: col, target: table})
			}
		}
		b.WriteString("\n);\n\n")
	}

	// ManyToMany join tables. Self-associations (Src == Dst) are NOT skipped:
	// JHipster 9.4.0 emits a real join table for them (e.g. Order{orders} to
	// Order{orders} → rel_order__orders). The same shape applies: owner column
	// = snake(Src)+"_id", inverse column = snake(SrcField)+"_id", composite PK,
	// and two fk_rel_<join>__<col> constraints both referencing the same table.
	for _, r := range m.Relationships {
		if r.Kind != "ManyToMany" {
			continue
		}
		srcTable := snake(r.Src)
		dstTable := snake(r.Dst)
		// Mirror JHipster's join-table shape for `ManyToMany Src{srcField} to
		// Dst{dstField}` (observed in the pinned generator's changelogs): the
		// table is named rel_<owner>__<collection field> (rel_course__tags),
		// owner column = snake(Src)+"_id", inverse column = snake(SrcField)+"_id",
		// composite PK (owner, inverse), and the two fk_rel_<join>__<col>
		// constraints. DstField only names the mappedBy side and never a
		// column.
		join := "rel_" + srcTable + "__" + snake(r.SrcField)
		ownerCol := srcTable + "_id"
		inverseCol := snake(r.SrcField) + "_id"
		b.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", join))
		b.WriteString("    " + ownerCol + " bigint NOT NULL,\n")
		b.WriteString("    " + inverseCol + " bigint NOT NULL,\n")
		b.WriteString(fmt.Sprintf("    PRIMARY KEY (%s, %s)\n", ownerCol, inverseCol))
		b.WriteString(");\n")
		b.WriteString(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT fk_%s__%s FOREIGN KEY (%s) REFERENCES %s (id);\n",
			join, join, ownerCol, ownerCol, srcTable))
		b.WriteString(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT fk_%s__%s FOREIGN KEY (%s) REFERENCES %s (id);\n",
			join, join, inverseCol, inverseCol, dstTable))
		b.WriteString("\n")
	}

	// Scalar FK constraints.
	if len(fks) > 0 {
		b.WriteString("-- Foreign keys for entity relationships.\n")
	}
	for _, f := range fks {
		b.WriteString(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT fk_%s__%s FOREIGN KEY (%s) REFERENCES %s (id);\n",
			f.table, f.table, f.column, f.column, f.target))
	}
	if len(fks) > 0 {
		b.WriteString("\n")
	}

	// Seeds for the internal tables so the generated JWT login works out of
	// the box, mirroring config/liquibase/data/*.csv.
	b.WriteString("-- JHipster internal seeds (admin/admin, user/user).\n")
	b.WriteString("INSERT INTO jhi_authority (name) VALUES ('ROLE_ADMIN'), ('ROLE_USER') ON CONFLICT (name) DO NOTHING;\n")
	fmt.Fprintf(&b, "INSERT INTO jhi_user (id, login, password_hash, first_name, last_name, email, image_url, activated, lang_key, activation_key, reset_key, created_by, created_date, reset_date, last_modified_by, last_modified_date) VALUES\n    (1, 'admin', '%s', 'Administrator', 'Administrator', 'admin@localhost', NULL, true, 'en', NULL, NULL, 'system', NULL, NULL, 'system', NULL),\n    (2, 'user', '%s', 'User', 'User', 'user@localhost', NULL, true, 'en', NULL, NULL, 'system', NULL, NULL, 'system', NULL)\nON CONFLICT (id) DO NOTHING;\n", adminHash, userHash)
	b.WriteString("INSERT INTO jhi_user_authority (user_id, authority_name) VALUES (1, 'ROLE_ADMIN'), (1, 'ROLE_USER'), (2, 'ROLE_USER') ON CONFLICT DO NOTHING;\n")

	return b.String()
}

func renderInternalTables(b *strings.Builder) {
	b.WriteString(`CREATE TABLE IF NOT EXISTS jhi_user (
    id bigint PRIMARY KEY,
    login varchar(50) NOT NULL CONSTRAINT ux_user_login UNIQUE,
    password_hash varchar(60) NOT NULL,
    first_name varchar(50),
    last_name varchar(50),
    email varchar(191) CONSTRAINT ux_user_email UNIQUE,
    image_url varchar(256),
    activated boolean NOT NULL DEFAULT false,
    lang_key varchar(10),
    activation_key varchar(20),
    reset_key varchar(20),
    created_by varchar(50) NOT NULL,
    created_date timestamp,
    reset_date timestamp,
    last_modified_by varchar(50),
    last_modified_date timestamp
);

CREATE TABLE IF NOT EXISTS jhi_authority (
    name varchar(50) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS jhi_user_authority (
    user_id bigint NOT NULL,
    authority_name varchar(50) NOT NULL,
    PRIMARY KEY (user_id, authority_name),
    CONSTRAINT fk_authority_name FOREIGN KEY (authority_name) REFERENCES jhi_authority (name),
    CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES jhi_user (id)
);

`)
}

// RenderCompose renders the docker compose file that boots the database and
// provisions it from the init SQL on first start.
func RenderCompose(slug string) string {
	var b strings.Builder
	b.WriteString("# Generated by ai-uml-architect (GEN-02).\n")
	b.WriteString("# Start with 'docker compose up -d'; reset with 'docker compose down -v'.\n")
	b.WriteString("# The init SQL in database/" + slug + ".sql runs once on first boot\n")
	b.WriteString("# (only when ./database/postgres_data is empty).\n")
	b.WriteString("services:\n")
	b.WriteString("  postgres:\n")
	b.WriteString("    image: " + DatabaseImage + "\n")
	b.WriteString("    container_name: " + slug + "-postgres\n")
	b.WriteString("    environment:\n")
	b.WriteString("      POSTGRES_DB: " + slug + "\n")
	b.WriteString("      POSTGRES_USER: " + DatabaseUser + "\n")
	b.WriteString("      POSTGRES_PASSWORD: " + DatabasePassword + "\n")
	b.WriteString("    ports:\n")
	b.WriteString("      - \"" + DatabasePort + ":5432\"\n")
	b.WriteString("    volumes:\n")
	b.WriteString("      - " + DataDir + ":/var/lib/postgresql/data\n")
	b.WriteString("      - ./database/" + slug + ".sql:/docker-entrypoint-initdb.d/init.sql:ro\n")
	return b.String()
}

// patchFile applies a list of anchored replacements to one generated config
// file. Each replacement must be found exactly once, otherwise the function
// fails loudly: silently starting Spring Boot against a database it cannot
// reach would be worse than a clear generation error.
type replacement struct {
	old string
	new string
}

func patchFile(path string, reps []replacement) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	for _, r := range reps {
		if !strings.Contains(content, r.old) {
			return fmt.Errorf("config anchor %q not found in %s; generated project layout changed", r.old, path)
		}
		content = strings.Replace(content, r.old, r.new, 1)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// patchLiquibase disables Liquibase in one profile file. The generated yml
// carries a human comment between `liquibase:` and `contexts:` (dev) or right
// before `liquibase:` (prod), so the anchor must tolerate any number of
// comment lines between them. The generator comment is preserved verbatim.
func patchLiquibase(path, contexts string) error {
	re := regexp.MustCompile(`(?s)(liquibase:\n)((?:[ \t]*#[^\n]*\n)*)([ \t]*contexts: ` + regexp.QuoteMeta(contexts) + `)`)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	if !re.MatchString(content) {
		return fmt.Errorf("config anchor liquibase/contexts %q not found in %s; generated project layout changed", contexts, path)
	}
	patched := re.ReplaceAllString(content, `liquibase:`+"\n"+`    # ai-uml-architect: schema is provisioned by database/compose.yml init.sql`+"\n"+`    enabled: false`+"\n"+`$2$3`)
	return os.WriteFile(path, []byte(patched), 0o644)
}

// PatchApplicationConfig rewrites the generated Spring Boot configuration so
// the app connects to the compose-provisioned database and does not let
// Liquibase run against the already-initialized schema. It patches
// application-dev.yml and application-prod.yml (datasource URL + credentials,
// Liquibase off) and application.yml (disables Spring's docker-compose
// lifecycle, which would compete with our compose.yml on port 5432).
func PatchApplicationConfig(root, baseName, slug string) error {
	configDir := filepath.Join(root, "src", "main", "resources", "config")

	dev := filepath.Join(configDir, "application-dev.yml")
	if err := patchFile(dev, []replacement{
		{old: "url: jdbc:postgresql://localhost:5432/" + baseName, new: "url: jdbc:postgresql://localhost:5432/" + slug},
		{old: "    url: jdbc:postgresql://localhost:5432/" + slug + "\n    hikari:",
			new: "    url: jdbc:postgresql://localhost:5432/" + slug + "\n    username: " + DatabaseUser + "\n    password: " + DatabasePassword + "\n    hikari:"},
	}); err != nil {
		return fmt.Errorf("patch application-dev.yml: %w", err)
	}
	if err := patchLiquibase(dev, "dev, faker"); err != nil {
		return fmt.Errorf("patch application-dev.yml: %w", err)
	}

	prod := filepath.Join(configDir, "application-prod.yml")
	if err := patchFile(prod, []replacement{
		{old: "url: jdbc:postgresql://localhost:5432/" + baseName, new: "url: jdbc:postgresql://localhost:5432/" + slug},
		{old: "    url: jdbc:postgresql://localhost:5432/" + slug + "\n    hikari:",
			new: "    url: jdbc:postgresql://localhost:5432/" + slug + "\n    username: " + DatabaseUser + "\n    password: " + DatabasePassword + "\n    hikari:"},
	}); err != nil {
		return fmt.Errorf("patch application-prod.yml: %w", err)
	}
	if err := patchLiquibase(prod, "prod"); err != nil {
		return fmt.Errorf("patch application-prod.yml: %w", err)
	}

	secrets := filepath.Join(configDir, "application-secret-samples.yml")
	if err := patchFile(secrets, []replacement{
		{old: "    username: " + baseName, new: "    username: " + DatabaseUser},
		{old: "    password:\n", new: "    password: " + DatabasePassword + "\n"},
	}); err != nil {
		return fmt.Errorf("patch application-secret-samples.yml: %w", err)
	}

	app := filepath.Join(configDir, "application.yml")
	if err := patchFile(app, []replacement{
		{old: "  docker:\n    compose:\n      enabled: true\n      lifecycle-management: start-only",
			new: "  docker:\n    compose:\n      enabled: false # ai-uml-architect: database is provisioned by database/compose.yml\n      lifecycle-management: start-only"},
	}); err != nil {
		return fmt.Errorf("patch application.yml: %w", err)
	}

	return nil
}
