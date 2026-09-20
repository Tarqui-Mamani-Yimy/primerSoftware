// Package jhipster runs the pinned JHipster generator (generator-jhipster)
// via `pnpm dlx` inside an isolated temp directory and packages the generated
// backend as a downloadable ZIP whose manifest lists exactly the files the
// generator produced.
//
// Rules this package enforces:
//   - No global install: the pinned generator is executed through pnpm's
//     ephemeral `dlx` store (authorized installs in this project are pnpm-only).
//   - No shell: commands run through os/exec with fixed arguments.
//   - No repository writes: every request generates in its own os.MkdirTemp
//     scratch space that is removed when the request completes.
//   - No invented content: the JDL comes from jdlgen, and the manifest lists
//     only files physically present after generation.
package jhipster

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
)

// Version is the pinned generator-jhipster release. It is a deliberate pin:
// generator output moves between minors, so the artifact contract is tied to
// an exact version. Bump deliberately and re-verify the generation flow.
const Version = "9.4.0"

// CommandRunner executes a named command with fixed arguments inside a
// workdir without a shell. Combined output is returned for diagnostics.
// Tests inject a fake runner so the generation flow never needs pnpm.
type CommandRunner interface {
	Run(ctx context.Context, dir, name string, args ...string) (string, error)
}

// ExecRunner is the production CommandRunner backed by os/exec.
type ExecRunner struct{}

// Run uses exec.CommandContext so a caller-controlled context can bound the
// generation budget (the first run also pulls the pinned generator into the
// pnpm store, which is the slowest path).
func (ExecRunner) Run(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Result is the downloadable artifact the generator produces.
type Result struct {
	FileName string
	Content  []byte
}

// Generator produces Result ZIPs from a UML document. Runner may be swapped
// for tests; the zero Generator is not usable — use NewGenerator.
type Generator struct {
	Runner  CommandRunner
	Version string
	Timeout time.Duration // generation budget; 0 disables the extra bound
}

// NewGenerator returns a Generator wired to the pinned version, pnpm via
// PATH, and a 10-minute budget covering the first pnpm store download.
func NewGenerator() *Generator {
	return &Generator{Runner: ExecRunner{}, Version: Version, Timeout: 10 * time.Minute}
}

// Generate validates the application options, exports the JDL, runs the
// pinned JHipster generator in an isolated temp dir, and packages the files
// that exist after generation into a ZIP with a provenance manifest.
func (g *Generator) Generate(ctx context.Context, doc domain.DiagramDocument, o jdlgen.Options) (Result, error) {
	jdl, rep, err := jdlgen.ExportArtifact(doc, o)
	if err != nil {
		return Result{}, err
	}
	genVersion := g.Version
	if genVersion == "" {
		genVersion = Version
	}
	if g.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, g.Timeout)
		defer cancel()
	}

	workDir, err := os.MkdirTemp("", "jhipster-artifact-*")
	if err != nil {
		return Result{}, fmt.Errorf("create workdir: %w", err)
	}
	defer os.RemoveAll(workDir)

	if err := os.WriteFile(filepath.Join(workDir, "model.jdl"), []byte(jdl), 0o644); err != nil {
		return Result{}, fmt.Errorf("write model.jdl: %w", err)
	}

	out, err := g.Runner.Run(ctx, workDir, "pnpm", "dlx", "generator-jhipster@"+genVersion, "jdl", "model.jdl", "--force", "--skip-install")
	if err != nil {
		return Result{}, fmt.Errorf("JHipster generation failed: %s", sanitizeOutput(out, workDir))
	}

	files, err := listGenerated(workDir)
	if err != nil {
		return Result{}, fmt.Errorf("list generated files: %w", err)
	}

	m := Manifest{
		Generator:     ManifestGenerator{Name: "generator-jhipster", Version: genVersion, Invocation: invocation(genVersion)},
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		BaseName:      o.BaseName,
		PackageName:   o.PackageName,
		Entities:      rep.Entities,
		Relationships: rep.Relationships,
		Warnings:      rep.Warnings,
		Skipped:       skippedFromReport(rep.Dropped),
		Files:         files,
		FileCount:     len(files),
	}

	content, err := buildZip(workDir, o.BaseName, m)
	if err != nil {
		return Result{}, fmt.Errorf("package zip: %w", err)
	}
	return Result{FileName: o.BaseName + "-jhipster-backend.zip", Content: content}, nil
}

