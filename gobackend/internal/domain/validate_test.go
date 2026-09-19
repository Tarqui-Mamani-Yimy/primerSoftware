package domain_test

import (
	"strings"
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/domain"
)

func validDoc(rels ...domain.Relationship) domain.DiagramDocument {
	return domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Orders",
		Classes: []domain.UmlClass{
			{ID: "order", Name: "Order", Stereotype: strPtr("«Entity»")},
			{ID: "customer", Name: "Customer", Stereotype: strPtr("«Interface»")},
		},
		Relationships: rels,
	}
}

func rel(id, source, target, typ string, sourceMult, targetMult *string) domain.Relationship {
	return domain.Relationship{
		ID: id, SourceID: source, TargetID: target, Type: typ,
		SourceMultiplicity: sourceMult, TargetMultiplicity: targetMult,
	}
}

func assertContains(t *testing.T, errors []string, want string) {
	t.Helper()
	for _, e := range errors {
		if strings.Contains(e, want) {
			return
		}
	}
	t.Errorf("expected an error containing %q, got %v", want, errors)
}

func TestValidateAcceptsValidRelationshipDocument(t *testing.T) {
	doc := validDoc(rel("rel-1", "order", "customer", "association", strPtr("1"), strPtr("0..*")))
	if errs := domain.ValidateDocument(doc); len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidateRejectsMissingEndpointsUnsupportedTypesAndDuplicates(t *testing.T) {
	assertContains(t,
		domain.ValidateDocument(validDoc(rel("rel-1", "order", "missing", "association", nil, nil))),
		"missing class")
	assertContains(t,
		domain.ValidateDocument(validDoc(rel("rel-1", "order", "customer", "invalid", nil, nil))),
		"unsupported type")

	first := rel("rel-1", "order", "customer", "association", nil, nil)
	duplicate := rel("rel-2", "order", "customer", "association", nil, nil)
	assertContains(t, domain.ValidateDocument(validDoc(first, duplicate)), "duplicate")
}

func TestValidateRejectsInvalidMultiplicityAndInvalidRealization(t *testing.T) {
	assertContains(t,
		domain.ValidateDocument(validDoc(rel("rel-1", "order", "customer", "association", strPtr("one"), strPtr("0..*")))),
		"invalid source multiplicity")
	assertContains(t,
		domain.ValidateDocument(validDoc(rel("rel-1", "customer", "order", "realization", nil, nil))),
		"Realization")
}

func TestValidateMultiplicityTable(t *testing.T) {
	cases := []struct {
		name  string
		value *string
		valid bool
	}{
		{name: "single digit accepted", value: strPtr("1"), valid: true},
		{name: "star accepted", value: strPtr("*"), valid: true},
		{name: "range accepted", value: strPtr("0..*"), valid: true},
		{name: "spaced range accepted", value: strPtr(" 1 .. * "), valid: true},
		{name: "word rejected", value: strPtr("one"), valid: false},
		{name: "trailing text rejected", value: strPtr("1..* extra"), valid: false},
		{name: "empty range rejected", value: strPtr(".."), valid: false},
		{name: "nil skipped", value: nil, valid: true},
		{name: "blank skipped", value: strPtr("   "), valid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := validDoc(rel("rel-1", "order", "customer", "association", tc.value, nil))
			errs := domain.ValidateDocument(doc)
			if tc.valid && len(errs) != 0 {
				t.Errorf("expected no errors, got %v", errs)
			}
			if !tc.valid {
				assertContains(t, errs, "invalid source multiplicity")
			}
		})
	}
}

func TestValidateRejectsSelfLoopDuplicateClassIdsAndDuplicateRelIds(t *testing.T) {
	assertContains(t,
		domain.ValidateDocument(validDoc(rel("rel-1", "order", "order", "association", nil, nil))),
		"cannot connect a class to itself")

	dupClasses := validDoc()
	dupClasses.Classes = append(dupClasses.Classes, domain.UmlClass{ID: "order", Name: "Order2"})
	assertContains(t, domain.ValidateDocument(dupClasses), "Class id must be unique: order")

	doc := validDoc(
		rel("rel-1", "order", "customer", "association", nil, nil),
		rel("rel-1", "customer", "order", "dependency", nil, nil),
	)
	assertContains(t, docErrors(doc), "Relationship id must be unique: rel-1")
}

func docErrors(doc domain.DiagramDocument) []string {
	return domain.ValidateDocument(doc)
}

func TestValidateRealizationRequiresInterfaceTarget(t *testing.T) {
	// Non-interface source with non-interface target still violates the rule.
	doc := validDoc(rel("rel-1", "order", "order-x", "realization", nil, nil))
	_ = doc
	plain := domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Orders",
		Classes: []domain.UmlClass{
			{ID: "a", Name: "A"},
			{ID: "b", Name: "B"},
		},
		Relationships: []domain.Relationship{
			{ID: "rel-1", SourceID: "a", TargetID: "b", Type: "realization"},
		},
	}
	assertContains(t, domain.ValidateDocument(plain), "Realization")

	iface := domain.DiagramDocument{
		SchemaVersion: 1,
		Name:          "Orders",
		Classes: []domain.UmlClass{
			{ID: "a", Name: "A"},
			{ID: "b", Name: "B", Stereotype: strPtr("«Interface»")},
		},
		Relationships: []domain.Relationship{
			{ID: "rel-1", SourceID: "a", TargetID: "b", Type: "realization"},
		},
	}
	if errs := domain.ValidateDocument(iface); len(errs) != 0 {
		t.Errorf("expected valid realization, got %v", errs)
	}
}
