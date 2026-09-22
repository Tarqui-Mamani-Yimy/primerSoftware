package jdlgen

import (
	"fmt"
	"strings"

	"github.com/ai-uml-architect/gobackend/internal/domain"
)

// Model is the structured conversion of a UML document that both renderers
// (JDL text and PostgreSQL init SQL) consume, so the downloadable artifact
// stays internally consistent: one naming pass, one set of relationships.
type Model struct {
	DiagramName   string
	Entities      []Entity
	Relationships []Relationship
}

// Entity is one emitted JDL entity with its scalar fields.
type Entity struct {
	Name   string
	Fields []Field
}

// Field is one emitted JDL field.
type Field struct {
	Name string
	Type string
}

// Relationship models one emitted JDL relationship line. Src is the owning
// side: it holds the foreign key for OneToOne/ManyToOne, its field names the
// collection for OneToMany (the FK lives on Dst), and it owns the join table
// for ManyToMany.
type Relationship struct {
	Kind     string // OneToOne | OneToMany | ManyToOne | ManyToMany
	Src      string
	Dst      string
	SrcField string // field declared on the source/owning entity
	DstField string // field declared on the destination entity
	// Required marks the FK-holding side's reference as mandatory: the
	// generated JDL gets `required` on that field and the SQL column gets
	// NOT NULL. It derives from the lower bound of the multiplicity on the
	// REFERENCED end (see isRequiredEnd) and is always false for ManyToMany
	// and for a self-reference, so the first row stays insertable.
	Required bool
}

// pluralField returns the collection-side field name in the same shape the
// pinned generator's naive inflector produces (append "s", or "es" when the
// word already ends in "s"). Declaring the plural form explicitly prevents
// JHipster from auto-deriving an inverse collection field, which fails the
// JDL parse with "duplicate properties in entity X: field".
func pluralField(s string) string {
	if strings.HasSuffix(s, "s") {
		return s + "es"
	}
	return s + "s"
}

