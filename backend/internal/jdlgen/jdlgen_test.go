package jdlgen_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
)

// updateGoldens rewrites testdata goldens: run
// `go test ./internal/jdlgen/ -update`, inspect the diff, then rerun without
// -update. Never commit a blind -update.
var updateGoldens = flag.Bool("update", false, "rewrite golden files under testdata/")

func strp(s string) *string { return &s }

func intp(i int) *int { return &i }

func assertGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *updateGoldens {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("rewrite golden %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update once, inspect, rerun)", path, err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

func reportJSON(t *testing.T, rep jdlgen.Report) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	return append(raw, '\n')
}

func TestSanitizeIdentifier(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain name kept", input: "Order", want: "Order"},
		{name: "empty becomes Unnamed", input: "", want: "Unnamed"},
		{name: "blank becomes Unnamed", input: "   ", want: "Unnamed"},
		{name: "reserved word gains suffix", input: "class", want: "class_"},
		{name: "reserved word int gains suffix", input: "int", want: "int_"},
		{name: "literal true gains suffix", input: "true", want: "true_"},
		{name: "uppercase Class is legal Java", input: "Class", want: "Class"},
		{name: "leading digit gains prefix", input: "123abc", want: "_123abc"},
		{name: "hyphen becomes underscore", input: "my-field", want: "my_field"},
		{name: "space becomes underscore", input: "Customer Name", want: "Customer_Name"},
		{name: "dollar becomes underscore", input: "order$", want: "order_"},
		{name: "lone underscore is reserved", input: "_", want: "Unnamed"},
		{name: "punctuation only becomes Unnamed", input: "!!!", want: "Unnamed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := jdlgen.SanitizeIdentifier(tc.input); got != tc.want {
				t.Errorf("SanitizeIdentifier(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestEntityAndFieldNameCasing(t *testing.T) {
	entityCases := []struct{ input, want string }{
		{"order", "Order"},
		{"class", "Class"},
		{"customer", "Customer"},
	}
	for _, tc := range entityCases {
		t.Run("entity "+tc.input, func(t *testing.T) {
			if got := jdlgen.EntityName(tc.input); got != tc.want {
				t.Errorf("EntityName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
	fieldCases := []struct{ input, want string }{
		{"Order", "order"},
		{"Class_", "class2"},
		{"Total", "total"},
	}
	for _, tc := range fieldCases {
		t.Run("field "+tc.input, func(t *testing.T) {
			if got := jdlgen.FieldName(tc.input); got != tc.want {
				t.Errorf("FieldName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// jdlReservedNamesDriveTheParser covers the root-cause fix: any field or
// entity name that collides exactly with a JDL lexer keyword breaks the
// pinned generator's parser (MismatchedTokenException), so such names must
// gain a deterministic "2" suffix (underscores are rejected by the JDL
// validator) and a warning.
func jdlReservedDoc() domain.DiagramDocument {
	return domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "reserved",
		Classes: []domain.UmlClass{
			{ID: "r1", Name: "Order", Attributes: []domain.Attribute{
				{ID: "a1", Name: "required", Type: "String"},
				{ID: "a2", Name: "unique", Type: "String"},
				{ID: "a3", Name: "baseName", Type: "String"},
				{ID: "a4", Name: "readOnly", Type: "Boolean"},
				{ID: "a5", Name: "code", Type: "String"},
			}},
			{ID: "r2", Name: "OneToOne"},
		},
	}
}

func TestFieldNameJDLReservedWords(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "validation required", input: "required", want: "required2"},
		{name: "validation unique", input: "unique", want: "unique2"},
		{name: "validation pattern", input: "pattern", want: "pattern2"},
		{name: "validation min", input: "min", want: "min2"},
		{name: "config baseName", input: "BaseName", want: "baseName2"},
		{name: "option readOnly", input: "ReadOnly", want: "readOnly2"},
		{name: "java reserved only", input: "class", want: "class2"},
		{name: "plain name kept", input: "code", want: "code"},
		{name: "input case folds to reserved", input: "Required", want: "required2"},
		{name: "superstring is safe", input: "requiredBy", want: "requiredBy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := jdlgen.FieldName(tc.input); got != tc.want {
				t.Errorf("FieldName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestEntityNameJDLReservedWords(t *testing.T) {
	cases := []struct{ input, want string }{
		{"OneToOne", "OneToOne2"},
		{"ManyToMany", "ManyToMany2"},
		{"Entity", "Entity"},
		{"Application", "Application"},
		{"Order", "Order"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			if got := jdlgen.EntityName(tc.input); got != tc.want {
				t.Errorf("EntityName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestExportRenamesJDLReservedNames(t *testing.T) {
	got, rep := jdlgen.Export(jdlReservedDoc())
	for _, want := range []string{
		"  required2 String",
		"  unique2 String",
		"  baseName2 String",
		"  readOnly2 Boolean",
		"entity OneToOne2 {",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("JDL must contain %q, got:\n%s", want, got)
		}
	}
	joined := strings.Join(rep.Warnings, "\n")
	for _, want := range []string{
		`attribute "required" renamed to field "required2" (JDL reserved word`,
		`attribute "unique" renamed to field "unique2" (JDL reserved word`,
		`attribute "baseName" renamed to field "baseName2" (JDL reserved word`,
		`attribute "readOnly" renamed to field "readOnly2" (JDL reserved word`,
		`class "OneToOne" renamed to entity "OneToOne2" (JDL reserved word`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("warnings must contain %q, got:\n%s", want, joined)
		}
	}
}

func TestEnsureUnique(t *testing.T) {
	used := map[string]struct{}{}
	if got := jdlgen.EnsureUnique("Order", used); got != "Order" {
		t.Fatalf("first use = %q, want %q", got, "Order")
	}
	if got := jdlgen.EnsureUnique("Order", used); got != "Order2" {
		t.Fatalf("second use = %q, want %q", got, "Order2")
	}
	if got := jdlgen.EnsureUnique("Order", used); got != "Order3" {
		t.Fatalf("third use = %q, want %q", got, "Order3")
	}
	if got := jdlgen.EnsureUnique("Customer", used); got != "Customer" {
		t.Fatalf("fresh base = %q, want %q", got, "Customer")
	}
}

// TestBuildModelRenamesJHipsterBuiltInEntities covers the collision between a
// UML class and one of JHipster's own internal entities (User, Authority):
// generator-jhipster 9.4.0 merges a same-named JDL entity into its built-in
// one (fields other than id and Src-side relationships are silently
// disregarded, generators/base-application/internal/utils.js:38-70) instead
// of creating it, so the class must be renamed before it ever reaches the
// generator.
func TestBuildModelRenamesJHipsterBuiltInEntities(t *testing.T) {
	doc := domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Shop",
		Classes: []domain.UmlClass{
			{ID: "c1", Name: "User"},
			{ID: "c2", Name: "Order"},
		},
		Relationships: []domain.Relationship{
			{ID: "r1", SourceID: "c2", TargetID: "c1", Type: "association",
				SourceMultiplicity: strp("*"), TargetMultiplicity: strp("1")},
		},
	}
	jdl, rep := jdlgen.Export(doc)

	wantEntities := []string{"AppUser", "Order"}
	if len(rep.Entities) != len(wantEntities) {
		t.Fatalf("entities = %v, want %v", rep.Entities, wantEntities)
	}
	for i, want := range wantEntities {
		if rep.Entities[i] != want {
			t.Errorf("entities = %v, want %v", rep.Entities, wantEntities)
			break
		}
	}
	if !strings.Contains(jdl, "entity AppUser {") {
		t.Errorf("JDL missing renamed entity AppUser:\n%s", jdl)
	}
	if strings.Contains(jdl, "entity User {") {
		t.Errorf("JDL must not declare the colliding entity User:\n%s", jdl)
	}
	// Relationships must resolve to the renamed entity automatically.
	if len(rep.Relationships) != 1 || rep.Relationships[0] != "Order{appUser required} to AppUser{orders}" {
		t.Errorf("relationship = %v, want [Order{appUser required} to AppUser{orders}]", rep.Relationships)
	}
	joined := strings.Join(rep.Warnings, "\n")
	if !strings.Contains(joined, `class "User" renamed to entity "AppUser" (collides with JHipster built-in entity)`) {
		t.Errorf("warnings must explain the built-in collision, got:\n%s", joined)
	}
}

// TestBuildModelRenamesAuthorityBuiltIn covers the same collision for
// JHipster's built-in Authority entity.
func TestBuildModelRenamesAuthorityBuiltIn(t *testing.T) {
	doc := domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Shop",
		Classes: []domain.UmlClass{
			{ID: "c1", Name: "Authority"},
		},
	}
	_, rep := jdlgen.Export(doc)
	if len(rep.Entities) != 1 || rep.Entities[0] != "AppAuthority" {
		t.Fatalf("entities = %v, want [AppAuthority]", rep.Entities)
	}
	joined := strings.Join(rep.Warnings, "\n")
	if !strings.Contains(joined, `class "Authority" renamed to entity "AppAuthority" (collides with JHipster built-in entity)`) {
		t.Errorf("warnings must explain the built-in collision, got:\n%s", joined)
	}
}

// TestBuildModelBuiltInEntityCollisionDeduplicates covers a UML model that
// already has a class named "AppUser" alongside one named "User": the
// renamed "User" must not silently overwrite "AppUser" (EnsureUnique must
// still run after the App-prefix rename).
func TestBuildModelBuiltInEntityCollisionDeduplicates(t *testing.T) {
	doc := domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Shop",
		Classes: []domain.UmlClass{
			{ID: "c1", Name: "AppUser"},
			{ID: "c2", Name: "User"},
		},
	}
	_, rep := jdlgen.Export(doc)
	wantEntities := []string{"AppUser", "AppUser2"}
	if len(rep.Entities) != len(wantEntities) {
		t.Fatalf("entities = %v, want %v", rep.Entities, wantEntities)
	}
	for i, want := range wantEntities {
		if rep.Entities[i] != want {
			t.Errorf("entities = %v, want %v", rep.Entities, wantEntities)
			break
		}
	}
	joined := strings.Join(rep.Warnings, "\n")
	if !strings.Contains(joined, `class "User" renamed to entity "AppUser2" (collides with JHipster built-in entity)`) {
		t.Errorf("warnings must explain the deduplicated built-in collision, got:\n%s", joined)
	}
}

func cardinalityDoc(srcMult, dstMult *string) domain.DiagramDocument {
	return domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "cards",
		Classes: []domain.UmlClass{
			{ID: "a", Name: "Alpha"},
			{ID: "b", Name: "Beta"},
		},
		Relationships: []domain.Relationship{
			{ID: "r", SourceID: "a", TargetID: "b", Type: "association",
				SourceMultiplicity: srcMult, TargetMultiplicity: dstMult},
		},
	}
}

func TestExportCardinalities(t *testing.T) {
	cases := []struct {
		name string
		src  *string
		dst  *string
		want string
	}{
		{name: "missing means one to one", src: nil, dst: nil, want: "Alpha{beta} to Beta{alpha}"},
		{name: "one to many", src: strp("1"), dst: strp("*"), want: "Alpha{betas} to Beta{alpha required}"},
		{name: "many to one", src: strp("*"), dst: strp("1"), want: "Alpha{beta required} to Beta{alphas}"},
		{name: "many to many", src: strp("*"), dst: strp("*"), want: "Alpha{betas} to Beta{alphas}"},
		{name: "range many", src: strp("0..1"), dst: strp("1..*"), want: "Alpha{betas} to Beta{alpha}"},
		{name: "numeric many", src: strp("2"), dst: strp("1"), want: "Alpha{beta required} to Beta{alphas}"},
		{name: "one to one ranges", src: strp("1"), dst: strp("1..1"), want: "Alpha{beta required} to Beta{alpha}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, rep := jdlgen.Export(cardinalityDoc(tc.src, tc.dst))
			if len(rep.Relationships) != 1 {
				t.Fatalf("expected 1 relationship, got %v", rep.Relationships)
			}
			if rep.Relationships[0] != tc.want {
				t.Errorf("relationship = %q, want %q", rep.Relationships[0], tc.want)
			}
		})
	}
}

func selfRefDoc(srcMult, dstMult *string) domain.DiagramDocument {
	return domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "self",
		Classes: []domain.UmlClass{
			{ID: "a", Name: "Alpha"},
		},
		Relationships: []domain.Relationship{
			{ID: "r", SourceID: "a", TargetID: "a", Type: "association",
				SourceMultiplicity: srcMult, TargetMultiplicity: dstMult},
		},
	}
}

// TestBuildModelRequiredByKind covers the lower-bound-drives-mandatory-FK
// rule end to end: isRequiredEnd (tested via multiplicity values across the
// full lower-bound range) combined with FK ownership per relationship kind,
// plus the self-reference override that never forces a required self-FK.
func TestBuildModelRequiredByKind(t *testing.T) {
	cases := []struct {
		name         string
		selfRef      bool
		src          *string
		dst          *string
		wantKind     string
		wantRequired bool
		wantRendered string
		wantWarning  string
	}{
		{name: "many to one required (lower bound 1)", src: strp("*"), dst: strp("1"),
			wantKind: "ManyToOne", wantRequired: true, wantRendered: "Alpha{beta required} to Beta{alphas}"},
		{name: "many to optional one (lower bound 0)", src: strp("*"), dst: strp("0..1"),
			wantKind: "ManyToOne", wantRequired: false, wantRendered: "Alpha{beta} to Beta{alphas}"},
		{name: "one to many required (lower bound 1)", src: strp("1"), dst: strp("*"),
			wantKind: "OneToMany", wantRequired: true, wantRendered: "Alpha{betas} to Beta{alpha required}"},
		{name: "optional one to many (lower bound 0)", src: strp("0..1"), dst: strp("*"),
			wantKind: "OneToMany", wantRequired: false, wantRendered: "Alpha{betas} to Beta{alpha}"},
		{name: "one to one required (lower bound 1)", src: strp("1"), dst: strp("1"),
			wantKind: "OneToOne", wantRequired: true, wantRendered: "Alpha{beta required} to Beta{alpha}"},
		{name: "many to many never required", src: strp("*"), dst: strp("*"),
			wantKind: "ManyToMany", wantRequired: false, wantRendered: "Alpha{betas} to Beta{alphas}"},
		{name: "nil multiplicities not required", src: nil, dst: nil,
			wantKind: "OneToOne", wantRequired: false, wantRendered: "Alpha{beta} to Beta{alpha}"},
		{name: "self-reference kept optional", selfRef: true, src: strp("*"), dst: strp("1"),
			wantKind: "ManyToOne", wantRequired: false, wantRendered: "Alpha{alpha} to Alpha{alphas}",
			wantWarning: `relationship r: self-reference "Alpha" kept optional (a required self-FK makes the first row impossible to insert)`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var doc domain.DiagramDocument
			if tc.selfRef {
				doc = selfRefDoc(tc.src, tc.dst)
			} else {
				doc = cardinalityDoc(tc.src, tc.dst)
			}
			m, rep := jdlgen.BuildModel(doc)
			if len(m.Relationships) != 1 {
				t.Fatalf("expected 1 relationship, got %+v", m.Relationships)
			}
			r := m.Relationships[0]
			if r.Kind != tc.wantKind {
				t.Errorf("Kind = %q, want %q", r.Kind, tc.wantKind)
			}
			if r.Required != tc.wantRequired {
				t.Errorf("Required = %v, want %v", r.Required, tc.wantRequired)
			}
			if len(rep.Relationships) != 1 || rep.Relationships[0] != tc.wantRendered {
				t.Errorf("rendered = %v, want [%q]", rep.Relationships, tc.wantRendered)
			}
			joined := strings.Join(rep.Warnings, "\n")
			if tc.wantWarning != "" && !strings.Contains(joined, tc.wantWarning) {
				t.Errorf("warnings must contain %q, got:\n%s", tc.wantWarning, joined)
			}
			if tc.wantWarning == "" && strings.Contains(joined, "self-reference") {
				t.Errorf("unexpected self-reference warning, got:\n%s", joined)
			}
		})
	}
}

func shopDocument() domain.DiagramDocument {
	return domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Shop",
		Classes: []domain.UmlClass{
			{
				ID: "c1", Name: "Order",
				PackageName: strp("shop"), TableBinding: strp("orders"),
				X: 10, Y: 20, Width: intp(220),
				Attributes: []domain.Attribute{
					{ID: "a1", Name: "total", Type: "Money", Visibility: strp("private")},
				},
				Methods: []domain.Method{
					{ID: "m1", Name: "checkout", ReturnType: "void", Visibility: strp("public")},
				},
			},
			{
				ID: "c2", Name: "Customer", X: 300, Y: 20,
				Attributes: []domain.Attribute{
					{ID: "a2", Name: "name", Type: "string"},
					{ID: "a3", Name: "email", Type: "String"},
				},
			},
		},
		Relationships: []domain.Relationship{
			{
				ID: "r1", SourceID: "c1", TargetID: "c2", Type: "association",
				SourceMultiplicity: strp("1"), TargetMultiplicity: strp("*"),
				Label: strp("placed by"),
			},
		},
	}
}

func TestExportShopGolden(t *testing.T) {
	gotJDL, rep := jdlgen.Export(shopDocument())
	assertGolden(t, filepath.Join("testdata", "shop.jdl.golden"), []byte(gotJDL))
	assertGolden(t, filepath.Join("testdata", "shop.report.golden"), reportJSON(t, rep))

	if len(rep.Warnings) != 0 {
		t.Errorf("shop should be warning-free, got %v", rep.Warnings)
	}
	if len(rep.Entities) != 2 || len(rep.Relationships) != 1 {
		t.Errorf("shop should yield 2 entities and 1 relationship, got %+v", rep)
	}
}

func edgeDocument() domain.DiagramDocument {
	return domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Edge",
		Classes: []domain.UmlClass{
			{
				ID: "e1", Name: "class",
				Attributes: []domain.Attribute{
					{ID: "a1", Name: "int", Type: "Mystery"},
				},
			},
			{
				ID: "e3", Name: "Order",
				Attributes: []domain.Attribute{
					{ID: "a2", Name: "title", Type: "String"},
				},
			},
			{
				ID: "e4", Name: "order",
				Attributes: []domain.Attribute{
					{ID: "a3", Name: "title", Type: "String"},
				},
			},
			{ID: "e5", Name: "Status", Stereotype: strp("Enum")},
		},
		Relationships: []domain.Relationship{
			{ID: "g1", SourceID: "e3", TargetID: "e4", Type: "generalization"},
			{ID: "d1", SourceID: "e3", TargetID: "e1", Type: "dependency"},
			{ID: "s1", SourceID: "e3", TargetID: "e3", Type: "association",
				SourceMultiplicity: strp("*"), TargetMultiplicity: strp("*")},
			{ID: "a1", SourceID: "e3", TargetID: "e1", Type: "aggregation",
				SourceMultiplicity: strp("1"), TargetMultiplicity: strp("*")},
			{ID: "x1", SourceID: "e1", TargetID: "nope", Type: "association"},
		},
	}
}

func TestExportEdgeGolden(t *testing.T) {
	gotJDL, rep := jdlgen.Export(edgeDocument())
	assertGolden(t, filepath.Join("testdata", "edge.jdl.golden"), []byte(gotJDL))
	assertGolden(t, filepath.Join("testdata", "edge.report.golden"), reportJSON(t, rep))

	joined := strings.Join(rep.Warnings, "\n")
	for _, want := range []string{
		`renamed to entity "Class"`,
		`renamed to entity "Order2"`,
		`stereotype "Enum" skipped`,
		`unknown UML type "Mystery"`,
		`generalization from "Order" to "Order2" skipped`,
		`unknown class id "nope"`,
		`flattened to plain JDL association(s)`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("warnings should contain %q, got:\n%s", want, joined)
		}
	}
}

// idAttributeDoc builds a document whose class carries a voice-created "id"
// attribute (case-insensitive, with surrounding whitespace) alongside a
// regular field, mirroring what the voice-command flow produces
// (frontend/src/App.tsx) before generation.
func idAttributeDoc(idType string) domain.DiagramDocument {
	return domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "voice",
		Classes: []domain.UmlClass{
			{
				ID: "c1", Name: "Cliente",
				Attributes: []domain.Attribute{
					{ID: "a1", Name: " Id ", Type: idType},
					{ID: "a2", Name: "nombre", Type: "string"},
				},
			},
		},
	}
}

func TestBuildModelSkipsIDAttribute(t *testing.T) {
	m, rep := jdlgen.BuildModel(idAttributeDoc("UUID"))
	if len(m.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %+v", m.Entities)
	}
	entity := m.Entities[0]
	for _, f := range entity.Fields {
		if strings.EqualFold(f.Name, "id") {
			t.Fatalf("expected no \"id\" field in the emitted Model, got %+v", entity.Fields)
		}
	}
	if len(entity.Fields) != 1 || entity.Fields[0].Name != "nombre" {
		t.Fatalf("expected only the \"nombre\" field to survive, got %+v", entity.Fields)
	}

	found := false
	for _, d := range rep.Dropped {
		if d.Kind == "attribute" && d.Location == "class Cliente attribute id" {
			found = true
			if !strings.Contains(d.Detail, "JHipster generates the Long primary key") {
				t.Errorf("Dropped detail must explain the JHipster-generated primary key, got %q", d.Detail)
			}
		}
	}
	if !found {
		t.Fatalf("expected a Dropped entry for the id attribute, got %+v", rep.Dropped)
	}

	joined := strings.Join(rep.Warnings, "\n")
	if !strings.Contains(joined, `attribute "id"`) {
		t.Errorf("a UUID-typed id attribute must warn (it does not map to Long), got warnings:\n%s", joined)
	}
}

func TestBuildModelSkipsIDAttributeNoWarningWhenLong(t *testing.T) {
	_, rep := jdlgen.BuildModel(idAttributeDoc("Long"))
	joined := strings.Join(rep.Warnings, "\n")
	if strings.Contains(joined, `attribute "id"`) {
		t.Errorf("a Long-typed id attribute must not warn, got warnings:\n%s", joined)
	}
}

// TestBuildModelSpanishTypeAliases covers VOICE-03: attribute types a
// Spanish voice transcript produces (case/accent-insensitive) must map to the
// same canonical JDL type the equivalent English/JDL word maps to, and must
// never trigger the "unknown UML type" warning.
func TestBuildModelSpanishTypeAliases(t *testing.T) {
	cases := []struct{ input, want string }{
		{"entero", "Integer"},
		{"ENTERO", "Integer"},
		{"texto", "String"},
		{"cadena", "String"},
		{"decimal", "BigDecimal"},
		{"fecha", "LocalDate"},
		{"fecha hora", "ZonedDateTime"},
		{"fecha y hora", "ZonedDateTime"},
		{"booleano", "Boolean"},
		{"lógico", "Boolean"},
		{"logico", "Boolean"},
		{"largo", "Long"},
		{"flotante", "Float"},
		{"doble", "Double"},
		{"uuid", "UUID"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			doc := domain.DiagramDocument{
				SchemaVersion: 1,
				Name:          "voice",
				Classes: []domain.UmlClass{
					{ID: "c1", Name: "Cliente", Attributes: []domain.Attribute{
						{ID: "a1", Name: "campo", Type: tc.input},
					}},
				},
			}
			m, rep := jdlgen.BuildModel(doc)
			if len(m.Entities) != 1 || len(m.Entities[0].Fields) != 1 {
				t.Fatalf("expected 1 field, got %+v", m.Entities)
			}
			got := m.Entities[0].Fields[0].Type
			if got != tc.want {
				t.Errorf("type %q mapped to %q, want %q", tc.input, got, tc.want)
			}
			joined := strings.Join(rep.Warnings, "\n")
			if strings.Contains(joined, "unknown UML type") {
				t.Errorf("unexpected unknown-type warning for %q:\n%s", tc.input, joined)
			}
		})
	}
}

func TestExportUnknownRelationshipTypeSkipped(t *testing.T) {
	doc := domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "odd",
		Classes: []domain.UmlClass{
			{ID: "a", Name: "Alpha"},
			{ID: "b", Name: "Beta"},
		},
		Relationships: []domain.Relationship{
			{ID: "r", SourceID: "a", TargetID: "b", Type: "telepathy"},
		},
	}
	_, rep := jdlgen.Export(doc)
	if len(rep.Relationships) != 0 {
		t.Fatalf("unknown type must not emit JDL, got %v", rep.Relationships)
	}
	if len(rep.Dropped) != 3 { // 2 layouts + 1 skipped relationship
		t.Fatalf("expected 3 dropped entries, got %+v", rep.Dropped)
	}
	if !strings.Contains(rep.Warnings[len(rep.Warnings)-1], `unknown type "telepathy"`) {
		t.Errorf("expected unknown-type warning, got %v", rep.Warnings)
	}
}

// cycleDoc builds classes a..n and one association per [src, dst, srcMult,
// dstMult] tuple, in order, with relationship IDs r1..rN.
func cycleDoc(classes []string, rels [][4]string) domain.DiagramDocument {
	doc := domain.DiagramDocument{SchemaVersion: 1, Name: "cycle"}
	for _, name := range classes {
		doc.Classes = append(doc.Classes, domain.UmlClass{ID: strings.ToLower(name), Name: name})
	}
	for i, r := range rels {
		doc.Relationships = append(doc.Relationships, domain.Relationship{
			ID: fmt.Sprintf("r%d", i+1), SourceID: strings.ToLower(r[0]), TargetID: strings.ToLower(r[1]),
			Type: "association", SourceMultiplicity: strp(r[2]), TargetMultiplicity: strp(r[3]),
		})
	}
	return doc
}

// TestBuildModelBreaksRequiredForeignKeyCycles covers required FKs that form
// a loop across entities: every NOT NULL column in the loop would make the
// first row of each table impossible to insert, so the relationship that
// closes the loop (the later one in document order) is kept optional.
func TestBuildModelBreaksRequiredForeignKeyCycles(t *testing.T) {
	cases := []struct {
		name         string
		classes      []string
		rels         [][4]string
		wantRequired []bool
		wantWarning  string
	}{
		{name: "two entities pointing at each other",
			classes:      []string{"Employee", "Department"},
			rels:         [][4]string{{"Employee", "Department", "*", "1"}, {"Department", "Employee", "*", "1"}},
			wantRequired: []bool{true, false},
			wantWarning:  `relationship r2: required foreign key from "Department" to "Employee" kept optional (it closes the required cycle Department -> Employee -> Department, so no row could be inserted first)`},
		{name: "three entity loop",
			classes:      []string{"Alpha", "Beta", "Gamma"},
			rels:         [][4]string{{"Alpha", "Beta", "*", "1"}, {"Beta", "Gamma", "*", "1"}, {"Gamma", "Alpha", "*", "1"}},
			wantRequired: []bool{true, true, false},
			wantWarning:  `relationship r3: required foreign key from "Gamma" to "Alpha" kept optional (it closes the required cycle Gamma -> Alpha -> Beta -> Gamma, so no row could be inserted first)`},
		{name: "one to many holder is the destination",
			classes:      []string{"Alpha", "Beta"},
			rels:         [][4]string{{"Alpha", "Beta", "1", "*"}, {"Alpha", "Beta", "*", "1"}},
			wantRequired: []bool{true, false},
			wantWarning:  `relationship r2: required foreign key from "Alpha" to "Beta" kept optional (it closes the required cycle Alpha -> Beta -> Alpha, so no row could be inserted first)`},
		{name: "chain without a loop stays required",
			classes:      []string{"Alpha", "Beta", "Gamma"},
			rels:         [][4]string{{"Alpha", "Beta", "*", "1"}, {"Beta", "Gamma", "*", "1"}},
			wantRequired: []bool{true, true}},
		{name: "optional edge does not close a loop",
			classes:      []string{"Employee", "Department"},
			rels:         [][4]string{{"Employee", "Department", "*", "1"}, {"Department", "Employee", "*", "0..1"}},
			wantRequired: []bool{true, false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, rep := jdlgen.BuildModel(cycleDoc(tc.classes, tc.rels))
			if len(m.Relationships) != len(tc.wantRequired) {
				t.Fatalf("got %d relationships, want %d", len(m.Relationships), len(tc.wantRequired))
			}
			for i, want := range tc.wantRequired {
				if m.Relationships[i].Required != want {
					t.Errorf("relationship %d Required = %v, want %v", i+1, m.Relationships[i].Required, want)
				}
			}
			var cycleWarnings []string
			for _, w := range rep.Warnings {
				if strings.Contains(w, "required cycle") {
					cycleWarnings = append(cycleWarnings, w)
				}
			}
			switch {
			case tc.wantWarning == "" && len(cycleWarnings) > 0:
				t.Errorf("unexpected cycle warnings: %v", cycleWarnings)
			case tc.wantWarning != "" && (len(cycleWarnings) != 1 || cycleWarnings[0] != tc.wantWarning):
				t.Errorf("cycle warnings = %v, want [%s]", cycleWarnings, tc.wantWarning)
			}
		})
	}
}
