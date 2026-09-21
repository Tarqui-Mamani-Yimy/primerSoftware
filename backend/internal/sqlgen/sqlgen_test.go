package sqlgen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
	"github.com/ai-uml-architect/gobackend/internal/sqlgen"
)

func strp(s string) *string { return &s }

func modelFromDoc(doc domain.DiagramDocument) jdlgen.Model {
	m, _ := jdlgen.BuildModel(doc)
	return m
}

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"UmlArchitect", "uml-architect"},
		{"MyProject", "my-project"},
		{"ProbeApp", "probe-app"},
		{"Shop", "shop"},
		{"Order2", "order2"},
		{"UMLArchitect", "uml-architect"},
		{"ACME2FullStack", "acme2-full-stack"},
	}
	for _, tc := range cases {
		got, err := sqlgen.Slugify(tc.in)
		if err != nil {
			t.Errorf("Slugify(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func probeDoc() domain.DiagramDocument {
	return domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Probe",
		Classes: []domain.UmlClass{
			{ID: "c1", Name: "Course", Attributes: []domain.Attribute{{ID: "a1", Name: "title", Type: "string"}}},
			{ID: "c2", Name: "Student", Attributes: []domain.Attribute{{ID: "a2", Name: "name", Type: "String"}}},
			{ID: "c3", Name: "Enrollment", Attributes: []domain.Attribute{{ID: "a3", Name: "grade", Type: "int"}}},
			{ID: "c4", Name: "Tag", Attributes: []domain.Attribute{{ID: "a4", Name: "label", Type: "string"}}},
			{ID: "c5", Name: "Profile", Attributes: []domain.Attribute{{ID: "a5", Name: "bio", Type: "text"}}},
			{
				ID: "c6", Name: "Zoo",
				Attributes: []domain.Attribute{
					{ID: "z1", Name: "big", Type: "bigdecimal"},
					{ID: "z2", Name: "fl", Type: "float"},
					{ID: "z3", Name: "db", Type: "double"},
					{ID: "z4", Name: "flag", Type: "boolean"},
					{ID: "z5", Name: "whenAt", Type: "datetime"},
					{ID: "z6", Name: "date2", Type: "localdate"},
					{ID: "z7", Name: "uid", Type: "uuid"},
					{ID: "z8", Name: "body", Type: "blob"},
					{ID: "z9", Name: "note", Type: "text"},
				},
			},
			{ID: "c7", Name: "Order", Attributes: []domain.Attribute{{ID: "o1", Name: "total", Type: "double"}}},
		},
		Relationships: []domain.Relationship{
			{ID: "r1", SourceID: "c3", TargetID: "c1", Type: "association", SourceMultiplicity: strp("1"), TargetMultiplicity: strp("1")},
			{ID: "r2", SourceID: "c3", TargetID: "c2", Type: "association", SourceMultiplicity: strp("1"), TargetMultiplicity: strp("*")},
			{ID: "r3", SourceID: "c1", TargetID: "c4", Type: "association", SourceMultiplicity: strp("*"), TargetMultiplicity: strp("*")},
			{ID: "r4", SourceID: "c5", TargetID: "c2", Type: "association", SourceMultiplicity: strp("1"), TargetMultiplicity: strp("1")},
			{ID: "r5", SourceID: "c7", TargetID: "c7", Type: "association", SourceMultiplicity: strp("*"), TargetMultiplicity: strp("*")},
		},
	}
}

func TestRenderSQLStructure(t *testing.T) {
	m := modelFromDoc(probeDoc())
	sql := sqlgen.RenderSQL(m)

	wants := []string{
		"CREATE SEQUENCE IF NOT EXISTS sequence_generator START WITH 1050 INCREMENT BY 50;",
		"CREATE TABLE IF NOT EXISTS jhi_user (",
		`login varchar(50) NOT NULL CONSTRAINT ux_user_login UNIQUE,`,
		`email varchar(191) CONSTRAINT ux_user_email UNIQUE,`,
		"CREATE TABLE IF NOT EXISTS jhi_authority (",
		"CREATE TABLE IF NOT EXISTS jhi_user_authority (",
		"    id bigint PRIMARY KEY",
		"CREATE TABLE IF NOT EXISTS course (",
		"    title varchar(255)",
		"CREATE TABLE IF NOT EXISTS enrollment (",
		"    course_id bigint",
		"CREATE TABLE IF NOT EXISTS student (",
		"    enrollment_id bigint",
		"ALTER TABLE student ADD CONSTRAINT fk_student__enrollment_id FOREIGN KEY (enrollment_id) REFERENCES enrollment (id);",
		"CREATE TABLE IF NOT EXISTS profile (",
		"    student_id bigint CONSTRAINT ux_profile__student_id UNIQUE",
		"CREATE TABLE IF NOT EXISTS rel_course__tags (",
		"    course_id bigint NOT NULL,",
		"    tags_id bigint NOT NULL,",
		"    PRIMARY KEY (course_id, tags_id)",
		"ALTER TABLE enrollment ADD CONSTRAINT fk_enrollment__course_id FOREIGN KEY (course_id) REFERENCES course (id);",
		"ALTER TABLE profile ADD CONSTRAINT fk_profile__student_id FOREIGN KEY (student_id) REFERENCES student (id);",
		"ALTER TABLE rel_course__tags ADD CONSTRAINT fk_rel_course__tags__course_id FOREIGN KEY (course_id) REFERENCES course (id);",
		"ALTER TABLE rel_course__tags ADD CONSTRAINT fk_rel_course__tags__tags_id FOREIGN KEY (tags_id) REFERENCES tag (id);",
		"-- JHipster internal seeds (admin/admin, user/user).",
		"INSERT INTO jhi_authority (name) VALUES ('ROLE_ADMIN'), ('ROLE_USER') ON CONFLICT (name) DO NOTHING;",
		"(1, 'admin', '$2a$10$gSAhZrxMllrbgj/kkK9UceBPpChGWJA7SYIb1Mqo.n5aNLq1/oRrC', 'Administrator',",
		"(2, 'user', '$2a$10$VEjxo0jq2YG9Rbk2HmX9S.k1uZBGYUHdUcid3g/vfiEl7lwWgOH/K', 'User',",
		"INSERT INTO jhi_user_authority (user_id, authority_name) VALUES (1, 'ROLE_ADMIN'), (1, 'ROLE_USER'), (2, 'ROLE_USER') ON CONFLICT DO NOTHING;",
		// Zoo types: decimal, floats, boolean, timestamp, uuid, blob+content_type
		"    big decimal(21,2)",
		"    fl float4",
		"    db double precision",
		"    flag boolean",
		"    when_at timestamp",
		"    date2 date",
		"    uid uuid",
		"    body bytea",
		"    body_content_type varchar(255)",
		"    note text",
		// OneToMany FK placement: Enrollment{student} -> Student{enrollments}
		"CREATE TABLE IF NOT EXISTS student (",
		// Self ManyToMany join table for Order{orders} to Order{orders}
		"CREATE TABLE IF NOT EXISTS rel_order__orders (",
		"    order_id bigint NOT NULL,",
		"    orders_id bigint NOT NULL,",
		"    PRIMARY KEY (order_id, orders_id)",
		"ALTER TABLE rel_order__orders ADD CONSTRAINT fk_rel_order__orders__order_id FOREIGN KEY (order_id) REFERENCES order (id);",
		"ALTER TABLE rel_order__orders ADD CONSTRAINT fk_rel_order__orders__orders_id FOREIGN KEY (orders_id) REFERENCES order (id);",
		// The old "self-association; no join table" comment must not remain.
	}
	for _, want := range wants {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL missing %q\n---\n%s", want, sql)
		}
	}

	// Deterministic: rendering twice must be byte-identical.
	if again := sqlgen.RenderSQL(m); again != sql {
		t.Error("RenderSQL is not deterministic")
	}
}