// BuildModel converts doc into the structured Model plus the machine-readable
// report of every dropped construct. Output order follows the document:
// classes and relationships appear in input order, so repeated runs are
// byte-identical. Rule notes:
//
//   - Field/relationship naming mirrors the pinned generator's own
//     conventions: FK-holding sides use the singular FieldName of the other
//     entity, collection sides use the plural form (see pluralField).
//   - A UML class marked as an association class (targeting an attached
//     relationship) is emitted as a plain JPA entity with explicit ManyToOne
//     links to BOTH association ends when the attachment resolves; JHipster
//     has no association-class construct.
func BuildModel(doc domain.DiagramDocument) (Model, Report) {
	rep := Report{
		DiagramName:   doc.Name,
		Entities:      []string{},
		Relationships: []string{},
		Dropped:       []Dropped{},
		Warnings:      []string{},
	}
	m := Model{DiagramName: doc.Name}

	// Phase 1: assign entity names in class order; skip «Enum» classes.
	idToEntity := map[string]string{}
	skipped := map[string]bool{}
	usedEntities := map[string]struct{}{}
	for _, class := range doc.Classes {
		if IsEnumStereotype(class.Stereotype) {
			skipped[class.ID] = true
			rep.Dropped = append(rep.Dropped, Dropped{
				Kind:     "enum",
				Location: "class " + class.Name,
				Detail:   `stereotype "Enum" without value list (the UML Attribute carries id/name/type/visibility only); JDL enum not emitted`,
			})
			rep.Warnings = append(rep.Warnings, fmt.Sprintf(
				`class %q with stereotype "Enum" skipped (no enum values in the UML model); remodel it as a JDL enum manually`, class.Name))
			continue
		}
		base := EntityName(class.Name)
		final := EnsureUnique(base, usedEntities)
		reason := "Java identifier sanitization"
		switch {
		case final != base:
			reason = "name collision after sanitization"
		case isJDLReservedWord(strings.TrimSuffix(base, "2")):
			reason = "JDL reserved word (would break the JHipster JDL parser)"
		}
		if final != class.Name {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf(
				"class %q renamed to entity %q (%s)", class.Name, final, reason))
		}
		idToEntity[class.ID] = final
		rep.Entities = append(rep.Entities, final)
	}

	// Phase 2: emit entity blocks with attribute fields.
	for _, class := range doc.Classes {
		if skipped[class.ID] {
			continue
		}
		entity := Entity{Name: idToEntity[class.ID]}
		usedFields := map[string]struct{}{}
		rep.Dropped = append(rep.Dropped, Dropped{
			Kind:     "layout",
			Location: "class " + entity.Name,
			Detail:   layoutDetail(class),
		})
		if class.IsAssociationClass != nil && *class.IsAssociationClass {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf(
				"class %q is an association class (attached relationship %s); emitted as a plain JPA entity with explicit ManyToOne links to both association ends (JHipster has no association-class construct)",
				entity.Name, attachedRelationshipRef(class)))
		}
		if class.PackageName != nil && strings.TrimSpace(*class.PackageName) != "" {
			rep.Dropped = append(rep.Dropped, Dropped{
				Kind:     "package",
				Location: "class " + entity.Name,
				Detail:   fmt.Sprintf("package %q (JHipster derives packages from baseName)", *class.PackageName),
			})
		}
		if class.TableBinding != nil && strings.TrimSpace(*class.TableBinding) != "" {
			rep.Dropped = append(rep.Dropped, Dropped{
				Kind:     "tableBinding",
				Location: "class " + entity.Name,
				Detail:   fmt.Sprintf("table %q (JHipster generates table names)", *class.TableBinding),
			})
		}
		for _, attr := range class.Attributes {
			base := FieldName(attr.Name)
			final := EnsureUnique(base, usedFields)
			if final != attr.Name {
				reason := "Java identifier sanitization"
				switch {
				case final != base:
					reason = "name collision after sanitization"
				case isJDLReservedWord(strings.TrimSuffix(base, "2")):
					reason = "JDL reserved word (would break the JHipster JDL parser)"
				}
				rep.Warnings = append(rep.Warnings, fmt.Sprintf(
					"class %q attribute %q renamed to field %q (%s)", entity.Name, attr.Name, final, reason))
			}
			jdlType, known := jdlTypeFor(attr.Type)
			if !known {
				rep.Warnings = append(rep.Warnings, fmt.Sprintf(
					"class %q field %q: unknown UML type %q; emitted as String", entity.Name, final, attr.Type))
			}
			entity.Fields = append(entity.Fields, Field{Name: final, Type: jdlType})
			if attr.Visibility != nil && strings.TrimSpace(*attr.Visibility) != "" {
				rep.Dropped = append(rep.Dropped, Dropped{
					Kind:     "visibility",
					Location: "class " + entity.Name + " attribute " + attr.Name,
					Detail:   fmt.Sprintf("visibility %q (not expressed in JDL)", *attr.Visibility),
				})
			}
		}
		for _, method := range class.Methods {
			rep.Dropped = append(rep.Dropped, Dropped{
				Kind:     "method",
				Location: "class " + entity.Name,
				Detail:   fmt.Sprintf("%s(): %s (behavior is not expressed in JDL)", method.Name, method.ReturnType),
			})
			if method.Visibility != nil && strings.TrimSpace(*method.Visibility) != "" {
				rep.Dropped = append(rep.Dropped, Dropped{
					Kind:     "visibility",
					Location: "class " + entity.Name + " method " + method.Name,
					Detail:   fmt.Sprintf("visibility %q (not expressed in JDL)", *method.Visibility),
				})
			}
		}
		m.Entities = append(m.Entities, entity)
	}

	// Phase 3: emit relationships in document order, then synthesize the
	// association-class links to both ends of the attached relationship.
	entityFields := map[string]map[string]struct{}{}
	for _, e := range m.Entities {
		used := map[string]struct{}{}
		for _, f := range e.Fields {
			used[f.Name] = struct{}{}
		}
		entityFields[e.Name] = used
	}
	flattened := 0
	for _, rel := range doc.Relationships {
		src, srcOK := idToEntity[rel.SourceID]
		dst, dstOK := idToEntity[rel.TargetID]
		if !srcOK || !dstOK {
			rep.Dropped = append(rep.Dropped, Dropped{
				Kind:     "relationship",
				Location: "relationship " + rel.ID,
				Detail:   fmt.Sprintf("%s with dangling endpoint (unknown class id %q)", rel.Type, danglingID(rel, srcOK)),
			})
			rep.Warnings = append(rep.Warnings, fmt.Sprintf(
				"relationship %s: unknown class id %q; relationship skipped", rel.ID, danglingID(rel, srcOK)))
			continue
		}
		switch rel.Type {
		case "association", "aggregation", "composition":
			card := cardinality(isManySide(rel.SourceMultiplicity), isManySide(rel.TargetMultiplicity))
			r := relationshipFields(card, src, dst, entityFields)
			r.Required = requiredEnd(card, rel.SourceMultiplicity, rel.TargetMultiplicity)
			if src == dst && r.Required {
				// A required self-FK makes the first row impossible to
				// insert (it would need to reference a row that does not
				// exist yet), so self-references always stay optional.
				r.Required = false
				rep.Warnings = append(rep.Warnings, fmt.Sprintf(
					`relationship %s: self-reference %q kept optional (a required self-FK makes the first row impossible to insert)`, rel.ID, src))
			}
			m.Relationships = append(m.Relationships, r)
			rep.Relationships = append(rep.Relationships, renderRelationship(r))
			if rel.Label != nil && strings.TrimSpace(*rel.Label) != "" {
				rep.Dropped = append(rep.Dropped, Dropped{
					Kind:     "label",
					Location: "relationship " + rel.ID,
					Detail:   fmt.Sprintf("label %q (not expressed in JDL)", *rel.Label),
				})
			}
			if rel.Type == "aggregation" || rel.Type == "composition" {
				flattened++
			}
		case "generalization", "realization", "dependency":
			rep.Dropped = append(rep.Dropped, Dropped{
				Kind:     "relationship",
				Location: "relationship " + rel.ID,
				Detail:   skipDetail(rel.Type, src, dst),
			})
			rep.Warnings = append(rep.Warnings, fmt.Sprintf(
				"relationship %s: %s from %q to %q skipped (%s)", rel.ID, rel.Type, src, dst, skipAdvice(rel.Type)))
		default:
			rep.Dropped = append(rep.Dropped, Dropped{
				Kind:     "relationship",
				Location: "relationship " + rel.ID,
				Detail:   fmt.Sprintf("unknown relationship type %q; relationship skipped", rel.Type),
			})
			rep.Warnings = append(rep.Warnings, fmt.Sprintf(
				"relationship %s: unknown type %q; relationship skipped", rel.ID, rel.Type))
		}
	}
	if flattened > 0 {
		rep.Warnings = append(rep.Warnings, fmt.Sprintf(
			"%d aggregation/composition relationship(s) flattened to plain JDL association(s); UML ownership and cascade semantics are not expressed in JDL", flattened))
	}

	// Association-class synthesis: a class attached to a relationship gets
	// real ManyToOne links to both of that relationship's endpoints, so the
	// generated JPA entity and its table actually reference the ends.
	for _, class := range doc.Classes {
		if class.IsAssociationClass == nil || !*class.IsAssociationClass {
			continue
		}
		attached := ""
		if class.AttachedRelationshipID != nil {
			attached = strings.TrimSpace(*class.AttachedRelationshipID)
		}
		if attached == "" {
			continue
		}
		var srcID, dstID string
		for _, rel := range doc.Relationships {
			if rel.ID == attached {
				srcID, dstID = rel.SourceID, rel.TargetID
				break
			}
		}
		if srcID == "" || dstID == "" {
			continue // phase 2 already warned about the missing attachment
		}
		cc, ok := idToEntity[class.ID]
		if !ok {
			continue
		}
		for _, endpointID := range []string{srcID, dstID} {
			endpoint, ok := idToEntity[endpointID]
			if !ok {
				continue
			}
			r := Relationship{
				Kind:     "ManyToOne",
				Src:      cc,
				Dst:      endpoint,
				SrcField: EnsureUnique(FieldName(endpoint), entityFields[cc]),
				DstField: EnsureUnique(pluralField(FieldName(cc)), entityFields[endpoint]),
			}
			m.Relationships = append(m.Relationships, r)
			rep.Relationships = append(rep.Relationships, renderRelationship(r))
			rep.Warnings = append(rep.Warnings, fmt.Sprintf(
				"association class %q linked to association end %q (%s)", cc, endpoint, renderRelationship(r)))
		}
	}

	return m, rep
}

