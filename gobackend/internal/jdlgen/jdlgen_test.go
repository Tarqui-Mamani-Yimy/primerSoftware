package jdlgen_test

import (
	"bytes"
	"encoding/json"
	"flag"
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
		{"class", "Class_"},
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
		{"Class_", "class_"},
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

func TestEnsureUnique(t *testing.T) {
	used := map[string]struct{}{}
	if got := jdlgen.EnsureUnique("Order", used); got != "Order" {
		t.Fatalf("first use = %q, want %q", got, "Order")
	}
	if got := jdlgen.EnsureUnique("Order", used); got != "Order_2" {
		t.Fatalf("second use = %q, want %q", got, "Order_2")
	}
	if got := jdlgen.EnsureUnique("Order", used); got != "Order_3" {
		t.Fatalf("third use = %q, want %q", got, "Order_3")
	}
	if got := jdlgen.EnsureUnique("Customer", used); got != "Customer" {
		t.Fatalf("fresh base = %q, want %q", got, "Customer")
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
		{name: "missing means one to one", src: nil, dst: nil, want: "OneToOne Alpha{beta} to Beta{alpha}"},
		{name: "one to many", src: strp("1"), dst: strp("*"), want: "OneToMany Alpha{beta} to Beta{alpha}"},
		{name: "many to one", src: strp("*"), dst: strp("1"), want: "ManyToOne Alpha{beta} to Beta{alpha}"},
		{name: "many to many", src: strp("*"), dst: strp("*"), want: "ManyToMany Alpha{beta} to Beta{alpha}"},
		{name: "range many", src: strp("0..1"), dst: strp("1..*"), want: "OneToMany Alpha{beta} to Beta{alpha}"},
		{name: "numeric many", src: strp("2"), dst: strp("1"), want: "ManyToOne Alpha{beta} to Beta{alpha}"},
		{name: "one to one ranges", src: strp("1"), dst: strp("1..1"), want: "OneToOne Alpha{beta} to Beta{alpha}"},
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
		`renamed to entity "Class_"`,
		`renamed to entity "Order_2"`,
		`stereotype "Enum" skipped`,
		`unknown UML type "Mystery"`,
		`generalization from "Order" to "Order_2" skipped`,
		`unknown class id "nope"`,
		`flattened to plain JDL association(s)`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("warnings should contain %q, got:\n%s", want, joined)
		}
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