func invocation(version string) string {
	// --ignore-scripts: pnpm 12 blocks unapproved build scripts by default and
	// ERR_PNPM_IGNORED_BUILDS aborts dlx on unrs-resolver's optional script.
	// JHipster 9.4.0 runs fine without it (unrs-resolver ships prebuilt binaries).
	return fmt.Sprintf("pnpm dlx --ignore-scripts generator-jhipster@%s jdl model.jdl --force --skip-install", version)
}

// sanitizeOutput strips the ephemeral absolute workdir from generator output
// (it contains the local temp path) and bounds the tail so error envelopes
// stay small enough to embed in a JSON message.
func sanitizeOutput(out, workDir string) string {
	out = strings.ReplaceAll(out, workDir, "<workdir>")
	const max = 1200
	if len(out) > max {
		out = "…" + out[len(out)-max:]
	}
	return strings.TrimSpace(out)
}

// excludedDirs never appear in the manifest or the ZIP: dependency install
// output, VCS metadata, and build products are not "generated files".
var excludedDirs = map[string]struct{}{
	"node_modules": {},
	".git":         {},
	"target":       {},
	".gradle":      {},
	".idea":        {},
}

// listGenerated walks the generated project and returns sorted relative
// paths of every file present, skipping excluded directories and symlinks.
func listGenerated(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if d.IsDir() {
			if _, skip := excludedDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// Manifest is the artifact's machine-readable provenance record. Files lists
// exactly the files physically present after generation.
type Manifest struct {
	Generator     ManifestGenerator `json:"generator"`
	GeneratedAt   string            `json:"generatedAt"`
	BaseName      string            `json:"baseName"`
	PackageName   string            `json:"packageName"`
	Entities      []string          `json:"entities"`
	Relationships []string          `json:"relationships"`
	Warnings      []string          `json:"warnings"`
	Skipped       []Skipped         `json:"skipped"`
	Files         []string          `json:"files"`
	FileCount     int               `json:"fileCount"`
}

// ManifestGenerator identifies the exact generator invocation that produced
// the artifact.
type ManifestGenerator struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Invocation string `json:"invocation"`
}

// Skipped records one UML construct JDL could not express, mirroring
// jdlgen's report so the manifest stays machine-readable.
type Skipped struct {
	Kind     string `json:"kind"`
	Location string `json:"location"`
	Detail   string `json:"detail"`
}

func skippedFromReport(dropped []jdlgen.Dropped) []Skipped {
	if len(dropped) == 0 {
		return []Skipped{}
	}
	out := make([]Skipped, 0, len(dropped))
	for _, d := range dropped {
		out = append(out, Skipped{Kind: d.Kind, Location: d.Location, Detail: d.Detail})
	}
	return out
}

// buildZip packages the generated project under a baseName/ root folder (so
// unzipping yields a single project directory) with manifest.json first at
// the ZIP root.
func buildZip(root, baseName string, m Manifest) ([]byte, error) {
	manifestRaw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	manifestRaw = append(manifestRaw, '\n')

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if fw, err := zw.Create("manifest.json"); err != nil {
		_ = zw.Close()
		return nil, fmt.Errorf("create manifest entry: %w", err)
	} else if _, err := fw.Write(manifestRaw); err != nil {
		_ = zw.Close()
		return nil, fmt.Errorf("write manifest entry: %w", err)
	}
	for _, rel := range m.Files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("read generated file %s: %w", rel, err)
		}
		entry := filepath.ToSlash(filepath.Join(baseName, filepath.FromSlash(rel)))
		fw, err := zw.Create(entry)
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("create zip entry %s: %w", entry, err)
		}
		if _, err := fw.Write(data); err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("write zip entry %s: %w", entry, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("finalize zip: %w", err)
	}
	return buf.Bytes(), nil
}
