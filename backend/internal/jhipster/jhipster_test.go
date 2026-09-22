package jhipster

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
)

// fakeRunner simulates the pinned generator: it records the invoked command
// and materializes the files it "generated" into the workdir. contents is
// optional; when absent a file gets a placeholder byte so the flow can be
// tested without the generator.
type fakeRunner struct {
	calls          []string
	generatedFiles []string
	contents       map[string]string
	out            string
	err            error
}

func (f *fakeRunner) Run(ctx context.Context, dir, name string, args ...string) (string, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	if f.err != nil {
		if f.out == "" {
			// Default failure output includes the ephemeral workdir, which the
			// generator must strip from the surfaced error message.
			return "fatal: " + dir + "/model.jdl boom", f.err
		}
		return f.out, f.err
	}
	for _, rel := range f.generatedFiles {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return "", err
		}
		body := "x"
		if c, ok := f.contents[rel]; ok {
			body = c
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			return "", err
		}
	}
	return f.out, nil
}

func alphaDoc() domain.DiagramDocument {
	return domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Shop",
		Classes: []domain.UmlClass{
			{ID: "a", Name: "Alpha", Attributes: []domain.Attribute{{ID: "a1", Name: "total", Type: "BigDecimal"}}},
		},
	}
}

func zipEntries(t *testing.T, content []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	entries := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read entry %s: %v", f.Name, err)
		}
		entries[f.Name] = data
	}
	return entries
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestGenerateProducesZipWithManifest(t *testing.T) {
	// The provisioned Spring config files must carry the exact anchors the
	// sqlgen patch operates on; placeholder content would fail the patch and
	// surface a provisioning error instead of an artifact.
	devYml := `spring:
  devtools:
    restart:
      enabled: true
  datasource:
    type: com.zaxxer.hikari.HikariDataSource
    url: jdbc:postgresql://localhost:5432/UmlArchitect
    hikari:
      poolName: Hikari
      auto-commit: false
  liquibase:
    contexts: dev, faker
jhipster:
  cors:
    allowed-origins: 'http://localhost:8100,https://localhost:8100'
    allowed-origin-patterns: 'https://*.githubpreview.dev'
    allowed-methods: '*'
    allowed-headers: '*'
    exposed-headers: 'Authorization,Link,X-Total-Count,X-${jhipster.clientApp.name}-alert,X-${jhipster.clientApp.name}-error,X-${jhipster.clientApp.name}-params'
    allow-credentials: true
    max-age: 1800
`
	prodYml := `spring:
  datasource:
    type: com.zaxxer.hikari.HikariDataSource
    url: jdbc:postgresql://localhost:5432/UmlArchitect
    hikari:
      poolName: Hikari
      auto-commit: false
  liquibase:
    contexts: prod
`
	secretsYml := `spring:
  datasource:
    username: UmlArchitect
    password:
`
	appYml := `spring:
  application:
    name: UmlArchitect
  docker:
    compose:
      enabled: true
      lifecycle-management: start-only
      file: src/main/docker/services.yml
`
	runner := &fakeRunner{
		generatedFiles: []string{
			"pom.xml",
			".yo-rc.json",
			"src/main/java/com/umlarchitect/Alpha.java",
			"node_modules/pkg/index.js", // must be excluded
			".git/config",               // must be excluded
			"src/main/resources/config/application-dev.yml",
			"src/main/resources/config/application-prod.yml",
			"src/main/resources/config/application-secret-samples.yml",
			"src/main/resources/config/application.yml",
		},
		contents: map[string]string{
			"src/main/resources/config/application-dev.yml":            devYml,
			"src/main/resources/config/application-prod.yml":           prodYml,
			"src/main/resources/config/application-secret-samples.yml": secretsYml,
			"src/main/resources/config/application.yml":                appYml,
		},
	}
	g := &Generator{Runner: runner, Version: Version, Timeout: 0}
	res, err := g.Generate(context.Background(), alphaDoc(), jdlgen.DefaultOptions())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if res.FileName != "UmlArchitect-jhipster-backend.zip" {
		t.Errorf("FileName = %q", res.FileName)
	}
	wantCmd := "pnpm dlx --ignore-scripts generator-jhipster@9.4.0 jdl model.jdl --force --skip-install"
	if len(runner.calls) != 1 || runner.calls[0] != wantCmd {
		t.Errorf("runner calls = %v, want [%s]", runner.calls, wantCmd)
	}

	entries := zipEntries(t, res.Content)
	for _, want := range []string{
		"manifest.json",
		"UmlArchitect/pom.xml",
		"UmlArchitect/.yo-rc.json",
		"UmlArchitect/src/main/java/com/umlarchitect/Alpha.java",
		"UmlArchitect/model.jdl",
		"UmlArchitect/compose.yml",
		"UmlArchitect/database/uml-architect.sql",
	} {
		if _, ok := entries[want]; !ok {
			t.Errorf("zip missing entry %q (have %v)", want, keys(entries))
		}
	}
	// The init SQL must be the rendered schema, not a placeholder.
	initSQL := string(entries["UmlArchitect/database/uml-architect.sql"])
	for _, want := range []string{
		"CREATE SEQUENCE IF NOT EXISTS sequence_generator",
		"CREATE TABLE IF NOT EXISTS jhi_user (",
		"CREATE TABLE IF NOT EXISTS alpha (",
		"    total decimal(21,2)",
		"INSERT INTO jhi_authority (name) VALUES ('ROLE_ADMIN'), ('ROLE_USER')",
	} {
		if !strings.Contains(initSQL, want) {
			t.Errorf("init SQL missing %q:\n%s", want, initSQL)
		}
	}
	// The patched configs must ship with the provisioned datasource.
	for path, want := range map[string]string{
		"UmlArchitect/src/main/resources/config/application-dev.yml":            "url: jdbc:postgresql://localhost:5432/uml-architect",
		"UmlArchitect/src/main/resources/config/application.yml":                "enabled: false # ai-uml-architect: database is provisioned by database/compose.yml",
		"UmlArchitect/src/main/resources/config/application-secret-samples.yml": "username: devuser",
	} {
		if !strings.Contains(string(entries[path]), want) {
			t.Errorf("%s missing %q after provisioning:\n%s", path, want, entries[path])
		}
	}
	// A separately developed frontend's local dev origin must be allowed in
	// application-dev.yml's CORS config, alongside the Ionic defaults.
	for path, want := range map[string]string{
		"UmlArchitect/src/main/resources/config/application-dev.yml": "http://localhost:5173",
	} {
		if !strings.Contains(string(entries[path]), want) {
			t.Errorf("%s missing %q after provisioning:\n%s", path, want, entries[path])
		}
	}
	for _, no := range []string{"node_modules", ".git"} {
		for name := range entries {
			if strings.Contains(name, no) {
				t.Errorf("zip must not contain excluded path %q", name)
			}
		}
	}

	var m Manifest
	if err := json.Unmarshal(entries["manifest.json"], &m); err != nil {
		t.Fatalf("manifest not json: %v", err)
	}
	if m.Generator.Name != "generator-jhipster" || m.Generator.Version != "9.4.0" {
		t.Errorf("generator meta = %+v", m.Generator)
	}
	if m.Generator.Invocation != wantCmd {
		t.Errorf("invocation = %q, want %q", m.Generator.Invocation, wantCmd)
	}
	if m.BaseName != "UmlArchitect" || m.PackageName != "com.umlarchitect" {
		t.Errorf("identity = %s/%s", m.BaseName, m.PackageName)
	}
	if m.DatabaseName != "uml-architect" || m.SQLFileName != "database/uml-architect.sql" {
		t.Errorf("database provisioning = %s/%s", m.DatabaseName, m.SQLFileName)
	}
	if !reflect.DeepEqual(m.RunCommands, []string{"cd UmlArchitect", "docker compose up -d", "./mvnw", "./mvnw -Pprod"}) {
		t.Errorf("runCommands = %v", m.RunCommands)
	}
	if !reflect.DeepEqual(m.Entities, []string{"Alpha"}) {
		t.Errorf("entities = %v", m.Entities)
	}
	wantFiles := []string{
		".yo-rc.json",
		"compose.yml",
		"database/uml-architect.sql",
		"model.jdl",
		"pom.xml",
		"src/main/java/com/umlarchitect/Alpha.java",
		"src/main/resources/config/application-dev.yml",
		"src/main/resources/config/application-prod.yml",
		"src/main/resources/config/application-secret-samples.yml",
		"src/main/resources/config/application.yml",
	}
	if !reflect.DeepEqual(m.Files, wantFiles) {
		t.Errorf("files = %v, want %v", m.Files, wantFiles)
	}
	if m.FileCount != len(m.Files) {
		t.Errorf("fileCount = %d, files = %d", m.FileCount, len(m.Files))
	}
	if m.GeneratedAt == "" {
		t.Errorf("generatedAt missing")
	}
}