// relationshipFields assigns the per-end field names for one relationship.
// Collection sides use the plural form (see pluralField) so the pinned
// generator never auto-derives a colliding inverse field; FK sides use the
// singular FieldName of the referenced entity. Names are deduplicated per
// entity.
func relationshipFields(card, src, dst string, entityFields map[string]map[string]struct{}) Relationship {
	r := Relationship{Kind: card, Src: src, Dst: dst}
	srcBase := FieldName(dst)
	dstBase := FieldName(src)
	switch card {
	case "OneToMany":
		// The source entity holds the collection of the destination entity.
		r.SrcField = pluralField(srcBase)
		r.DstField = dstBase
	case "ManyToOne":
		// The source entity holds the FK to the destination entity, and the
		// destination exposes the back-reference collection.
		r.SrcField = srcBase
		r.DstField = pluralField(dstBase)
	case "ManyToMany":
		r.SrcField = pluralField(srcBase)
		r.DstField = pluralField(dstBase)
	default: // OneToOne: both sides are single references
		r.SrcField = srcBase
		r.DstField = dstBase
	}
	r.SrcField = EnsureUnique(r.SrcField, entityFields[src])
	r.DstField = EnsureUnique(r.DstField, entityFields[dst])
	return r
}

// requiredEnd derives Relationship.Required from the multiplicity of the
// REFERENCED end, i.e. the end the FK-holding side points at: the FK is
// required whenever that end's multiplicity has a lower bound >= 1
// (Order * -- 1 Customer: every Order must reference a Customer). ManyToOne/OneToOne read the
// target multiplicity (the FK, declared on Src, references Dst); OneToMany
// reads the source multiplicity (the FK, declared on Dst, references Src).
// ManyToMany has no scalar FK column, so it is never required.
func requiredEnd(card string, srcMultiplicity, dstMultiplicity *string) bool {
	switch card {
	case "ManyToOne", "OneToOne":
		return isRequiredEnd(dstMultiplicity)
	case "OneToMany":
		return isRequiredEnd(srcMultiplicity)
	default: // ManyToMany
		return false
	}
}

