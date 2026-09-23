package jdlgen_test

import (
	"strings"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
)

func boolPtr(b bool) *bool { return &b }

func TestDefaultOptionsAreValid(t *testing.T) {
	if errs := jdlgen.ValidateOptions(jdlgen.DefaultOptions()); len(errs) > 0 {
		t.Fatalf("default options invalid: %v", errs)
	}
}

func TestValidateOptions(t *testing.T) {
	cases := []struct {
		name string
		opts jdlgen.Options
		want []string
	}{
		{name: "empty options", opts: jdlgen.Options{}, want: []string{"baseName is required", "packageName is required", `buildTool must be "maven" or "gradle" (got "")`, `authenticationType must be "jwt" (got "")`}},
		{name: "blank baseName", opts: jdlgen.Options{BaseName: "   ", PackageName: "com.x", BuildTool: "maven", AuthenticationType: "jwt"}, want: []string{"baseName is required"}},
		{name: "baseName with space", opts: jdlgen.Options{BaseName: "My App", PackageName: "com.x", BuildTool: "maven", AuthenticationType: "jwt"}, want: []string{"baseName must be alphanumeric"}},
		{name: "baseName ending in App", opts: jdlgen.Options{BaseName: "ShopApp", PackageName: "com.x", BuildTool: "maven", AuthenticationType: "jwt"}, want: []string{`baseName must not end with "App"`}},
		{name: "package starts with dot", opts: jdlgen.Options{BaseName: "Shop", PackageName: ".com.x", BuildTool: "maven", AuthenticationType: "jwt"}, want: []string{"packageName must be a dot-separated list"}},
		{name: "package segment with hyphen", opts: jdlgen.Options{BaseName: "Shop", PackageName: "com.my-app", BuildTool: "maven", AuthenticationType: "jwt"}, want: []string{`packageName segment "my-app" is not a legal Java identifier`}},
		{name: "package reserved word", opts: jdlgen.Options{BaseName: "Shop", PackageName: "com.class", BuildTool: "maven", AuthenticationType: "jwt"}, want: []string{`packageName segment "class" is not a legal Java identifier`}},
		{name: "bad buildTool", opts: jdlgen.Options{BaseName: "Shop", PackageName: "com.x", BuildTool: "npm", AuthenticationType: "jwt"}, want: []string{`buildTool must be "maven" or "gradle"`}},
		{name: "bad authenticationType", opts: jdlgen.Options{BaseName: "Shop", PackageName: "com.x", BuildTool: "maven", AuthenticationType: "oauth2"}, want: []string{`authenticationType must be "jwt"`}},
		{name: "valid gradle", opts: jdlgen.Options{BaseName: "Shop", PackageName: "com.x", BuildTool: "gradle", AuthenticationType: "jwt"}, want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := jdlgen.ValidateOptions(tc.opts)
			if tc.want == nil && len(got) != 0 {
				t.Fatalf("expected no errors, got %v", got)
			}
			if tc.want != nil && len(got) != len(tc.want) {
				t.Fatalf("expected %d errors %v, got %v", len(tc.want), tc.want, got)
			}
			for i := range tc.want {
				if !strings.HasPrefix(got[i], tc.want[i]) {
					t.Errorf("error[%d] = %q, want prefix %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestExportArtifactIncludesApplicationBlock(t *testing.T) {
	doc := domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Shop",
		Classes: []domain.UmlClass{
			{ID: "a", Name: "Alpha", Attributes: []domain.Attribute{{ID: "a1", Name: "total", Type: "BigDecimal"}}},
		},
	}
	jdl, rep, err := jdlgen.ExportArtifact(doc, jdlgen.DefaultOptions())
	if err != nil {
		t.Fatalf("ExportArtifact: %v", err)
	}
	for _, want := range []string{
		"application {",
		"baseName UmlArchitect",
		"applicationType monolith",
		"packageName com.umlarchitect",
		"authenticationType jwt",
		"buildTool maven",
		"databaseType sql",
		"prodDatabaseType postgresql",
		"skipClient true",
		"serverPort 8081",
		"entities Alpha",
		"service * with serviceImpl",
		"dto * with mapstruct",
		"paginate * with pagination",
		"filter *",
		"entity Alpha {",
		"  total BigDecimal",
	} {
		if !strings.Contains(jdl, want) {
			t.Errorf("JDL missing %q:\n%s", want, jdl)
		}
	}
	if len(rep.Entities) != 1 || rep.Entities[0] != "Alpha" {
		t.Errorf("report entities = %v, want [Alpha]", rep.Entities)
	}
	// The application block precedes the generated model header when both
	// are present, keeping single-file JDL scaffolds parseable top-down.
	if !strings.HasPrefix(jdl, "application {") {
		t.Errorf("JDL must start with the application block:\n%s", jdl)
	}
}

func TestExportArtifactInvalidOptions(t *testing.T) {
	_, _, err := jdlgen.ExportArtifact(domain.DiagramDocument{}, jdlgen.Options{BaseName: "Bad App"})
	if err == nil || !strings.Contains(err.Error(), "invalid JHipster application options") {
		t.Fatalf("expected options error, got %v", err)
	}
}

func TestExportWarnsAssociationClass(t *testing.T) {
	doc := domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Shop",
		Classes: []domain.UmlClass{
			{ID: "a", Name: "Alpha"},
			{ID: "b", Name: "Beta"},
			{ID: "c", Name: "Membership", IsAssociationClass: boolPtr(true), AttachedRelationshipID: strp("rel-9")},
		},
	}
	_, rep := jdlgen.Export(doc)
	found := false
	for _, w := range rep.Warnings {
		if strings.Contains(w, "association class") && strings.Contains(w, "rel-9") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected association-class warning, warnings: %v", rep.Warnings)
	}
	if len(rep.Entities) != 3 {
		t.Fatalf("association class must still be emitted as an entity, entities: %v", rep.Entities)
	}
}

func TestExportSynthesizesAssociationLinks(t *testing.T) {
	// Membership is an association class attached to rel-1 (Course→Student);
	// BuildModel must emit real ManyToOne links from Membership to BOTH ends.
	doc := domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "UML",
		Classes: []domain.UmlClass{
			{ID: "a", Name: "Course"},
			{ID: "b", Name: "Student"},
			{ID: "c", Name: "Membership", IsAssociationClass: boolPtr(true), AttachedRelationshipID: strp("rel-1")},
		},
		Relationships: []domain.Relationship{
			{
				ID: "rel-1", SourceID: "a", TargetID: "b", Type: "association",
				SourceMultiplicity: strp("*"), TargetMultiplicity: strp("*"),
			},
		},
	}
	jdl, rep := jdlgen.Export(doc)

	wantLinks := []string{
		"Membership{course} to Course{membership}",
		"Membership{student} to Student{membership}",
	}
	for _, want := range wantLinks {
		if !strings.Contains(jdl, want) {
			t.Errorf("JDL missing synthesized link %q:\n%s", want, jdl)
		}
	}
	if len(rep.Relationships) != 3 {
		t.Fatalf("expected the attachment plus 2 synthesized links, relationships: %v", rep.Relationships)
	}
	warned := 0
	for _, w := range rep.Warnings {
		if strings.Contains(w, "linked to association end") {
			warned++
		}
	}
	if warned != 2 {
		t.Fatalf("expected 2 association-end warnings, got %d: %v", warned, rep.Warnings)
	}
}

func TestExportSynthesizedLinkAvoidsFieldCollision(t *testing.T) {
	// The association class already declares an attribute named after an
	// endpoint (Course→field "student"): the synthesized ManyToOne must not
	// overwrite it, so it becomes "student2".
	doc := domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "UML",
		Classes: []domain.UmlClass{
			{ID: "a", Name: "Teacher"},
			{ID: "b", Name: "Student"},
			{ID: "c", Name: "Course", IsAssociationClass: boolPtr(true), AttachedRelationshipID: strp("rel-1"),
				Attributes: []domain.Attribute{{ID: "a1", Name: "student", Type: "string"}}},
		},
		Relationships: []domain.Relationship{
			{
				ID: "rel-1", SourceID: "a", TargetID: "b", Type: "association",
				SourceMultiplicity: strp("1"), TargetMultiplicity: strp("*"),
			},
		},
	}
	jdl, _ := jdlgen.Export(doc)
	for _, want := range []string{
		"Course{teacher} to Teacher{course}",
		"Course{student2} to Student{course}",
	} {
		if !strings.Contains(jdl, want) {
			t.Errorf("JDL missing %q:\n%s", want, jdl)
		}
	}
}