func TestRunCommandsPerBuildTool(t *testing.T) {
	if got := runCommands("MyApp", "gradle"); !reflect.DeepEqual(got, []string{"cd MyApp", "docker compose up -d", "./gradlew", "./gradlew -Pprod"}) {
		t.Errorf("runCommands(MyApp, gradle) = %v", got)
	}
	if got := runCommands("MyApp", "maven"); !reflect.DeepEqual(got, []string{"cd MyApp", "docker compose up -d", "./mvnw", "./mvnw -Pprod"}) {
		t.Errorf("runCommands(MyApp, maven) = %v", got)
	}
	if got := runCommands("MyApp", "anything-else"); !reflect.DeepEqual(got, []string{"cd MyApp", "docker compose up -d", "./mvnw", "./mvnw -Pprod"}) {
		t.Errorf("runCommands(MyApp, unknown) = %v", got)
	}
}

func TestGenerateProvisioningFailureSurfacesError(t *testing.T) {
	// The runner succeeds but the scaffold lacks src/main/resources/config, so
	// provisioning must fail loudly instead of shipping an artifact without
	// the database wiring.
	runner := &fakeRunner{generatedFiles: []string{"pom.xml", ".yo-rc.json"}}
	g := &Generator{Runner: runner, Version: Version, Timeout: 0}
	_, err := g.Generate(context.Background(), alphaDoc(), jdlgen.DefaultOptions())
	if err == nil {
		t.Fatal("expected provisioning error when config files are missing")
	}
	if !strings.Contains(err.Error(), "application") || !strings.Contains(err.Error(), ".yml") {
		t.Errorf("error must name the missing config file, got: %v", err)
	}
}

