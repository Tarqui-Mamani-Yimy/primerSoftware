// Command jdl-bundle prepares a downloadable JHipster bundle from a UML
// diagram JSON file (GEN-02 preparation step).
//
// It is a standalone entry point: nothing in the Go server imports it, so the
// runtime backend is unchanged. The bundle holds model.jdl, report.json, a
// pinned .yo-rc.json seed (monolith + PostgreSQL), and generate.sh with the
// exact ordered commands to run on a machine where JHipster is installed.
//
// Usage:
//
//	go run ./cmd/jdl-bundle -diagram diagram.json -app shop -out shop-jhipster
//
// The diagram file may hold either a DiagramDocument or a DiagramVersion
// (which wraps the document under the "document" key).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
)

// yoRcSeed is the pinned JHipster application seed: monolith with PostgreSQL
// for both dev and prod. __BASE__ is the sanitized baseName, __PKG__ the Java
// package. Timestamps and secrets are intentionally absent; JHipster fills
// generated values at scaffold time.
const yoRcSeed = `{
  "generator-jhipster": {
    "applicationType": "monolith",
    "authenticationType": "jwt",
    "baseName": "__BASE__",
    "buildTool": "maven",
    "cacheProvider": "ehcache",
    "clientFramework": "angular",
    "clientPackageManager": "npm",
    "databaseType": "sql",
    "devDatabaseType": "postgresql",
    "enableHibernateCache": true,
    "enableTranslation": false,
    "jhiPrefix": "jhi",
    "languages": [
      "en"
    ],
    "messageBroker": false,
    "nativeLanguage": "en",
    "packageName": "__PKG__",
    "prodDatabaseType": "postgresql",
    "searchEngine": false,
    "serverPort": "8080",
    "serviceDiscoveryType": "no",
    "skipClient": false,
    "skipUserManagement": false,
    "testFrameworks": [],
    "websocket": false,
    "withAdminUi": true
  }
}
`

// generateScript holds the exact ordered commands for the machine where
// JHipster exists. <pinned> must be replaced with (or exported via
// PINNED_JHIPSTER as) the exact generator version from
// `npm view generator-jhipster version`.
const generateScript = `#!/usr/bin/env bash
# GEN-02 bundle for __BASE__. Ordered commands — run on a machine with
# Node.js LTS, network access, and Java 17+ installed. JHipster was NOT
# installed where this bundle was prepared; step 1 installs a pinned generator.
set -euo pipefail

# 1. Install the pinned JHipster generator.
PINNED_JHIPSTER="${PINNED_JHIPSTER:-<pinned>}"
npm install -g "generator-jhipster@${PINNED_JHIPSTER}"

# 2. Verify the generator.
jhipster --version

# 3. Scaffold the monolith (answers seeded from .yo-rc.json) and import the model.
jhipster jdl model.jdl --force

# 4. Build with the production profile (PostgreSQL + Liquibase changelogs).
./mvnw -Pprod verify
`

func main() {
	diagramPath := flag.String("diagram", "", "path to diagram JSON (DiagramDocument or DiagramVersion)")
	app := flag.String("app", "", "application name (becomes the JHipster baseName)")
	out := flag.String("out", "", "output directory (default: <app>-jhipster)")
	force := flag.Bool("force", false, "overwrite a non-empty output directory")
	flag.Parse()

	if err := run(*diagramPath, *app, *out, *force); err != nil {
		fmt.Fprintf(os.Stderr, "jdl-bundle: %v\n", err)
		os.Exit(1)
	}
}

func run(diagramPath, app, out string, force bool) error {
	if strings.TrimSpace(diagramPath) == "" {
		return fmt.Errorf("missing -diagram path")
	}
	base, pkg, err := sanitizeAppName(app)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) == "" {
		out = base + "-jhipster"
	}

	doc, err := loadDocument(diagramPath)
	if err != nil {
		return err
	}
	jdl, rep := jdlgen.Export(doc)
	reportRaw, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	reportRaw = append(reportRaw, '\n')

	if err := prepareDir(out, force); err != nil {
		return err
	}
	files := map[string][]byte{
		"model.jdl":   []byte(jdl),
		"report.json": reportRaw,
		".yo-rc.json": []byte(strings.ReplaceAll(strings.ReplaceAll(yoRcSeed, "__BASE__", base), "__PKG__", pkg)),
		"generate.sh": []byte(strings.ReplaceAll(generateScript, "__BASE__", base)),
	}
	for name, data := range files {
		mode := os.FileMode(0o644)
		if name == "generate.sh" {
			mode = 0o755
		}
		if err := os.WriteFile(filepath.Join(out, name), data, mode); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	fmt.Printf("bundle ready in %s/\n", out)
	fmt.Printf("  model.jdl: %d entities, %d relationships\n", len(rep.Entities), len(rep.Relationships))
	fmt.Printf("  report.json: %d dropped constructs, %d warnings (review before generating)\n",
		len(rep.Dropped), len(rep.Warnings))
	fmt.Printf("next: cd %s && ./generate.sh  (needs Node LTS, Java 17+, network; set PINNED_JHIPSTER)\n", out)
	return nil
}

// loadDocument accepts either a DiagramDocument or a DiagramVersion payload.
func loadDocument(path string) (domain.DiagramDocument, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return domain.DiagramDocument{}, fmt.Errorf("read diagram: %w", err)
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return domain.DiagramDocument{}, fmt.Errorf("parse diagram JSON: %w", err)
	}
	if docRaw, ok := probe["document"]; ok {
		var version domain.DiagramVersion
		if err := json.Unmarshal(docRaw, &version.Document); err != nil {
			return domain.DiagramDocument{}, fmt.Errorf("parse DiagramVersion.document: %w", err)
		}
		return version.Document, nil
	}
	var doc domain.DiagramDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return domain.DiagramDocument{}, fmt.Errorf("parse DiagramDocument: %w", err)
	}
	return doc, nil
}

// sanitizeAppName derives a JHipster baseName (lowerCamel, alphanumeric) and
// its Java package from a free-form application name.
func sanitizeAppName(app string) (base, pkg string, err error) {
	var b strings.Builder
	for _, r := range app {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	base = b.String()
	if base == "" {
		return "", "", fmt.Errorf("app name %q has no alphanumeric characters", app)
	}
	if base[0] >= '0' && base[0] <= '9' {
		base = "app" + base
	}
	base = string(unicode.ToLower(rune(base[0]))) + base[1:]
	return base, "com.example." + strings.ToLower(base), nil
}

// prepareDir creates out, refusing a non-empty directory unless force is set.
func prepareDir(out string, force bool) error {
	entries, err := os.ReadDir(out)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(out, 0o755)
		}
		return fmt.Errorf("inspect output dir: %w", err)
	}
	if len(entries) > 0 && !force {
		return fmt.Errorf("output directory %q is not empty (use -force to overwrite)", out)
	}
	return os.MkdirAll(out, 0o755)
}
