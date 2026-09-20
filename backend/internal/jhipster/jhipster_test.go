package jhipster

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
)

// fakeRunner simulates the pinned generator: it records the invoked command
// and materializes the files it "generated" into the workdir.
type fakeRunner struct {
	calls          []string
	generatedFiles []string
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
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
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
	runner := &fakeRunner{generatedFiles: []string{
		"pom.xml",
		".yo-rc.json",
		"src/main/java/com/umlarchitect/Alpha.java",
		"node_modules/pkg/index.js", // must be excluded
		".git/config",               // must be excluded
	}}
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
	} {
		if _, ok := entries[want]; !ok {
			t.Errorf("zip missing entry %q (have %v)", want, keys(entries))
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
	if !reflect.DeepEqual(m.Entities, []string{"Alpha"}) {
		t.Errorf("entities = %v", m.Entities)
	}
	wantFiles := []string{
		".yo-rc.json",
		"model.jdl",
		"pom.xml",
		"src/main/java/com/umlarchitect/Alpha.java",
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
