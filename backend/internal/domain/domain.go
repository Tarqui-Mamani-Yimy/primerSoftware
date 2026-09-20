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

// CreateProjectRequest mirrors POST /api/v1/projects input. description is
// optional: projects.description is nullable in V1.
type CreateProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// JoinProjectRequest mirrors POST /api/v1/projects/join input.
type JoinProjectRequest struct {
	AccessCode string `json:"accessCode"`
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

// ProjectCreatedResponse is the 201 body for POST /api/v1/projects: the list
// item shape plus the generated access code the owner shares with the class.
type ProjectCreatedResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	Role         string  `json:"role"`
	DiagramCount int     `json:"diagramCount"`
	AccessCode   string  `json:"accessCode"`
}

// DiagramSummary mirrors DiagramService.DiagramSummary.
// The exact timestamp wire format is pinned in GOBE-02 against the live
// backend. Version is the highest current checkpoint number (0 when none),
// alongside the diagram's last autosave timestamp. ReviewNumber is the
// monotonic per-diagram work counter echoed back so clients know which
// baseline to send on the next write.
type DiagramSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	UpdatedAt    string `json:"updatedAt"`
	Version      int    `json:"version"`
	ReviewNumber int64  `json:"reviewNumber"`
}

// DiagramVersion mirrors DiagramService.DiagramVersionResponse plus the
// actor, the optional checkpoint message, and the per-diagram work counter
// at the moment of the explicit save so version-history consumers can
// correlate the entry with concurrent autosaves.
type DiagramVersion struct {
	ID            string          `json:"id"`
	VersionNumber int             `json:"versionNumber"`
	ReviewNumber  int64           `json:"reviewNumber"`
	CreatedAt     string          `json:"createdAt"`
	CreatedBy     string          `json:"createdBy"`
	Message       *string         `json:"message"`
	Document      DiagramDocument `json:"document"`
}

// DiagramDocument mirrors the Java DiagramDocument record. The Java UUID id has
// no @NotNull and Jackson always serializes it, so a missing id must emit
// "id": null rather than omit the key (frontend documents declare id optional).
//
// Version is the explicit-checkpoint counter: it equals the highest
// version_number written to diagram_versions for this diagram, OR 0 when the
// diagram has never been checkpointed. Autosave (PUT) keeps Version stable;
// only the explicit /checkpoints POST advances it. Clients send Version back
// in the request body (or via the If-Match header) so the server can detect a
// concurrent checkpoint and reject stale writes with 409.
//
// ReviewNumber is a separate, monotonic per-diagram work counter that bumps
// on every successful save or checkpoint write. Clients send the previous
// value on the next PUT or POST /checkpoints (the JSON body's reviewNumber
// or the X-Diagram-Review header). A mismatch returns 409, forbidding
// silent overwrites even when an explicit checkpoint never happened.
type DiagramDocument struct {
	SchemaVersion int            `json:"schemaVersion"`
	ID            *string        `json:"id"`
	Version       int            `json:"version"`
	ReviewNumber  int64          `json:"reviewNumber"`
	Name          string         `json:"name"`
	Classes       []UmlClass     `json:"classes"`
	Relationships []Relationship `json:"relationships"`
}

// CheckpointRequest is the POST body for /checkpoints: the baseline version
// is required to detect a concurrent checkpoint; the message is optional and
// is stored on the version row to preserve authorship of the explicit save.
type CheckpointRequest struct {
	Version       *int    `json:"version"`
	Message       *string `json:"message"`
}

// UmlClass mirrors DiagramDocument.UmlClass. Nullable Java fields use pointers
// without omitempty so they serialize as null, matching Jackson's default.
// IsAssociationClass and AttachedRelationshipID are the association-class
// extension: a class may be attached to one existing relationship so the
// diagram editor can render it at the relationship's midpoint.
type UmlClass struct {
	ID                     string      `json:"id"`
	Name                   string      `json:"name"`
	Stereotype             *string     `json:"stereotype"`
	PackageName            *string     `json:"package"`
	TableBinding           *string     `json:"tableBinding"`
	X                      int         `json:"x"`
	Y                      int         `json:"y"`
	Width                  *int        `json:"width"`
	Attributes             []Attribute `json:"attributes"`
	Methods                []Method    `json:"methods"`
	IsAssociationClass     *bool       `json:"isAssociationClass"`
	AttachedRelationshipID *string     `json:"attachedRelationshipId"`
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

// ----- Realtime collaboration envelopes -------------------------------------
//
// The realtime hub speaks JSON over WebSocket using a typed `type` field so
// the same envelope schema can be extended without breaking existing
// clients. Every envelope sent by the server carries an ISO-8601 UTC
// `serverTime` so connected members can correct for clock drift and the
// presence UI can fade old peers. The contract documented in the
// "realtime-diagram-collaboration" tracker is the source of truth.

// EnvelopeType is the closed set of realtime message kinds.
type EnvelopeType string

const (
	EnvelopeSnapshot      EnvelopeType = "snapshot"
	EnvelopePresenceJoin  EnvelopeType = "presence.join"
	EnvelopePresenceLeave EnvelopeType = "presence.leave"
	EnvelopePresenceHeart EnvelopeType = "presence.heartbeat"
	EnvelopeDiagramChange EnvelopeType = "diagram.changed"
	EnvelopeError         EnvelopeType = "error"
)

// Envelope is the wrapper every realtime message uses. The Type field is
// always present and matches one of the EnvelopeType values; the Payload is
// the typed body (marshalled as a JSON object).
type Envelope struct {
	Type       EnvelopeType `json:"type"`
	ServerTime string       `json:"serverTime"`
	Payload    any          `json:"payload,omitempty"`
}

// PresenceMember is what a dashboard renders in the "online collaborators"
// list. UserID is the auth identity, DisplayName is the value the realtime
// envelope received at hello time and LastSeen is the partner of the
// heartbeat: a peer is considered online as long as their last_heartbeat is
// within freshnessWindow (default 60s).
type PresenceMember struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	LastSeen    string `json:"lastSeen"`
}

// PresenceSnapshot is the first payload the server sends to a freshly
// connected member. It carries the live membership roster, the working
// document mirror, and the latest version + review counters so the client
// can reconcile without an extra GET.
type PresenceSnapshot struct {
	ProjectID    string          `json:"projectId"`
	DiagramID    string          `json:"diagramId"`
	Members      []PresenceMember `json:"members"`
	Document     DiagramDocument `json:"document"`
	Version      int             `json:"version"`
	ReviewNumber int64           `json:"reviewNumber"`
}

// PresenceDelta is the join/leave event payload.
type PresenceDelta struct {
	ProjectID string         `json:"projectId"`
	DiagramID string         `json:"diagramId"`
	Member    PresenceMember `json:"member"`
}

// DiagramChangedEvent is the payload emitted on every successful autosave or
// checkpoint POST. ReviewNumber is the new monotonic work counter clients
// must echo on their next write; Version is the explicit-checkpoint count
// (unchanged by autosave). ActorID is the user that produced the write so
// connected peers can render "Edited by <name>" and reject stale echoes.
// Kind is one of {"autosave", "checkpoint", "restore"}.
type DiagramChangedEvent struct {
	ActorID      string `json:"actorId"`
	ReviewNumber int64  `json:"reviewNumber"`
	Version      int    `json:"version"`
	Kind         string `json:"kind"`
	Message      string `json:"message,omitempty"`
}

// RealtimeError is the error payload. Code is a stable string clients can
// switch on (e.g. "stale_review"); Message is human-readable.
type RealtimeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