// TestRenderSQLFKOwnershipMatchesColumnTable asserts that every FK column
// declared inside a CREATE TABLE has its ALTER TABLE ... FOREIGN KEY on the
// SAME table. A column declared in table A with a constraint on table B is
// a broken schema that PostgreSQL rejects at runtime.
func TestRenderSQLFKOwnershipMatchesColumnTable(t *testing.T) {
	m := modelFromDoc(probeDoc())
	sql := sqlgen.RenderSQL(m)

	tables := map[string]map[string]bool{}
	var currentTable string

	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "CREATE TABLE IF NOT EXISTS ") {
			header := strings.TrimPrefix(trimmed, "CREATE TABLE IF NOT EXISTS ")
			header = strings.TrimSuffix(header, " (")
			currentTable = strings.TrimSpace(header)
			tables[currentTable] = map[string]bool{}
			continue
		}
		if trimmed == ");" {
			currentTable = ""
			continue
		}
		if currentTable != "" {
			if strings.HasPrefix(trimmed, "PRIMARY KEY") || strings.HasPrefix(trimmed, "CONSTRAINT") {
				continue
			}
			fields := strings.Fields(trimmed)
			if len(fields) >= 1 {
				col := strings.TrimSuffix(fields[0], ",")
				tables[currentTable][col] = true
			}
		}
	}

	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "ALTER TABLE ") {
			continue
		}
		parts := strings.Fields(trimmed)
		if len(parts) < 3 {
			continue
		}
		alterTable := parts[2]

		fkIdx := strings.Index(trimmed, "FOREIGN KEY (")
		if fkIdx == -1 {
			continue
		}
		colStart := fkIdx + len("FOREIGN KEY (")
		colEnd := strings.Index(trimmed[colStart:], ")")
		colName := trimmed[colStart : colStart+colEnd]

		cols, exists := tables[alterTable]
		if !exists {
			t.Errorf("ALTER TABLE references unknown table %q", alterTable)
			continue
		}
		if !cols[colName] {
			t.Errorf("Schema mismatch: ALTER TABLE %s references column %q, but column %q was NOT declared in CREATE TABLE %s (declared columns: %v)",
				alterTable, colName, colName, alterTable, cols)
		}
	}
}

