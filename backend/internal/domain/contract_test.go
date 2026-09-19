package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/domain"
)

func strPtr(s string) *string { return &s }

func intPtr(i int) *int { return &i }

func sampleDocument() domain.DiagramDocument {
	return domain.DiagramDocument{
		SchemaVersion: 1,
		ID:            strPtr("doc-1"),
		Name:          "sample",
		Classes: []domain.UmlClass{
			{
				ID: "c1", Name: "Order",
				Stereotype:   strPtr("Entity"),
				PackageName:  strPtr("shop"),
				TableBinding: strPtr("orders"),
				X:            10, Y: 20, Width: intPtr(220),
				Attributes: []domain.Attribute{
					{ID: "a1", Name: "total", Type: "Money", Visibility: strPtr("private")},
				},
				Methods: []domain.Method{
					{ID: "m1", Name: "checkout", ReturnType: "void", Visibility: strPtr("public")},
				},
			},
			{ID: "c2", Name: "Customer", X: 300, Y: 20},
		},
		Relationships: []domain.Relationship{
			{
				ID: "r1", SourceID: "c1", TargetID: "c2", Type: "association",
				SourceMultiplicity: strPtr("1"), TargetMultiplicity: strPtr("*"),
				Label: strPtr("placed by"),
			},
		},
	}
}

func mustMarshal(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal to map failed: %v", err)
	}
	return decoded
}

func assertKeys(t *testing.T, decoded map[string]any, want []string) {
	t.Helper()
	for _, key := range want {
		if _, ok := decoded[key]; !ok {
			t.Errorf("missing JSON key %q in %v", key, decoded)
		}
	}
}

func TestDiagramDocumentJSONShape(t *testing.T) {
	decoded := mustMarshal(t, sampleDocument())
	assertKeys(t, decoded, []string{"schemaVersion", "id", "name", "classes", "relationships"})

	classes, ok := decoded["classes"].([]any)
	if !ok || len(classes) != 2 {
		t.Fatalf("expected 2 classes, got %v", decoded["classes"])
	}
	class, ok := classes[0].(map[string]any)
	if !ok {
		t.Fatalf("class is not an object: %v", classes[0])
	}
	assertKeys(t, class, []string{
		"id", "name", "stereotype", "package", "tableBinding",
		"x", "y", "width", "attributes", "methods",
	})
	if class["package"] != "shop" {
		t.Errorf("expected package %q, got %v", "shop", class["package"])
	}
	if class["x"] != float64(10) || class["y"] != float64(20) {
		t.Errorf("expected x/y 10/20, got %v/%v", class["x"], class["y"])
	}

	rels, ok := decoded["relationships"].([]any)
	if !ok || len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %v", decoded["relationships"])
	}
	rel, ok := rels[0].(map[string]any)
	if !ok {
		t.Fatalf("relationship is not an object: %v", rels[0])
	}
	assertKeys(t, rel, []string{
		"id", "sourceId", "targetId", "type",
		"sourceMultiplicity", "targetMultiplicity", "label",
	})
	if rel["type"] != "association" {
		t.Errorf("expected type association, got %v", rel["type"])
	}
}

func TestDiagramDocumentRoundTrip(t *testing.T) {
	want := sampleDocument()
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var got domain.DiagramDocument
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.Name != want.Name || got.SchemaVersion != want.SchemaVersion {
		t.Errorf("round trip changed document: %+v", got)
	}
	if len(got.Classes) != 2 || len(got.Relationships) != 1 {
		t.Errorf("round trip lost members: %+v", got)
	}
	if got.Classes[0].PackageName == nil || *got.Classes[0].PackageName != "shop" {
		t.Errorf("round trip lost package binding: %+v", got.Classes[0])
	}
}

func TestDiagramDocumentMissingIDEmitsNull(t *testing.T) {
	// The frontend type declares id?: string, and the Java record UUID id has no
	// @NotNull; Jackson must serialize a missing id as "id": null, never omit it.
	decoded := mustMarshal(t, domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "no-id",
		Classes:       []domain.UmlClass{},
		Relationships: []domain.Relationship{},
	})
	id, ok := decoded["id"]
	if !ok {
		t.Fatalf("Jackson emits %q for a missing id; Go omitted the key: %v", "id:null", decoded)
	}
	if id != nil {
		t.Errorf("expected null id (Jackson parity), got %v", id)
	}
}

