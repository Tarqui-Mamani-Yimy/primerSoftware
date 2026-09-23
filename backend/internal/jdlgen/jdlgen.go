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

// Options pins the JHipster application scaffold emitted by ExportArtifact.
// The zero value is not valid; call DefaultOptions and override only the
// fields the client provided. Names are validated strictly (no silent
// mutation): a hard, early error is clearer than a scaffold that fails
// halfway through generation on a machine that cannot be debugged.
type Options struct {
	BaseName           string
	PackageName        string
	BuildTool          string // maven | gradle
	AuthenticationType string // jwt
}

// DefaultOptions is the documented neutral application identity: it carries
// no client domain (no invented packages or business entities), just a sane
// scaffold the client may override per request.
func DefaultOptions() Options {
	return Options{
		BaseName:           "UmlArchitect",
		PackageName:        "com.umlarchitect",
		BuildTool:          "maven",
		AuthenticationType: "jwt",
	}
}

// ValidateOptions returns clear, actionable error strings for invalid
// application configuration. An empty slice means valid. Base names follow
// JHipster's rule (alphanumeric, not ending in the suffix JHipster appends to
// the application class); package segments must be legal Java identifiers
// that are not reserved words.
func ValidateOptions(o Options) []string {
	var errs []string
	base := strings.TrimSpace(o.BaseName)
	if base == "" {
		errs = append(errs, "baseName is required")
	} else {
		for _, r := range base {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
				errs = append(errs, "baseName must be alphanumeric (got "+strconv.Quote(o.BaseName)+")")
				break
			}
		}
		if len(errs) == 0 && base[0] >= '0' && base[0] <= '9' {
			errs = append(errs, "baseName must not start with a digit (got "+strconv.Quote(o.BaseName)+")")
		}
		if len(errs) == 0 && strings.HasSuffix(base, "App") {
			errs = append(errs, "baseName must not end with \"App\" (JHipster appends it to the application class; got "+strconv.Quote(o.BaseName)+")")
		}
	}
	pkg := strings.TrimSpace(o.PackageName)
	switch {
	case pkg == "":
		errs = append(errs, "packageName is required")
	case strings.HasPrefix(pkg, ".") || strings.HasSuffix(pkg, ".") || strings.Contains(pkg, ".."):
		errs = append(errs, "packageName must be a dot-separated list of Java identifiers (got "+strconv.Quote(o.PackageName)+")")
	default:
		for _, seg := range strings.Split(pkg, ".") {
			if !validJavaIdentifier(seg) {
				errs = append(errs, "packageName segment "+strconv.Quote(seg)+" is not a legal Java identifier (got "+strconv.Quote(o.PackageName)+")")
				break
			}
		}
	}
	switch o.BuildTool {
	case "maven", "gradle":
	default:
		errs = append(errs, "buildTool must be \"maven\" or \"gradle\" (got "+strconv.Quote(o.BuildTool)+")")
	}
	switch o.AuthenticationType {
	case "jwt":
	default:
		errs = append(errs, "authenticationType must be \"jwt\" (got "+strconv.Quote(o.AuthenticationType)+")")
	}
	return errs
}

// validJavaIdentifier reports whether s is a legal, non-reserved Java
// identifier.
func validJavaIdentifier(s string) bool {
	if s == "" {
		return false
	}
	if r, _ := utf8.DecodeRuneInString(s); !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || r == '$') {
		return false
	}
	for _, r := range s[1:] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '$') {
			return false
		}
	}
	_, reserved := javaReserved[s]
	return !reserved
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

// jdlSafeIdentifier rewrites raw into a bare JDL-safe name stem: characters
// outside [A-Za-z0-9] are dropped, leading digits are stripped up to the first
// letter, and empty or digit-only input yields "Unnamed". JHipster 9.4.0's
// JDL validator requires entity names to match /^[A-Z][A-Za-z0-9]*$/ and field
// names /^[A-Za-z][A-Za-z0-9]*$/, so no underscore may remain in the final
// name; FieldName/EntityName apply camel case and the reserved-word checks on
// top of this stem. The mapping is deterministic.
func jdlSafeIdentifier(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	hasLetter := false
	for _, r := range out {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return "Unnamed"
	}
	cut := 0
	for cut < len(out) && out[cut] >= '0' && out[cut] <= '9' {
		cut++
	}
	return out[cut:]
}

