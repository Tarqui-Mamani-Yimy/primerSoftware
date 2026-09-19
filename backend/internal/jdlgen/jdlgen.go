// Package jdlgen converts a UML DiagramDocument into JHipster JDL text plus a
// machine-readable report of every construct that JDL cannot express.
//
// GEN-02 preparation scope: this package is intentionally NOT imported by the
// Go server (main.go, internal/httpapi, internal/service). The Go runtime stays
// untouched; a standalone CLI (gobackend/cmd/jdl-bundle) and a shell wrapper
// (gobackend/tools/prepare-jhipster-bundle.sh) consume this package to prepare
// a downloadable bundle. Running `jhipster jdl` requires a machine with
// JHipster installed (see gobackend/docs/jhipster-generation-runbook.md).
//
// Mapping decisions (also documented in the runbook):
//   - Classes become JDL entities; UML attribute types map through jdlTypeFor
//     (unknown types fall back to String with a warning).
//   - association/aggregation/composition become JDL relationships whose
//     cardinality derives from source/target multiplicities. Aggregation and
//     composition ownership/cascade semantics flatten to plain JDL associations
//     and are reported as a warning.
//   - generalization/realization/dependency use warn-and-skip: they are listed
//     in the report and omitted from the JDL. Inheritance must be remodeled in
//     the generated JHipster project by hand.
//   - A class with an "Enum" stereotype is skipped with a warning because the
//     UML Attribute carries id/name/type/visibility only — there is no value
//     list to emit a JDL enum from.
//   - Methods, member visibility, canvas layout (x/y/width), package names,
//     table bindings, and relationship labels have no JDL equivalent and are
//     recorded in the report as dropped.
package jdlgen

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ai-uml-architect/gobackend/internal/domain"
)

// Dropped records a single UML construct that has no JDL equivalent.
type Dropped struct {
	Kind     string `json:"kind"`
	Location string `json:"location"`
	Detail   string `json:"detail"`
}

// Report is the machine-readable companion of the generated model.jdl. Every
// dropped construct appears in Dropped; Warnings holds human-readable notes
// the user must review before running `jhipster jdl`.
type Report struct {
	DiagramName   string    `json:"diagramName"`
	Entities      []string  `json:"entities"`
	Relationships []string  `json:"relationships"`
	Dropped       []Dropped `json:"dropped"`
	Warnings      []string  `json:"warnings"`
}

// javaReserved is the JLS keyword/literal set plus "_" (reserved since Java 9).
// Comparison is exact: "Class" is legal Java, only "class" is rejected.
var javaReserved = map[string]struct{}{
	"abstract": {}, "assert": {}, "boolean": {}, "break": {}, "byte": {},
	"case": {}, "catch": {}, "char": {}, "class": {}, "const": {},
	"continue": {}, "default": {}, "do": {}, "double": {}, "else": {},
	"enum": {}, "extends": {}, "final": {}, "finally": {}, "float": {},
	"for": {}, "goto": {}, "if": {}, "implements": {}, "import": {},
	"instanceof": {}, "int": {}, "interface": {}, "long": {}, "native": {},
	"new": {}, "package": {}, "private": {}, "protected": {}, "public": {},
	"return": {}, "short": {}, "static": {}, "strictfp": {}, "super": {},
	"switch": {}, "synchronized": {}, "this": {}, "throw": {}, "throws": {},
	"transient": {}, "try": {}, "void": {}, "volatile": {}, "while": {},
	"true": {}, "false": {}, "null": {}, "var": {}, "yield": {},
	"record": {}, "sealed": {}, "permits": {}, "_": {},
}

// SanitizeIdentifier rewrites raw into a legal Java identifier: characters
// outside [A-Za-z0-9_] become "_", a leading digit gains a "_" prefix, and
// Java reserved words gain a "_" suffix. Empty or all-underscore input yields
// "Unnamed". The mapping is deterministic.
func SanitizeIdentifier(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" || strings.Trim(out, "_") == "" {
		return "Unnamed"
	}
	if r, _ := utf8.DecodeRuneInString(out); r >= '0' && r <= '9' {
		out = "_" + out
	}
	if _, reserved := javaReserved[out]; reserved {
		out += "_"
	}
	return out
}