func TestProjectNullDescriptionEmitsNull(t *testing.T) {
	// V1 defines projects.description TEXT without NOT NULL, so a null must
	// serialize as "description": null (Jackson default), never "".
	decoded := mustMarshal(t, domain.ProjectResponse{
		ID: "p1", Name: "shop", Role: "OWNER",
	})
	desc, ok := decoded["description"]
	if !ok {
		t.Fatalf("Jackson emits %q for a null description; Go omitted the key: %v", "description:null", decoded)
	}
	if desc != nil {
		t.Errorf("expected null description (Jackson parity), got %v", desc)
	}
}

func TestRelationshipTypeSet(t *testing.T) {
	want := []string{"association", "aggregation", "composition", "generalization", "realization", "dependency"}
	if len(domain.RelationshipTypes) != len(want) {
		t.Fatalf("expected %d relationship types, got %v", len(want), domain.RelationshipTypes)
	}
	for i, w := range want {
		if domain.RelationshipTypes[i] != w {
			t.Fatalf("expected relationship types %v, got %v", want, domain.RelationshipTypes)
		}
	}
}

func TestIsValidRelationshipType(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "association accepted", input: "association", want: true},
		{name: "aggregation accepted", input: "aggregation", want: true},
		{name: "composition accepted", input: "composition", want: true},
		{name: "generalization accepted", input: "generalization", want: true},
		{name: "realization accepted", input: "realization", want: true},
		{name: "dependency accepted", input: "dependency", want: true},
		{name: "wrong case rejected", input: "Association", want: false},
		{name: "unknown rejected", input: "extends", want: false},
		{name: "empty rejected", input: "", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domain.IsValidRelationshipType(tc.input); got != tc.want {
				t.Errorf("IsValidRelationshipType(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestLoginJSONKeys(t *testing.T) {
	req := mustMarshal(t, domain.LoginRequest{Email: "a@b.c", Password: "secret"})
	assertKeys(t, req, []string{"email", "password"})

	resp := mustMarshal(t, domain.LoginResponse{
		AccessToken: "tok", UserID: "u1", DisplayName: "Ada", Email: "a@b.c",
	})
	assertKeys(t, resp, []string{"accessToken", "userId", "displayName", "email"})
}

func TestProjectJSONKeys(t *testing.T) {
	decoded := mustMarshal(t, domain.ProjectResponse{
		ID: "p1", Name: "shop", Description: strPtr("d"), Role: "OWNER", DiagramCount: 2,
	})
	assertKeys(t, decoded, []string{"id", "name", "description", "role", "diagramCount"})
}

func TestCreateAndJoinProjectJSONKeys(t *testing.T) {
	created := mustMarshal(t, domain.ProjectCreatedResponse{
		ID: "p1", Name: "shop", Description: strPtr("d"), Role: "OWNER", AccessCode: "X7K2P9",
	})
	assertKeys(t, created, []string{"id", "name", "description", "role", "diagramCount", "accessCode"})

	// A missing description must serialize as null (nullable projects.description).
	nullDesc := mustMarshal(t, domain.ProjectCreatedResponse{ID: "p2", Name: "shop", Role: "OWNER", AccessCode: "X7K2P9"})
	if desc, ok := nullDesc["description"]; !ok || desc != nil {
		t.Errorf("expected \"description\": null, got %v", nullDesc)
	}

	join := mustMarshal(t, domain.JoinProjectRequest{AccessCode: "x"})
	assertKeys(t, join, []string{"accessCode"})

	create := mustMarshal(t, domain.CreateProjectRequest{Name: "shop"})
	assertKeys(t, create, []string{"name", "description"})
}

func TestErrorEnvelopeJSONKey(t *testing.T) {
	raw, err := json.Marshal(domain.ErrorEnvelope{Message: "boom"})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(raw) != `{"message":"boom"}` {
		t.Errorf("expected exact {message} envelope, got %s", raw)
	}
}