// FieldName sanitizes raw into a lowerCamelCase JDL field name. Names that
// collide exactly with a JDL lexer keyword (for example "required", "unique",
// "baseName", "readOnly") or a Java reserved word gain a "2" suffix: the
// pinned generator's chevrotain lexer tokenizes those words as grammar tokens
// instead of NAME tokens, so `required String` fails with
// MismatchedTokenException, the JDL validator rejects underscores in field
// names, and a Java keyword like "class" would break the generated entity.
// The mapping is deterministic; see jdlReservedWords for the authoritative set.
func FieldName(raw string) string {
	s := jdlSafeIdentifier(raw)
	r, size := utf8.DecodeRuneInString(s)
	out := string(unicode.ToLower(r)) + s[size:]
	if _, reserved := javaReserved[out]; reserved {
		out += "2"
	}
	if isJDLReservedWord(out) {
		out += "2"
	}
	return out
}

// EntityName sanitizes raw into an UpperCamelCase JHipster entity name. The
// same JDL lexer guard as FieldName applies: a class literally named
// "OneToOne" would lex as the relationship-type token and break the parser,
// and the validator rejects underscores in entity names. UpperCamelCase can
// never equal a lowercase Java keyword, so only the JDL guard applies here.
func EntityName(raw string) string {
	s := jdlSafeIdentifier(raw)
	r, size := utf8.DecodeRuneInString(s)
	out := string(unicode.ToUpper(r)) + s[size:]
	if isJDLReservedWord(out) {
		out += "2"
	}
	return out
}

// jdlReservedWords is the authoritative set of words the pinned JDL lexer
// tokenizes as something other than NAME. Extracted from generator-jhipster
// 9.4.0: dist/lib/jdl/core/parsing/lexer/{lexer,option-tokens,
// relationship-type-tokens,validation-tokens,minmax-tokens}.js and
// dist/lib/jdl/core/built-in-options/tokens/{application-tokens,
// deployment-tokens}.js. Comparison is exact (the lexer patterns are
// case-sensitive): "required" collides, "Required" does not.
var jdlReservedWords = map[string]struct{}{
	// Core language keywords (lexer.js)
	"config": {}, "entities": {}, "application": {}, "deployment": {},
	"serviceDiscoveryType": {}, "true": {}, "false": {}, "entity": {},
	"enum": {}, "relationship": {}, "builtInEntity": {}, "to": {},
	// Option keywords (option-tokens.js)
	"with": {}, "except": {}, "use": {}, "for": {}, "clientRootFolder": {},
	"noFluentMethod": {}, "readOnly": {}, "embedded": {}, "dto": {},
	"paginate": {}, "service": {}, "microservice": {}, "search": {},
	"angularSuffix": {}, "filter": {},
	// Validation keywords (validation-tokens.js, minmax-tokens.js)
	"required": {}, "unique": {}, "pattern": {}, "minlength": {},
	"maxlength": {}, "minbytes": {}, "maxbytes": {}, "min": {}, "max": {},
	// Application configuration keys (application-tokens.js)
	"baseName": {}, "blueprints": {}, "blueprint": {}, "creationTimestamp": {},
	"gatewayServerPort": {}, "packageName": {}, "authenticationType": {},
	"cacheProvider": {}, "enableHibernateCache": {}, "websocket": {},
	"databaseType": {}, "devDatabaseType": {}, "prodDatabaseType": {},
	"buildTool": {}, "searchEngine": {}, "enableTranslation": {},
	"applicationType": {}, "testFrameworks": {}, "languages": {},
	"serverPort": {}, "jhiPrefix": {}, "jwtSecretKey": {}, "jhipsterVersion": {},
	"clientFramework": {}, "clientThemeVariant": {}, "clientTheme": {},
	"withAdminUi": {}, "nativeLanguage": {}, "frontendBuilder": {},
	"skipUserManagement": {}, "enableSwaggerCodegen": {}, "reactive": {},
	"entitySuffix": {}, "dtoSuffix": {}, "skipClient": {}, "skipServer": {},
	"rememberMeKey": {}, "enableGradleDevelocity": {}, "gradleDevelocityHost": {},
	"microfrontends": {}, "microfrontend": {}, "nodePackageManager": {},
	// Deployment configuration keys (deployment-tokens.js)
	"appsFolders": {}, "clusteredDbApps": {}, "deploymentType": {},
	"directoryPath": {}, "dockerPushCommand": {}, "dockerRepositoryName": {},
	"gatewayType": {}, "ingressDomain": {}, "ingressType": {}, "istio": {},
	"kubernetesNamespace": {}, "kubernetesServiceType": {},
	"kubernetesStorageClassName": {}, "kubernetesUseDynamicStorage": {},
	"monitoring": {}, "registryReplicas": {}, "storageType": {},
	// Relationship type keywords (relationship-type-tokens.js)
	"OneToOne": {}, "OneToMany": {}, "ManyToOne": {}, "ManyToMany": {},
}