// EntityName sanitizes raw into an UpperCamelCase JHipster entity name.
func EntityName(raw string) string {
	s := SanitizeIdentifier(raw)
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

// FieldName sanitizes raw into a lowerCamelCase JDL field name.
func FieldName(raw string) string {
	s := SanitizeIdentifier(raw)
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[size:]
}

// EnsureUnique returns base when unused, otherwise base_2, base_3, … —
// deterministically. The returned name is recorded in used.
func EnsureUnique(base string, used map[string]struct{}) string {
	if _, taken := used[base]; !taken {
		used[base] = struct{}{}
		return base
	}
	for i := 2; ; i++ {
		candidate := base + "_" + strconv.Itoa(i)
		if _, taken := used[candidate]; !taken {
			used[candidate] = struct{}{}
			return candidate
		}
	}
}

// IsEnumStereotype reports whether a UML stereotype means «Enum».
func IsEnumStereotype(stereotype *string) bool {
	if stereotype == nil {
		return false
	}
	s := strings.TrimSpace(*stereotype)
	s = strings.Trim(s, "«»")
	return strings.EqualFold(s, "enum")
}

// jdlTypeFor maps a UML attribute type to a JDL field type. The boolean is
// false when the UML type is unknown; callers then emit String plus a warning.
func jdlTypeFor(umlType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(umlType)) {
	case "string":
		return "String", true
	case "text":
		return "TextBlob", true
	case "int", "integer", "short", "byte":
		return "Integer", true
	case "long":
		return "Long", true
	case "float":
		return "Float", true
	case "double":
		return "Double", true
	case "bigdecimal", "decimal", "money":
		return "BigDecimal", true
	case "boolean", "bool":
		return "Boolean", true
	case "date", "localdate":
		return "LocalDate", true
	case "datetime", "timestamp", "zoneddatetime":
		return "ZonedDateTime", true
	case "instant":
		return "Instant", true
	case "uuid":
		return "UUID", true
	case "blob", "bytes", "byte[]":
		return "Blob", true
	default:
		return "String", false
	}
}

// isManySide interprets a UML multiplicity: anything containing "*" ("*",
// "0..*", "1..*") or a bound above 1 ("2", "1..5") is many; missing, empty,
// "1", "0..1", and "1..1" are one. A missing multiplicity defaults to one so
// the emitted JDL cardinality is always explicit and reviewable.
func isManySide(multiplicity *string) bool {
	if multiplicity == nil {
		return false
	}
	s := strings.TrimSpace(*multiplicity)
	if s == "" {
		return false
	}
	if strings.Contains(s, "*") {
		return true
	}
	if strings.EqualFold(s, "many") || strings.EqualFold(s, "n") {
		return true
	}
	parts := strings.Split(s, "..")
	last := strings.TrimSpace(parts[len(parts)-1])
	if n, err := strconv.Atoi(last); err == nil {
		return n > 1
	}
	return false
}

// cardinality derives the JDL relationship kind from the many-ness of each end.
func cardinality(srcMany, dstMany bool) string {
	switch {
	case srcMany && dstMany:
		return "ManyToMany"
	case srcMany:
		return "ManyToOne"
	case dstMany:
		return "OneToMany"
	default:
		return "OneToOne"
	}
}

type entityDef struct {
	name   string
	fields []fieldDef
}

type fieldDef struct {
	name string
	typ  string
}

