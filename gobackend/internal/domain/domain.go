// Package domain holds JSON domain types for the Go backend skeleton (GOBE-01).
//
// The structs mirror the public REST contract of the existing Java backend
// without changing it: field names and JSON tags stay byte-compatible with
// the Jackson-serialized Java records.
package domain

// ErrorEnvelope is the uniform error body: {"message": "..."}.
type ErrorEnvelope struct {
	Message string `json:"message"`
}

// LoginRequest mirrors POST /api/v1/auth/login input.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse mirrors the Java AuthService.LoginResponse record.
type LoginResponse struct {
	AccessToken string `json:"accessToken"`
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

// ProjectResponse mirrors ProjectService.ProjectResponse. projects.description
// is nullable in V1, so a nil description must serialize as "description": null.
type ProjectResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	Role         string  `json:"role"`
	DiagramCount int     `json:"diagramCount"`
}

// DiagramSummary mirrors DiagramService.DiagramSummary.
// The exact timestamp wire format is pinned in GOBE-02 against the live backend.
type DiagramSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updatedAt"`
}

// DiagramVersion mirrors DiagramService.DiagramVersionResponse.
type DiagramVersion struct {
	ID            string          `json:"id"`
	VersionNumber int             `json:"versionNumber"`
	CreatedAt     string          `json:"createdAt"`
	Document      DiagramDocument `json:"document"`
}

// DiagramDocument mirrors the Java DiagramDocument record. The Java UUID id has
// no @NotNull and Jackson always serializes it, so a missing id must emit
// "id": null rather than omit the key (frontend documents declare id optional).
type DiagramDocument struct {
	SchemaVersion int            `json:"schemaVersion"`
	ID            *string        `json:"id"`
	Name          string         `json:"name"`
	Classes       []UmlClass     `json:"classes"`
	Relationships []Relationship `json:"relationships"`
}

// UmlClass mirrors DiagramDocument.UmlClass. Nullable Java fields use pointers
// without omitempty so they serialize as null, matching Jackson's default.
type UmlClass struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Stereotype   *string     `json:"stereotype"`
	PackageName  *string     `json:"package"`
	TableBinding *string     `json:"tableBinding"`
	X            int         `json:"x"`
	Y            int         `json:"y"`
	Width        *int        `json:"width"`
	Attributes   []Attribute `json:"attributes"`
	Methods      []Method    `json:"methods"`
}

// Attribute mirrors DiagramDocument.Attribute.
type Attribute struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Visibility *string `json:"visibility"`
}

// Method mirrors DiagramDocument.Method.
type Method struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	ReturnType string  `json:"returnType"`
	Visibility *string `json:"visibility"`
}

// Relationship mirrors DiagramDocument.Relationship.
type Relationship struct {
	ID                 string  `json:"id"`
	SourceID           string  `json:"sourceId"`
	TargetID           string  `json:"targetId"`
	Type               string  `json:"type"`
	SourceMultiplicity *string `json:"sourceMultiplicity"`
	TargetMultiplicity *string `json:"targetMultiplicity"`
	Label              *string `json:"label"`
}

// RelationshipTypes is the closed set of supported relationship types,
// mirroring DiagramDocumentValidator.RELATIONSHIP_TYPES.
var RelationshipTypes = []string{
	"association",
	"aggregation",
	"composition",
	"generalization",
	"realization",
	"dependency",
}

// IsValidRelationshipType reports whether t is a supported relationship type.
func IsValidRelationshipType(t string) bool {
	for _, valid := range RelationshipTypes {
		if t == valid {
			return true
		}
	}
	return false
}