func isJDLReservedWord(s string) bool {
	_, reserved := jdlReservedWords[s]
	return reserved
}

// jhipsterBuiltInEntities lists JHipster's own internal entities that a
// same-named JDL entity gets silently MERGED into rather than creating: see
// generator-jhipster 9.4.0's
// generators/base-application/generators/bootstrap/generator.js:101-102 —
// `entityName === 'User'` or `'Authority'` (exact, case-sensitive) routes to
// createUserEntity/createAuthorityEntity instead of the JDL definition, and
// only the "id" field plus relationships declared on the OTHER side survive
// (generators/base-application/internal/utils.js:38-70, console warnings
// only). Our own sqlgen would also emit a table for the colliding entity
// that collides with the internal jhi_user/jhi_authority tables (USER and
// AUTHORITY are both PostgreSQL-reserved, see sqlgen/reserved.go), so
// BuildModel renames the class outright instead of letting either of these
// silent drops happen.
var jhipsterBuiltInEntities = map[string]struct{}{
	"User": {}, "Authority": {},
}

// isJHipsterBuiltInEntity reports whether name (an already-sanitized
// EntityName result) collides with one of JHipster's built-in entities. The
// comparison is exact/case-sensitive, matching the generator's own check —
// EntityName always uppercases the first letter, so this only ever matches
// the sanitized form, not arbitrary raw casing.
func isJHipsterBuiltInEntity(name string) bool {
	_, builtIn := jhipsterBuiltInEntities[name]
	return builtIn
}

