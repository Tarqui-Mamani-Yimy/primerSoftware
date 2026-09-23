// End-to-end test against the real pinned generator. It is skipped unless
// JHIPSTER_E2E=1: the run downloads generator-jhipster@9.4.0 into the pnpm
// store on first use and takes minutes, so it must never run on a normal
// unit-test pass.
//
// The test generates a document that exercises every supported relationship
// shape plus the blob/typed columns, unzips the artifact, and cross-checks
// the provisioned init SQL against the generator's own Liquibase changelogs.
// A mismatch there is drift in the schema contract and fails the test.
package jhipster

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
)

func e2eDoc() domain.DiagramDocument {
	strp := func(s string) *string { return &s }
	return domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "E2EProbe",
		Classes: []domain.UmlClass{
			{ID: "c1", Name: "Course", Attributes: []domain.Attribute{{ID: "a1", Name: "title", Type: "string"}}},
			{ID: "c2", Name: "Tag", Attributes: []domain.Attribute{{ID: "a2", Name: "label", Type: "string"}}},
			// "contraseña" exercises GBU-03 transliteration end to end: the field
		// must survive as "contrasena", not the accent-dropped "contrasea".
		{ID: "c3", Name: "Student", Attributes: []domain.Attribute{
			{ID: "a3", Name: "name", Type: "String"},
			{ID: "a3b", Name: "contraseña", Type: "String"},
		}},
			{ID: "c4", Name: "Profile", Attributes: []domain.Attribute{{ID: "a4", Name: "bio", Type: "text"}}},
			{ID: "c5", Name: "Enrollment", Attributes: []domain.Attribute{{ID: "a5", Name: "grade", Type: "int"}}},
			{ID: "c6", Name: "Image", Attributes: []domain.Attribute{{ID: "a6", Name: "data", Type: "blob"}}},
			// Order is a PostgreSQL-reserved table name (-> jhi_order) and
			// "user" a PostgreSQL-reserved column name (-> jhi_user); both
			// must come out prefixed here exactly as the real generator
			// prefixes them.
			{ID: "c7", Name: "Order", Attributes: []domain.Attribute{{ID: "a7", Name: "user", Type: "string"}}},
			// "User" collides with JHipster's own built-in User entity; jdlgen
			// must rename it to "AppUser" so the generator creates a real
			// app_user table instead of silently merging/disregarding it.
			{ID: "c8", Name: "User", Attributes: []domain.Attribute{{ID: "a8", Name: "nickname", Type: "string"}}},
		},
		Relationships: []domain.Relationship{
			// ManyToMany: Course{tag} -> Tag{course} (GBU-04: JDL declares the
			// singular stem; the real generator's own pluralize() computes
			// the JSON property names "tags"/"courses").
			{ID: "r1", SourceID: "c1", TargetID: "c2", Type: "association", SourceMultiplicity: strp("*"), TargetMultiplicity: strp("*")},
			// OneToOne: Profile{student} -> Student{profile}
			{ID: "r2", SourceID: "c4", TargetID: "c3", Type: "association", SourceMultiplicity: strp("1"), TargetMultiplicity: strp("1")},
			// OneToMany: Enrollment{student} -> Student{enrollment}
			{ID: "r3", SourceID: "c5", TargetID: "c3", Type: "association", SourceMultiplicity: strp("1"), TargetMultiplicity: strp("*")},
			// ManyToOne: Order{course} -> Course{order}
			{ID: "r4", SourceID: "c7", TargetID: "c1", Type: "association", SourceMultiplicity: strp("*"), TargetMultiplicity: strp("1")},
			// ManyToOne: Order{appUser} -> AppUser{order} (renamed User)
			{ID: "r5", SourceID: "c7", TargetID: "c8", Type: "association", SourceMultiplicity: strp("*"), TargetMultiplicity: strp("1")},
		},
	}
}