func TestGenerateInvalidOptionsSkipsRunner(t *testing.T) {
	runner := &fakeRunner{}
	g := &Generator{Runner: runner, Version: Version, Timeout: 0}
	_, err := g.Generate(context.Background(), alphaDoc(), jdlgen.Options{BaseName: "Bad App"})
	if err == nil || !strings.Contains(err.Error(), "invalid JHipster application options") {
		t.Fatalf("expected options error, got %v", err)
	}
	if len(runner.calls) != 0 {
		t.Errorf("runner must not be invoked on invalid options, calls: %v", runner.calls)
	}
}

func TestGenerateSanitizesRunnerFailure(t *testing.T) {
	runner := &fakeRunner{err: os.ErrPermission}
	g := &Generator{Runner: runner, Version: Version, Timeout: 0}
	_, err := g.Generate(context.Background(), alphaDoc(), jdlgen.DefaultOptions())
	if err == nil {
		t.Fatal("expected generation error")
	}
	if !strings.Contains(err.Error(), "JHipster generation failed") {
		t.Errorf("error missing runner context: %v", err)
	}
	if !strings.Contains(err.Error(), "<workdir>") {
		t.Errorf("error must use the <workdir> placeholder: %v", err)
	}
	if strings.Contains(err.Error(), "jhipster-artifact-") {
		t.Errorf("error must not leak the ephemeral workdir: %v", err)
	}
}