// EnsureUnique returns base when unused, otherwise base2, base3, … —
// deterministically. The returned name is recorded in used.
func EnsureUnique(base string, used map[string]struct{}) string {
	if _, taken := used[base]; !taken {
		used[base] = struct{}{}
		return base
	}
	for i := 2; ; i++ {
		candidate := base + strconv.Itoa(i)
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
//
// Alongside the English/JDL vocabulary, it accepts the small Spanish voice
// vocabulary VOICE-03 defines (comparison is case-insensitive; both accented
// and unaccented spellings are accepted): entero, texto/cadena, decimal,
// fecha, "fecha hora"/"fecha y hora", booleano/lógico, largo, flotante,
// doble, uuid.
func jdlTypeFor(umlType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(umlType)) {
	case "string", "texto", "cadena":
		return "String", true
	case "text":
		return "TextBlob", true
	case "int", "integer", "short", "byte", "entero":
		return "Integer", true
	case "long", "largo":
		return "Long", true
	case "float", "flotante":
		return "Float", true
	case "double", "doble":
		return "Double", true
	case "bigdecimal", "decimal", "money":
		return "BigDecimal", true
	case "boolean", "bool", "booleano", "lógico", "logico":
		return "Boolean", true
	case "date", "localdate", "fecha":
		return "LocalDate", true
	case "datetime", "timestamp", "zoneddatetime", "fecha hora", "fecha y hora":
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

// isRequiredEnd interprets a UML multiplicity's lower bound: nil, empty, and
// "*" are optional (false); an explicit lower bound of 1 or more ("1",
// "1..*", "2..5") is required (true); a lower bound of 0 ("0", "0..1",
// "0..*") is optional. The lower bound is the part before ".." when present,
// otherwise the whole value.
func isRequiredEnd(multiplicity *string) bool {
	if multiplicity == nil {
		return false
	}
	s := strings.TrimSpace(*multiplicity)
	if s == "" {
		return false
	}
	lower := strings.TrimSpace(strings.SplitN(s, "..", 2)[0])
	if lower == "*" {
		return false
	}
	n, err := strconv.Atoi(lower)
	if err != nil {
		return false
	}
	return n >= 1
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

// Export converts doc into deterministic JDL text plus a report of every
// dropped construct. Output order follows the document: classes and
// relationships appear in input order, so repeated runs are byte-identical.
func Export(doc domain.DiagramDocument) (string, Report) {
	m, rep := BuildModel(doc)
	return renderJDL(m), rep
}

// renderApplicationBlock emits the JDL application block that scaffolds the
// backend-only monolith app with the pinned provider, database, pinned build
// tool, and JWT authentication. The entities list must name every entity the
// JDL declares: JHipster 9 only imports entities listed inside the application
// block ("entities Alpha, Beta"), so keeping it in the block means one
// `jhipster jdl model.jdl` call scaffolds the app AND its entities.
//
// The key set is intentionally stable and minimal: every key here is accepted
// by the pinned generator-jhipster release this package targets. Values JHipster
// can infer safely are left to its defaults. baseName and packageName are
// unquoted in the JDL grammar (quoted values are a parse error in v9).
func renderApplicationBlock(o Options, entities []string) string {
	var b strings.Builder
	b.WriteString("application {\n  config {\n")
	fmt.Fprintf(&b, "    baseName %s\n", o.BaseName)
	b.WriteString("    applicationType monolith\n")
	fmt.Fprintf(&b, "    packageName %s\n", o.PackageName)
	fmt.Fprintf(&b, "    authenticationType %s\n", o.AuthenticationType)
	fmt.Fprintf(&b, "    buildTool %s\n", o.BuildTool)
	b.WriteString("    databaseType sql\n")
	b.WriteString("    prodDatabaseType postgresql\n")
	b.WriteString("    devDatabaseType postgresql\n")
	b.WriteString("    skipClient true\n")
	b.WriteString("  }\n")
	if len(entities) > 0 {
		fmt.Fprintf(&b, "  entities %s\n", strings.Join(entities, ", "))
		// Without these two options JHipster generates bare spring-data
		// repositories; with them every entity gets a real Service/ServiceImpl,
		// a MapStruct DTO, and a REST resource with CRUD endpoints, which is
		// what the generated artifact promises (GEN-02 contract).
		b.WriteString("  service * with serviceImpl\n")
		b.WriteString("  dto * with mapstruct\n")
		// GBU-01: without `paginate` the generated *Resource#getAll ignores
		// ?page=&size= and never sends X-Total-Count/Link, forcing every
		// consumer (Postman, a separate frontend) to fetch entire collections.
		// PAGINATE is a binary option (`paginate * with pagination`, see the
		// pinned generator's dist/lib/jdl/core/parsing/lexer/option-tokens.js);
		// FILTER is unary (`filter *`, no `with` clause) and generates the
		// JPA Specification-based *QueryService/*Criteria classes consumed
		// through query params like ?name.contains=. Both are valid inside the
		// application block at the same grammar level as service/dto (see
		// applicationSubDeclaration in dist/lib/jdl/core/parsing/jdl-parser.js).
		b.WriteString("  paginate * with pagination\n")
		b.WriteString("  filter *\n")
	}
	b.WriteString("}\n")
	return b.String()
}

// ExportArtifactModel converts doc into the standalone JHipster backend
// scaffold AND the structured Model the database packager consumes, so the SQL
// init script stays consistent with the JPA schema JHipster will generate.
// Invalid options return a clear error and no JDL.
func ExportArtifactModel(doc domain.DiagramDocument, o Options) (string, Model, Report, error) {
	if errs := ValidateOptions(o); len(errs) > 0 {
		return "", Model{}, Report{}, fmt.Errorf("invalid JHipster application options: %s", strings.Join(errs, "; "))
	}
	m, rep := BuildModel(doc)
	return renderApplicationBlock(o, rep.Entities) + "\n" + renderJDL(m), m, rep, nil
}

// ExportArtifact converts doc into a standalone JHipster backend scaffold:
// the application block (options-pinned, backend-only, with the entity list
// and the service/DTO options) followed by the same entities and
// relationships Export produces. The Report is returned so the caller can
// surface warnings and skipped constructs in the artifact manifest.
// Invalid options return a clear error and no JDL.
func ExportArtifact(doc domain.DiagramDocument, o Options) (string, Report, error) {
	jdl, _, rep, err := ExportArtifactModel(doc, o)
	return jdl, rep, err
}