// Export converts doc into deterministic JDL text plus a report of every
// dropped construct. Output order follows the document: classes and
// relationships appear in input order, so repeated runs are byte-identical.
func Export(doc domain.DiagramDocument) (string, Report) {
	rep := Report{
		DiagramName:   doc.Name,
		Entities:      []string{},
		Relationships: []string{},
		Dropped:       []Dropped{},
		Warnings:      []string{},
	}

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
		if final != base {
			reason = "name collision after sanitization"
		}
		if final != class.Name {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf(
				"class %q renamed to entity %q (%s)", class.Name, final, reason))
		}
		idToEntity[class.ID] = final
		rep.Entities = append(rep.Entities, final)
	}

	// Phase 2: emit entity blocks with attribute fields.
	entities := make([]entityDef, 0, len(rep.Entities))
	for _, class := range doc.Classes {
		if skipped[class.ID] {
			continue
		}
		entity := entityDef{name: idToEntity[class.ID]}
		usedFields := map[string]struct{}{}
		rep.Dropped = append(rep.Dropped, Dropped{
			Kind:     "layout",
			Location: "class " + entity.name,
			Detail:   layoutDetail(class),
		})
		if class.PackageName != nil && strings.TrimSpace(*class.PackageName) != "" {
			rep.Dropped = append(rep.Dropped, Dropped{
				Kind:     "package",
				Location: "class " + entity.name,
				Detail:   fmt.Sprintf("package %q (JHipster derives packages from baseName)", *class.PackageName),
			})
		}
		if class.TableBinding != nil && strings.TrimSpace(*class.TableBinding) != "" {
			rep.Dropped = append(rep.Dropped, Dropped{
				Kind:     "tableBinding",
				Location: "class " + entity.name,
				Detail:   fmt.Sprintf("table %q (JHipster generates table names)", *class.TableBinding),
			})
		}
		for _, attr := range class.Attributes {
			base := FieldName(attr.Name)
			final := EnsureUnique(base, usedFields)
			if final != attr.Name {
				reason := "Java identifier sanitization"
				if final != base {
					reason = "name collision after sanitization"
				}
				rep.Warnings = append(rep.Warnings, fmt.Sprintf(
					"class %q attribute %q renamed to field %q (%s)", entity.name, attr.Name, final, reason))
			}
			jdlType, known := jdlTypeFor(attr.Type)
			if !known {
				rep.Warnings = append(rep.Warnings, fmt.Sprintf(
					"class %q field %q: unknown UML type %q; emitted as String", entity.name, final, attr.Type))
			}
			entity.fields = append(entity.fields, fieldDef{name: final, typ: jdlType})
			if attr.Visibility != nil && strings.TrimSpace(*attr.Visibility) != "" {
				rep.Dropped = append(rep.Dropped, Dropped{
					Kind:     "visibility",
					Location: "class " + entity.name + " attribute " + attr.Name,
					Detail:   fmt.Sprintf("visibility %q (not expressed in JDL)", *attr.Visibility),
				})
			}
		}
		for _, method := range class.Methods {
			rep.Dropped = append(rep.Dropped, Dropped{
				Kind:     "method",
				Location: "class " + entity.name,
				Detail:   fmt.Sprintf("%s(): %s (behavior is not expressed in JDL)", method.Name, method.ReturnType),
			})
			if method.Visibility != nil && strings.TrimSpace(*method.Visibility) != "" {
				rep.Dropped = append(rep.Dropped, Dropped{
					Kind:     "visibility",
					Location: "class " + entity.name + " method " + method.Name,
					Detail:   fmt.Sprintf("visibility %q (not expressed in JDL)", *method.Visibility),
				})
			}
		}
		entities = append(entities, entity)
	}

	// Phase 3: emit relationships in document order.
	entityFields := map[string]map[string]struct{}{}
	for _, e := range entities {
		used := map[string]struct{}{}
		for _, f := range e.fields {
			used[f.name] = struct{}{}
		}
		entityFields[e.name] = used
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
			onSrc := EnsureUnique(FieldName(dst), entityFields[src])
			onDst := EnsureUnique(FieldName(src), entityFields[dst])
			rep.Relationships = append(rep.Relationships,
				fmt.Sprintf("%s %s{%s} to %s{%s}", card, src, onSrc, dst, onDst))
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

	return renderJDL(doc.Name, entities, rep.Relationships), rep
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

// renderJDL assembles the final model.jdl. Sections follow input order and the
// file always ends with a single newline.
func renderJDL(diagramName string, entities []entityDef, relationships []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// Generated from UML diagram %q by ai-uml-architect (GEN-02).\n", diagramName)
	b.WriteString("// Scaffold with: jhipster jdl model.jdl (requires JHipster installed).\n")
	for _, e := range entities {
		b.WriteString("\nentity " + e.name + " {\n")
		for _, f := range e.fields {
			b.WriteString("  " + f.name + " " + f.typ + "\n")
		}
		b.WriteString("}\n")
	}
	for _, r := range relationships {
		parts := strings.SplitN(r, " ", 2)
		b.WriteString("\nrelationship " + parts[0] + " {\n")
		b.WriteString("  " + parts[1] + "\n")
		b.WriteString("}\n")
	}
	return b.String()
}