func TestRenderSQLRelationshipsUsePluralCollectionNames(t *testing.T) {
	m := modelFromDoc(probeDoc())
	sql := sqlgen.RenderSQL(m)
	// The OneToMany collection declared in plural form (enrollments) must not
	// appear as a column anywhere; only the FK-holding side (student_id) does.
	if strings.Contains(sql, "enrollments_id") {
		t.Errorf("plural collection field leaked as a column:\n%s", sql)
	}
}

func TestRenderCompose(t *testing.T) {
	yml := sqlgen.RenderCompose("my-project")
	for _, want := range []string{
		"image: postgres:16-alpine",
		"container_name: my-project-postgres",
		"POSTGRES_DB: my-project",
		"POSTGRES_USER: devuser",
		"POSTGRES_PASSWORD: devpassword",
		`      - "5432:5432"`,
		"- ./database/postgres_data:/var/lib/postgresql/data",
		"- ./database/my-project.sql:/docker-entrypoint-initdb.d/init.sql:ro",
		"rm -rf database/postgres_data",
	} {
		if !strings.Contains(yml, want) {
			t.Errorf("compose missing %q\n---\n%s", want, yml)
		}
	}
}

func writeFixture(t *testing.T, root, name, content string) {
	t.Helper()
	dir := filepath.Join(root, "src", "main", "resources", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const devFixture = `spring:
  devtools:
    restart:
      enabled: true
  datasource:
    type: com.zaxxer.hikari.HikariDataSource
    url: jdbc:postgresql://localhost:5432/ProbeApp
    hikari:
      poolName: Hikari
      auto-commit: false
  liquibase:
    # Remove 'faker' if you do not want the sample data to be loaded automatically
    contexts: dev, faker
  mail:
    host: localhost
`

const prodFixture = `spring:
  datasource:
    type: com.zaxxer.hikari.HikariDataSource
    url: jdbc:postgresql://localhost:5432/ProbeApp
    hikari:
      poolName: Hikari
      auto-commit: false
  # Replace by 'prod, faker' to add the faker context and have sample data loaded in production
  liquibase:
    contexts: prod
  mail:
    host: localhost
`

const secretsFixture = `logging:
  level:
    org.springframework.security: DEBUG

spring:
  datasource:
    username: ProbeApp
    password:

jhipster:
  security:
    authentication:
      jwt:
        base64-secret: dGVzdA==
`

const appFixture = `spring:
  application:
    name: ProbeApp
  docker:
    compose:
      enabled: true
      lifecycle-management: start-only
      file: src/main/docker/services.yml
  profiles:
    active: '@spring.profiles.active@'
`

func TestPatchApplicationConfig(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "application-dev.yml", devFixture)
	writeFixture(t, root, "application-prod.yml", prodFixture)
	writeFixture(t, root, "application-secret-samples.yml", secretsFixture)
	writeFixture(t, root, "application.yml", appFixture)

	if err := sqlgen.PatchApplicationConfig(root, "ProbeApp", "probe-app"); err != nil {
		t.Fatalf("PatchApplicationConfig: %v", err)
	}

	read := func(name string) string {
		data, err := os.ReadFile(filepath.Join(root, "src", "main", "resources", "config", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for name, want := range map[string]string{
		"application-dev.yml":            `url: jdbc:postgresql://localhost:5432/probe-app`,
		"application-prod.yml":           `url: jdbc:postgresql://localhost:5432/probe-app`,
		"application-secret-samples.yml": "username: devuser",
		"application.yml":                "enabled: false # ai-uml-architect: database is provisioned by database/compose.yml",
	} {
		if !strings.Contains(read(name), want) {
			t.Errorf("%s missing %q after patch:\n%s", name, want, read(name))
		}
	}
	// No unpatched defaults remain anywhere.
	for name, bad := range map[string]string{
		"application-dev.yml":            "localhost:5432/ProbeApp",
		"application-prod.yml":           "localhost:5432/ProbeApp",
		"application-secret-samples.yml": "username: ProbeApp",
		"application.yml":                "enabled: true",
	} {
		if strings.Contains(read(name), bad) {
			t.Errorf("%s still contains %q after patch:\n%s", name, bad, read(name))
		}
	}
	// Liquibase must be explicitly off in both profile files.
	for _, name := range []string{"application-dev.yml", "application-prod.yml"} {
		if !strings.Contains(read(name), "enabled: false") {
			t.Errorf("%s does not disable liquibase:\n%s", name, read(name))
		}
	}
}

func TestPatchApplicationConfigMissingAnchor(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "application-dev.yml", devFixture)
	writeFixture(t, root, "application-prod.yml", prodFixture)
	writeFixture(t, root, "application-secret-samples.yml", secretsFixture)
	// application.yml is missing entirely (layout drift simulation).
	if err := sqlgen.PatchApplicationConfig(root, "ProbeApp", "probe-app"); err == nil {
		t.Fatal("expected error for missing application.yml, got nil")
	}
}