func TestGenerateClassifiesMissingPrerequisite(t *testing.T) {
	runner := &fakeRunner{err: exec.ErrNotFound}
	g := &Generator{Runner: runner, Version: Version, Timeout: 0}
	_, err := g.Generate(context.Background(), alphaDoc(), jdlgen.DefaultOptions())
	if err == nil {
		t.Fatal("expected generation error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "prerequisite missing") || !strings.Contains(msg, "pnpm") {
		t.Errorf("error must name the missing prerequisite (pnpm): %v", err)
	}
}

func TestGenerateClassifiesJDLParseRejection(t *testing.T) {
	const out = "ERROR! ERROR! MismatchedTokenException: Found an invalid token '}', at line: 27 and column: 1.\n" +
		"    at /home/test-user/.cache/pnpm/dlx/abcdef/node_modules/chevrotain/src/parse/parser/parser.js:123:45\n"
	runner := &fakeRunner{err: errors.New("exit status 1"), out: out}
	g := &Generator{Runner: runner, Version: Version, Timeout: 0}
	_, err := g.Generate(context.Background(), alphaDoc(), jdlgen.DefaultOptions())
	if err == nil {
		t.Fatal("expected generation error")
	}
	msg := err.Error()
	for _, want := range []string{"rejected the generated JDL", "MismatchedTokenException", "line: 27"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error must contain %q, got:\n%s", want, msg)
		}
	}
	if !strings.Contains(msg, "<abs-path>") {
		t.Errorf("error must scrub absolute paths inside the log: %v", err)
	}
	if strings.Contains(msg, "/home/test-user") {
		t.Errorf("error must not leak absolute home paths: %v", err)
	}
}

func TestGenerateBoundsLogHeadAndTail(t *testing.T) {
	big := "BEGIN:" + strings.Repeat("x", 3000) + ":END"
	runner := &fakeRunner{err: errors.New("boom"), out: big}
	g := &Generator{Runner: runner, Version: Version, Timeout: 0}
	_, err := g.Generate(context.Background(), alphaDoc(), jdlgen.DefaultOptions())
	if err == nil {
		t.Fatal("expected generation error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "BEGIN:") || !strings.Contains(msg, ":END") {
		t.Errorf("sanitized log must keep the head and the tail, got:\n%s", msg)
	}
	if !strings.Contains(msg, "bytes omitted") {
		t.Errorf("oversized log must carry an omission marker, got:\n%s", msg)
	}
}

func TestGenerateClassifiesTimeout(t *testing.T) {
	runner := &fakeRunner{err: errors.New("signal: killed")}
	g := &Generator{Runner: runner, Version: Version, Timeout: time.Minute}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err := g.Generate(ctx, alphaDoc(), jdlgen.DefaultOptions())
	if err == nil {
		t.Fatal("expected generation error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "timed out after") {
		t.Errorf("expired context must be classified as a timeout, got:\n%s", msg)
	}
}

func TestGenerateClassifiesPnpmError(t *testing.T) {
	runner := &fakeRunner{err: errors.New("boom"), out: "ERR_PNPM_UNSUPPORTED_ENGINE Unsupported environment (bad pnpm version)\nmore output"}
	g := &Generator{Runner: runner, Version: Version, Timeout: 0}
	_, err := g.Generate(context.Background(), alphaDoc(), jdlgen.DefaultOptions())
	if err == nil {
		t.Fatal("expected generation error")
	}
	if !strings.Contains(err.Error(), "pnpm reported an error") || !strings.Contains(err.Error(), "ERR_PNPM_UNSUPPORTED_ENGINE") {
		t.Errorf("pnpm failure must be classified as such, got:\n%s", err.Error())
	}
}