// renderRelationship renders one body line of a JDL relationship block. The
// cardinality keyword belongs only to the block header ("relationship X {");
// the grammar rejects it again inside the body (MismatchedTokenException).
// When Required is set, `required` is appended inside the braces of the
// FK-holding field only: ManyToOne/OneToOne hold the FK on Src, OneToMany
// holds it on Dst.
func renderRelationship(r Relationship) string {
	srcField, dstField := r.SrcField, r.DstField
	if r.Required {
		switch r.Kind {
		case "ManyToOne", "OneToOne":
			srcField += " required"
		case "OneToMany":
			dstField += " required"
		}
	}
	return fmt.Sprintf("%s{%s} to %s{%s}", r.Src, srcField, r.Dst, dstField)
}

func danglingID(rel domain.Relationship, srcOK bool) string {
	if !srcOK {
		return rel.SourceID
	}
	return rel.TargetID
}

func skipDetail(relType, src, dst string) string {
	switch relType {
	case "generalization", "realization":
		return fmt.Sprintf("%s from %q to %q (warn-and-skip; remodel inheritance in JHipster)", relType, src, dst)
	default:
		return fmt.Sprintf("%s from %q to %q (warn-and-skip; dependencies are not expressed in JDL)", relType, src, dst)
	}
}

func skipAdvice(relType string) string {
	switch relType {
	case "generalization", "realization":
		return "inheritance is remodeled in JHipster, not generated"
	default:
		return "dependencies are not expressed in JDL"
	}
}

func layoutDetail(class domain.UmlClass) string {
	if class.Width != nil {
		return fmt.Sprintf("x=%d y=%d width=%d (canvas coordinates do not exist in JDL)",
			class.X, class.Y, *class.Width)
	}
	return fmt.Sprintf("x=%d y=%d (canvas coordinates do not exist in JDL)", class.X, class.Y)
}

// attachedRelationshipRef renders the attachment target for the
// association-class warning, or "(none)" when no relationship is attached.
func attachedRelationshipRef(class domain.UmlClass) string {
	if class.AttachedRelationshipID != nil && strings.TrimSpace(*class.AttachedRelationshipID) != "" {
		return *class.AttachedRelationshipID
	}
	return "(none)"
}

// renderJDL assembles the final model.jdl. Sections follow input order and the
// file always ends with a single newline.
func renderJDL(m Model) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// Generated from UML diagram %q by ai-uml-architect (GEN-02).\n", m.DiagramName)
	b.WriteString("// Scaffold with: jhipster jdl model.jdl (requires JHipster installed).\n")
	for _, e := range m.Entities {
		b.WriteString("\nentity " + e.Name + " {\n")
		for _, f := range e.Fields {
			b.WriteString("  " + f.Name + " " + f.Type + "\n")
		}
		b.WriteString("}\n")
	}
	for _, r := range m.Relationships {
		b.WriteString("\nrelationship " + r.Kind + " {\n")
		b.WriteString("  " + renderRelationship(r) + "\n")
		b.WriteString("}\n")
	}
	return b.String()
}