func TestGenerateE2EAgainstRealGenerator(t *testing.T) {
	if os.Getenv("JHIPSTER_E2E") != "1" {
		t.Skip("set JHIPSTER_E2E=1 to run the real generator end to end")
	}
	g := NewGenerator()
	opts := jdlgen.DefaultOptions()
	opts.BaseName = "UmlArchitect" // drives the derived database slug: uml-architect
	res, err := g.Generate(context.Background(), e2eDoc(), opts)
	if err != nil {
		t.Fatalf("Generate (real pnpm): %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(res.Content), int64(len(res.Content)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	entries := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		entries[f.Name] = string(data)
	}

	prefix := "UmlArchitect/"
	file := func(rel string) string {
		v, ok := entries[prefix+rel]
		if !ok {
			t.Fatalf("zip missing %s", rel)
		}
		return v
	}

	// Provisioned files must exist.
	initSQL := file("database/uml-architect.sql")
	compose := file("compose.yml")
	for _, want := range []string{
		"image: postgres:16-alpine",
		"POSTGRES_DB: uml-architect",
		"- ./database/uml-architect.sql:/docker-entrypoint-initdb.d/init.sql:ro",
	} {
		if !strings.Contains(compose, want) {
			t.Errorf("compose missing %q:\n%s", want, compose)
		}
	}

	// Cross-check the join table against the generator's real changelog.
	// JHipster names it rel_<owner>__<collection field> and its FKs
	// fk_rel_<join>__<column>; our init SQL must match byte-for-byte on the
	// names, not just be plausible.
	joinTable := "rel_course__tag"
	if !strings.Contains(initSQL, "CREATE TABLE IF NOT EXISTS "+joinTable) {
		t.Errorf("init SQL missing join table %s", joinTable)
	}
	for _, fk := range []string{
		"fk_rel_course__tag__course_id",
		"fk_rel_course__tag__tag_id",
		"fk_profile__student_id",
	} {
		if !strings.Contains(initSQL, fk) {
			t.Errorf("init SQL missing constraint %s", fk)
		}
		// Same constraint must exist in the generator's changelogs.
		if !changelogContains(entries, prefix, fk) {
			t.Errorf("changelogs do not contain %s (schema drift)", fk)
		}
	}
	if !changelogContains(entries, prefix, joinTable) {
		t.Errorf("changelogs do not contain join table %s (schema drift)", joinTable)
	}
	// The OneToOne unique index must exist on both sides.
	if !strings.Contains(initSQL, "ux_profile__student_id") {
		t.Errorf("init SQL missing unique ux_profile__student_id")
	}
	if !changelogContains(entries, prefix, "ux_profile__student_id") {
		t.Errorf("changelogs do not contain ux_profile__student_id (schema drift)")
	}

	// Blob columns get a content type column in both schemas.
	if !strings.Contains(initSQL, "data_content_type varchar(255)") {
		t.Errorf("init SQL missing blob content-type column")
	}
	if !changelogContains(entries, prefix, "data_content_type") {
		t.Errorf("changelogs do not contain data_content_type (schema drift)")
	}

	// GBU-03: "contraseña" must transliterate to "contrasena" end to end —
	// SQL column, JPA domain field, and the real generator must agree.
	if !strings.Contains(initSQL, "contrasena varchar(255)") {
		t.Errorf("init SQL missing transliterated column contrasena:\n%s", initSQL)
	}
	if strings.Contains(initSQL, "contrasea ") {
		t.Errorf("init SQL must not contain the accent-dropped column contrasea:\n%s", initSQL)
	}
	if !changelogContains(entries, prefix, "contrasena") {
		t.Errorf("changelogs do not contain contrasena (schema drift)")
	}
	studentDomain := file("src/main/java/com/umlarchitect/domain/Student.java")
	if !strings.Contains(studentDomain, "contrasena") {
		t.Errorf("Student.java missing transliterated field contrasena:\n%s", studentDomain)
	}
	if strings.Contains(studentDomain, "contrasea;") || strings.Contains(studentDomain, "contrasea ") {
		t.Errorf("Student.java must not contain the accent-dropped field contrasea:\n%s", studentDomain)
	}

	// Reserved-keyword naming: ORDER (table) and USER (column) are both
	// PostgreSQL-reserved in generator-jhipster 9.4.0's own list, so both
	// come out jhi_-prefixed; our init SQL must match the real changelogs
	// byte-for-byte on these names too, not just avoid a syntax error.
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS jhi_order (",
		"jhi_user varchar(255)",
	} {
		if !strings.Contains(initSQL, want) {
			t.Errorf("init SQL missing %q (reserved-keyword naming):\n%s", want, initSQL)
		}
	}
	if strings.Contains(initSQL, "IF NOT EXISTS order (") {
		t.Errorf("init SQL declares the unprefixed reserved table name \"order\":\n%s", initSQL)
	}
	for _, fragment := range []string{"jhi_order", "jhi_user"} {
		if !changelogContains(entries, prefix, fragment) {
			t.Errorf("changelogs do not contain %s (schema drift)", fragment)
		}
	}

	// Internal JHipster tables and seeds must exist in our SQL.
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS jhi_user (",
		"CREATE TABLE IF NOT EXISTS jhi_authority (",
		"INSERT INTO jhi_user_authority (user_id, authority_name)",
	} {
		if !strings.Contains(initSQL, want) {
			t.Errorf("init SQL missing %q", want)
		}
	}

	// The UML class "User" collides with JHipster's built-in User entity, so
	// jdlgen must rename it to AppUser; the real generator must then create
	// a genuine app_user table (not merge/disregard it), and the internal
	// jhi_user table must stay singular (no collision).
	if !strings.Contains(initSQL, "CREATE TABLE IF NOT EXISTS app_user (") {
		t.Errorf("init SQL missing renamed table app_user:\n%s", initSQL)
	}
	if strings.Count(initSQL, "CREATE TABLE IF NOT EXISTS jhi_user (") != 1 {
		t.Errorf("init SQL must declare jhi_user exactly once (no collision), got:\n%s", initSQL)
	}
	if !changelogContains(entries, prefix, "app_user") {
		t.Errorf("changelogs do not contain app_user (schema drift)")
	}

	// Patched datasource: Liquibase off + our database, on the non-clashing
	// host port (GBU-02: the project's own Postgres already owns 5432).
	dev := file("src/main/resources/config/application-dev.yml")
	for _, want := range []string{
		"jdbc:postgresql://localhost:5433/uml-architect",
		"username: devuser",
		"enabled: false",
	} {
		if !strings.Contains(dev, want) {
			t.Errorf("patched application-dev.yml missing %q:\n%s", want, dev)
		}
	}

	// GBU-02: serverPort in the JDL application config drives server.port in
	// both profiles directly (application-{dev,prod}.yml.ejs render
	// "port: <%- serverPort %>"), avoiding a clash with the project's own Go
	// backend on 8080.
	prodForPort := file("src/main/resources/config/application-prod.yml")
	for _, f := range map[string]string{"application-dev.yml": dev, "application-prod.yml": prodForPort} {
		if !strings.Contains(f, "\nserver:\n  port: 8081\n") && !strings.Contains(f, "server:\n  port: 8081") {
			t.Errorf("%s missing server.port 8081:\n%s", f, f)
		}
	}

	// The generated compose file must publish Postgres on 5433 (container
	// port stays 5432).
	if !strings.Contains(compose, `"5433:5432"`) {
		t.Errorf("compose must publish postgres on host port 5433:\n%s", compose)
	}

	// CORS: a separately developed frontend's local dev origins must be
	// allowed alongside the JHipster Ionic defaults, and the original
	// allowed-origins must survive the patch.
	for _, want := range []string{
		"http://localhost:8100", "https://localhost:8100",
		"http://localhost:5173", "http://127.0.0.1:5173",
		"http://localhost:3000", "http://127.0.0.1:3000",
		"http://localhost:4200",
	} {
		if !strings.Contains(dev, want) {
			t.Errorf("patched application-dev.yml missing CORS origin %q:\n%s", want, dev)
		}
	}
	if !strings.Contains(dev, `exposed-headers: 'Authorization,Link,X-Total-Count`) {
		t.Errorf("patched application-dev.yml missing exposed-headers with Authorization,Link,X-Total-Count:\n%s", dev)
	}

	// prod must stay untouched: enabling CORS for arbitrary origins there is a
	// separate, explicit decision this generator does not make.
	prod := file("src/main/resources/config/application-prod.yml")
	if strings.Contains(prod, "5173") {
		t.Errorf("application-prod.yml must not gain the local dev CORS origins:\n%s", prod)
	}
}

// TestGenerateE2EPluralNamingAcceptanceScenario covers GBU-04's specific
// acceptance requirement against the real generator: an entity named "Roles"
// (a ManyToMany relationship to "Permisos", also already plural-looking in
// Spanish) and an entity "Usuario" (OneToMany from Roles) — the exact shape
// reported in the 2026-09-23 acceptance run, where the old naive pluralField
// produced JDL fields "roleses"/"permisoses" and the real generator's own
// pluralize() then re-pluralized them to "roleseses"/"permisoseses" in the
// generated DTOs. Asserts the actual generated Java DTO field names directly.
func TestGenerateE2EPluralNamingAcceptanceScenario(t *testing.T) {
	if os.Getenv("JHIPSTER_E2E") != "1" {
		t.Skip("set JHIPSTER_E2E=1 to run the real generator end to end")
	}
	strp := func(s string) *string { return &s }
	doc := domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "PluralProbe",
		Classes: []domain.UmlClass{
			{ID: "roles", Name: "Roles", Attributes: []domain.Attribute{{ID: "a1", Name: "nombre", Type: "string"}}},
			{ID: "permisos", Name: "Permisos", Attributes: []domain.Attribute{{ID: "a2", Name: "nombre", Type: "string"}}},
			{ID: "usuario", Name: "Usuario", Attributes: []domain.Attribute{{ID: "a3", Name: "correo", Type: "string"}}},
		},
		Relationships: []domain.Relationship{
			{ID: "r1", SourceID: "roles", TargetID: "permisos", Type: "association",
				SourceMultiplicity: strp("*"), TargetMultiplicity: strp("*")},
			{ID: "r2", SourceID: "roles", TargetID: "usuario", Type: "association",
				SourceMultiplicity: strp("1"), TargetMultiplicity: strp("*")},
		},
	}
	g := NewGenerator()
	opts := jdlgen.DefaultOptions()
	opts.BaseName = "PluralProbe"
	res, err := g.Generate(context.Background(), doc, opts)
	if err != nil {
		t.Fatalf("Generate (real pnpm): %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(res.Content), int64(len(res.Content)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	entries := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		entries[f.Name] = string(data)
	}
	prefix := "PluralProbe/"
	rolesDTO := entries[prefix+"src/main/java/com/umlarchitect/service/dto/RolesDTO.java"]
	permisosDTO := entries[prefix+"src/main/java/com/umlarchitect/service/dto/PermisosDTO.java"]
	if rolesDTO == "" || permisosDTO == "" {
		t.Fatalf("zip missing RolesDTO.java/PermisosDTO.java, have %v", keysOf(entries))
	}
	// Correct plurals, computed once by the real generator's own pluralize():
	// RolesDTO.permisos, PermisosDTO.roles. Never the double-pluralized
	// "permisoses"/"roleses"/"roleseses"/"permisoseses" from the old bug.
	for _, want := range []string{"Set<PermisosDTO> permisos", "getPermisos()", "setPermisos("} {
		if !strings.Contains(rolesDTO, want) {
			t.Errorf("RolesDTO.java missing %q:\n%s", want, rolesDTO)
		}
	}
	for _, want := range []string{"Set<RolesDTO> roles", "getRoles()", "setRoles("} {
		if !strings.Contains(permisosDTO, want) {
			t.Errorf("PermisosDTO.java missing %q:\n%s", want, permisosDTO)
		}
	}
	for _, bad := range []string{"roleses", "permisoses", "roleseses", "permisoseses"} {
		if strings.Contains(rolesDTO, bad) {
			t.Errorf("RolesDTO.java must not contain double-plural %q:\n%s", bad, rolesDTO)
		}
		if strings.Contains(permisosDTO, bad) {
			t.Errorf("PermisosDTO.java must not contain double-plural %q:\n%s", bad, permisosDTO)
		}
	}
	// OneToMany Roles -> Usuario: the domain's back-reference collection
	// (Roles.usuarios) must be a plain, correctly-pluralized "usuarios", not
	// "usuarioses" from a stray double-pluralization.
	rolesDomain := entries[prefix+"src/main/java/com/umlarchitect/domain/Roles.java"]
	if rolesDomain == "" {
		t.Fatalf("zip missing Roles.java, have %v", keysOf(entries))
	}
	if !strings.Contains(rolesDomain, "Set<Usuario> usuarios") {
		t.Errorf("Roles.java missing correctly-pluralized field usuarios:\n%s", rolesDomain)
	}
	if strings.Contains(rolesDomain, "usuarioses") {
		t.Errorf("Roles.java must not contain double-plural usuarioses:\n%s", rolesDomain)
	}
}

// keysOf returns the keys of a string-keyed map, for diagnostic output.
func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// changelogContains reports whether any Liquibase changelog in the artifact
// contains the fragment.
func changelogContains(entries map[string]string, prefix, fragment string) bool {
	for name, content := range entries {
		if !strings.HasPrefix(name, prefix+"src/main/resources/config/liquibase/") {
			continue
		}
		if strings.Contains(content, fragment) {
			return true
		}
	}
	return false
}

var _ = filepath.Join // keep path import if assertions change
