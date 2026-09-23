// Package httpapi serves the /api/v1 contract (GOBE-02): opaque-token login,
// Bearer authentication, membership-gated project/diagram CRUD with
// versioning, uniform {"message"} errors, and single-origin CORS mirroring
// SecurityConfig.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/realtime"
	"github.com/ai-uml-architect/gobackend/internal/service"
)

// Route describes one public API endpoint.
type Route struct {
	Method  string
	Pattern string
}

// Routes returns the full public /api/v1 route table. Order is stable.
func Routes() []Route {
	return []Route{
		{Method: http.MethodPost, Pattern: "/api/v1/auth/login"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/join"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects/{projectId}/diagrams"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/{projectId}/diagrams"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}"},
		{Method: http.MethodPut, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/checkpoints"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/versions"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/versions/{version}/restore"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/artifact"},
		{Method: http.MethodPost, Pattern: "/api/v1/voice/transcriptions"},
	}
}

// RealtimeRoutes returns the WebSocket upgrade and ticket-issue routes
// layered on top of the REST table. They share the membership gate with
// the underlying store; the hub's resolver enforces it on every upgrade
// attempt. Exposing them here keeps the REST table small while letting
// the RouteTable harness assert their existence end-to-end.
func RealtimeRoutes() []Route {
	return []Route{
		{Method: http.MethodPost, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/realtime-tickets"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/ws"},
	}
}

// ticketIssuer is the boundary the httpapi.Server uses to mint WS handshake
// tickets. Production wires a realtime.TicketSigner; tests inject a stub.
type ticketIssuer interface {
	Issue(projectID, diagramID, userID string, ttl time.Duration) (string, error)
}

// loginPattern is the only route that does not require a Bearer token,
// mirroring SecurityConfig's permitAll for POST /api/v1/auth/login.
const loginPattern = "/api/v1/auth/login"

// Server serves the API over a Service with a configured CORS origin. The
// hub field is optional: when nil, the WebSocket route is not registered.
// tickets is the ticket signer used by the browser-friendly ticket REST
// route; nil disables ticket issuance.
type Server struct {
	services         *service.Service
	origin           string
	hub              HubUpgradeHost
	tickets          ticketIssuer
	voiceTranscriber *DeepgramTranscriber
}

// HubUpgradeHost is the realtime wiring point: http.HandlerFunc returning
// method satisfies it. The interface keeps httpapi decoupled from
// gorilla/websocket so the REST harness stays pure stdlib.
type HubUpgradeHost interface {
	Upgrade(auth realtime.AuthFunc) http.HandlerFunc
}

// NewServer returns a Server. corsOrigin mirrors app.cors.allowed-origin
// (single allowed origin). Pass hub == nil to drop the WebSocket route.
// Pass tickets == nil to drop the ticket REST route. Production wires all
// three so the browser WebSocket has a ticket endpoint it can call.
func NewServer(svc *service.Service, corsOrigin string, hub HubUpgradeHost, tickets ticketIssuer) *Server {
	return &Server{services: svc, origin: corsOrigin, hub: hub, tickets: tickets}
}

// SetVoiceTranscriber configures the authenticated, server-side voice proxy.
func (s *Server) SetVoiceTranscriber(transcriber *DeepgramTranscriber) {
	s.voiceTranscriber = transcriber
}

// NewServerLegacy retains the GOBE-03 constructor: callers that have no
// hub wired (the REST route-table harness) keep working.
func NewServerLegacy(svc *service.Service, corsOrigin string) *Server {
	return &Server{services: svc, origin: corsOrigin}
}

// NewMux preserves the GOBE-01 constructor for the route-table harness: the
// route table, Bearer scheme, and login permitAll behave identically, while
// authenticated routes 401 without a resolvable token (no store attached).
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	for _, route := range Routes() {
		mux.HandleFunc(route.Method+" "+route.Pattern, stub)
	}
	return mux
}

func stub(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != loginPattern && !isBearer(r.Header.Get("Authorization")) {
		writeError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
		return
	}
	writeError(w, http.StatusNotImplemented, "not implemented (GOBE-01 skeleton stub)")
}

type ctxKey struct{}

// Handler builds the full handler chain: CORS -> routes (+ auth per route).
// The WebSocket route is registered only when a Hub is attached so the
// REST endpoint harness can validate the table without a hub instance.
//
// s.hub is expected to be a real *realtime.Hub; we accept the unit
// `HubUpgradeHost` interface when callers want to swap to a static-stub
// hub (used in tests).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(http.MethodPost+" "+loginPattern, s.handleLogin)
	mux.HandleFunc(http.MethodGet+" /api/v1/projects", s.withAuth(s.handleAssigned))
	mux.HandleFunc(http.MethodPost+" /api/v1/projects", s.withAuth(s.handleCreateProject))
	mux.HandleFunc(http.MethodPost+" /api/v1/projects/join", s.withAuth(s.handleJoinProject))
	mux.HandleFunc(http.MethodGet+" /api/v1/projects/{projectId}/diagrams", s.withAuth(s.handleListDiagrams))
	mux.HandleFunc(http.MethodPost+" /api/v1/projects/{projectId}/diagrams", s.withAuth(s.handleCreateDiagram))
	mux.HandleFunc(http.MethodGet+" /api/v1/projects/{projectId}/diagrams/{id}", s.withAuth(s.handleGetDiagram))
	mux.HandleFunc(http.MethodPut+" /api/v1/projects/{projectId}/diagrams/{id}", s.withAuth(s.handleUpdateDiagram))
	mux.HandleFunc(http.MethodPost+" /api/v1/projects/{projectId}/diagrams/{id}/checkpoints", s.withAuth(s.handleCreateCheckpoint))
	mux.HandleFunc(http.MethodGet+" /api/v1/projects/{projectId}/diagrams/{id}/versions", s.withAuth(s.handleListVersions))
	mux.HandleFunc(http.MethodPost+" /api/v1/projects/{projectId}/diagrams/{id}/versions/{version}/restore", s.withAuth(s.handleRestore))
	mux.HandleFunc(http.MethodPost+" /api/v1/projects/{projectId}/diagrams/{id}/artifact", s.withAuth(s.handleGenerateArtifact))
	mux.HandleFunc(http.MethodPost+" /api/v1/voice/transcriptions", s.withAuth(s.handleVoiceTranscription))
	if s.tickets != nil {
		mux.HandleFunc(http.MethodPost+" /api/v1/projects/{projectId}/diagrams/{id}/realtime-tickets", s.withAuth(s.handleIssueRealtimeTicket))
	}
	if s.hub != nil {
		upgrade, ok := s.hub.(realtime.HubUpgrader)
		if ok {
			// No withAuth here: browsers cannot set an Authorization header
			// on a WebSocket handshake, so a Bearer gate would block the
			// ticket path outright. Auth happens inside the upgrade —
			// bearer header or one-shot ?ticket= — followed by the same
			// membership pre-flight every REST endpoint enforces.
			mux.HandleFunc(http.MethodGet+" /api/v1/projects/{projectId}/diagrams/{id}/ws", upgrade.Upgrade(AuthFuncFor(s.services)))
		}
	}
	return s.withCORS(mux)
}

// handleVoiceTranscription keeps the public route protected even when the
// server has no Deepgram credentials. Configuration availability is revealed
// only after the standard Bearer authentication gate succeeds.
func (s *Server) handleVoiceTranscription(w http.ResponseWriter, r *http.Request) {
	if s.voiceTranscriber == nil {
		writeError(w, http.StatusServiceUnavailable, "Voice transcription is not configured")
		return
	}
	s.voiceTranscriber.Handle(w, r)
}

// withAuth mirrors BearerFilter + the authenticated() rule: only
// "Bearer <token>" is accepted, the token must resolve to a live user, and
// anything else is 401 with the uniform envelope.
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !isBearer(header) {
			writeError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
			return
		}
		userID := ""
		var err error
		if s.services != nil {
			userID, err = s.services.Authenticate(r.Context(), header[len("Bearer "):])
		}
		if err != nil || userID == "" {
			writeError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, userID)))
	}
}

func userOf(r *http.Request) string {
	id, _ := r.Context().Value(ctxKey{}).(string)
	return id
}

// projectOf extracts and validates the projectId path value, mirroring
// Spring's UUID @PathVariable conversion (malformed -> 400).
func projectOf(w http.ResponseWriter, r *http.Request) (string, bool) {
	projectID := r.PathValue("projectId")
	if !isValidUUID(projectID) {
		writeError(w, http.StatusBadRequest, "Invalid project id: "+projectID)
		return "", false
	}
	return projectID, true
}

// diagramOf validates the diagram id path value (malformed -> 400).
func diagramOf(w http.ResponseWriter, r *http.Request) (string, bool) {
	diagramID := r.PathValue("id")
	if !isValidUUID(diagramID) {
		writeError(w, http.StatusBadRequest, "Invalid diagram id: "+diagramID)
		return "", false
	}
	return diagramID, true
}

// mapError translates service errors to the ApiExceptionHandler statuses:
// InvalidCredentials -> 401, AccessDenied -> 403, NoSuchElement -> 404,
// IllegalArgument/validation -> 400, optimistic-concurrency -> 409 with the
// server's current document so the client can merge.
func mapError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch e := err.(type) {
	case service.CredentialsError:
		writeError(w, http.StatusUnauthorized, e.Error())
	case service.ForbiddenError:
		writeError(w, http.StatusForbidden, e.Error())
	case service.NotFoundError:
		writeError(w, http.StatusNotFound, e.Message)
	case service.ValidationError:
		writeError(w, http.StatusBadRequest, e.Message)
	case service.ConflictError:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(conflictEnvelope{Message: e.Error(), Current: e.Current})
		return true
	case service.GenerationError:
		writeError(w, http.StatusBadGateway, e.Message)
	default:
		writeError(w, http.StatusInternalServerError, "Request failed")
	}
	return true
}

// conflictEnvelope is the 409 body: the standard message envelope plus the
// current document so the client can reload without a second round-trip.
type conflictEnvelope struct {
	Message string                 `json:"message"`
	Current domain.DiagramDocument `json:"current"`
}

func (s *Server) handleAssigned(w http.ResponseWriter, r *http.Request) {
	out, err := s.services.AssignedProjects(r.Context(), userOf(r))
	if mapError(w, err) {
		return
	}
	if out == nil {
		out = []domain.ProjectResponse{}
	}
	s.writeJSON(w, http.StatusOK, out)
}

// handleCreateProject serves POST /api/v1/projects: the authenticated user
// becomes OWNER and receives the generated classroom access code with the
// created project (201).
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateProjectRequest
	if !decode(w, r, &input) {
		return
	}
	out, err := s.services.CreateProject(r.Context(), userOf(r), input.Name, input.Description)
	if mapError(w, err) {
		return
	}
	s.writeJSON(w, http.StatusCreated, out)
}

// handleJoinProject serves POST /api/v1/projects/join: idempotent membership
// by code, returning the same list item shape as GET /api/v1/projects.
func (s *Server) handleJoinProject(w http.ResponseWriter, r *http.Request) {
	var input domain.JoinProjectRequest
	if !decode(w, r, &input) {
		return
	}
	out, err := s.services.JoinProject(r.Context(), userOf(r), input.AccessCode)
	if mapError(w, err) {
		return
	}
	s.writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleListDiagrams(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectOf(w, r)
	if !ok {
		return
	}
	out, err := s.services.ListDiagrams(r.Context(), projectID, userOf(r))
	if mapError(w, err) {
		return
	}
	if out == nil {
		out = []domain.DiagramSummary{}
	}
	s.writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateDiagram(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectOf(w, r)
	if !ok {
		return
	}
	var raw domain.DiagramDocument
	if !decode(w, r, &raw) {
		return
	}
	doc, err := s.services.CreateDiagram(r.Context(), projectID, userOf(r), raw)
	if mapError(w, err) {
		return
	}
	s.writeJSON(w, http.StatusCreated, doc)
}

func (s *Server) handleGetDiagram(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectOf(w, r)
	if !ok {
		return
	}
	diagramID, ok := diagramOf(w, r)
	if !ok {
		return
	}
	doc, err := s.services.GetDiagram(r.Context(), projectID, diagramID, userOf(r))
	if mapError(w, err) {
		return
	}
	s.writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleUpdateDiagram(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectOf(w, r)
	if !ok {
		return
	}
	diagramID, ok := diagramOf(w, r)
	if !ok {
		return
	}
	var raw domain.DiagramDocument
	if !decode(w, r, &raw) {
		return
	}
	ifMatch := ifMatchVersion(r.Header.Get("If-Match"))
	doc, err := s.services.UpdateDiagram(r.Context(), projectID, diagramID, userOf(r), raw, ifMatch, reviewFromRequest(r))
	if mapError(w, err) {
		return
	}
	s.writeJSON(w, http.StatusOK, doc)
}

// ifMatchVersion extracts a non-negative integer from a strong or weak
// validator ("3", "\"3\"", `W/"3"`). Quotes and the W/ weak marker are
// tolerated; anything that does not parse is returned as nil so the service
// falls back to the body-supplied Version field.
func ifMatchVersion(header string) *int {
	header = strings.TrimSpace(header)
	if header == "" {
		return nil
	}
	header = strings.TrimPrefix(header, "W/")
	header = strings.Trim(header, "\"")
	n, err := strconv.Atoi(header)
	if err != nil || n < 0 {
		return nil
	}
	p := n
	return &p
}

// reviewFromRequest extracts the per-diagram work counter from either the
// JSON body's reviewNumber field or the X-Diagram-Review header. nil is
// returned when neither is set; the service then refuses to overwrite an
// existing diagram blindly and surfaces a 409 carrying the live baseline.
func reviewFromRequest(r *http.Request) *int64 {
	if v := r.Header.Get("X-Diagram-Review"); v != "" {
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return &n
		}
	}
	return nil
}

// fromRequestBody is a body-scan version of reviewFromRequest; reserved
// for routes where the body is the only source (e.g. PATCH) and currently
// unused. It is doc-exported so future service implementations can opt in.
func fromRequestBody(_ domain.DiagramDocument) *int64 { return nil }

// handleCreateCheckpoint is the explicit-save path: it writes the new
// working document AND appends a new diagram_versions row stamping the
// caller as created_by. The body shape matches PUT (DiagramDocument) so the
// client sends the exact working document it wants to checkpoint; the
// optional message travels via the X-Checkpoint-Message header, mirroring
// GitHub's API. The version is supplied through If-Match (preferred) or the
// body's version field.
func (s *Server) handleCreateCheckpoint(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectOf(w, r)
	if !ok {
		return
	}
	diagramID, ok := diagramOf(w, r)
	if !ok {
		return
	}
	var doc domain.DiagramDocument
	if !decode(w, r, &doc) {
		return
	}
	ifMatch := ifMatchVersion(r.Header.Get("If-Match"))
	if ifMatch != nil {
		doc.Version = *ifMatch
	}
	var message *string
	if raw := strings.TrimSpace(r.Header.Get("X-Checkpoint-Message")); raw != "" {
		message = &raw
	}
	version, err := s.services.CreateCheckpoint(r.Context(), projectID, diagramID, userOf(r), doc, message, ifMatch, reviewFromRequest(r))
	if mapError(w, err) {
		return
	}
	s.writeJSON(w, http.StatusCreated, version)
}

func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectOf(w, r)
	if !ok {
		return
	}
	diagramID, ok := diagramOf(w, r)
	if !ok {
		return
	}
	out, err := s.services.ListVersions(r.Context(), projectID, diagramID, userOf(r))
	if mapError(w, err) {
		return
	}
	if out == nil {
		out = []domain.DiagramVersion{}
	}
	s.writeJSON(w, http.StatusOK, out)
}

// handleIssueRealtimeTicket serves POST /api/v1/projects/{projectId}/diagrams/{id}/realtime-tickets.
// It returns a short-lived signed ticket a browser WebSocket can send via
// ?ticket=<...> without writing a custom Authorization header. The route
// is bearer-gated like every other authenticated endpoint; the ticket is
// re-verified at WS upgrade time so a stolen ticket cannot outlive
// realtime.MaxTicketTTL.
func (s *Server) handleIssueRealtimeTicket(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectOf(w, r)
	if !ok {
		return
	}
	diagramID, ok := diagramOf(w, r)
	if !ok {
		return
	}
	ttl := realtime.MaxTicketTTL
	if raw := strings.TrimSpace(r.Header.Get("X-Realtime-Ticket-TTL")); raw != "" {
		if n, err := time.ParseDuration(raw); err == nil && n > 0 && n <= realtime.MaxTicketTTL {
			ttl = n
		}
	}
	ticket, err := s.tickets.Issue(projectID, diagramID, userOf(r), ttl)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not issue realtime ticket")
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{
		"ticket":    ticket,
		"expiresIn": int(ttl.Seconds()),
	})
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectOf(w, r)
	if !ok {
		return
	}
	diagramID, ok := diagramOf(w, r)
	if !ok {
		return
	}
	version := r.PathValue("version")
	n, err := strconv.Atoi(version)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid version: "+version)
		return
	}
	doc, err := s.services.RestoreDiagram(r.Context(), projectID, diagramID, userOf(r), n)
	if mapError(w, err) {
		return
	}
	s.writeJSON(w, http.StatusOK, doc)
}

// decode reads a JSON request body into v, replying with the uniform 400
// envelope when the payload is malformed.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return false
	}
	return true
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	resp, err := s.services.Login(r.Context(), input.Email, input.Password)
	if mapError(w, err) {
		return
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// withCORS mirrors SecurityConfig's CorsConfigurationSource: the single
// configured origin, GET/POST/PUT/OPTIONS methods, Content-Type, If-Match,
// X-Checkpoint-Message, and X-Diagram-Review headers on /api/**.
func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := s.origin != "" && (origin == s.origin || origin == "")
		if r.Method == http.MethodOptions {
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", s.origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, If-Match, X-Checkpoint-Message, X-Diagram-Review")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if allowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.origin)
			w.Header().Set("Vary", "Origin")
		}
		next.ServeHTTP(w, r)
	})
}

// isBearer accepts only the "Bearer <token>" scheme, mirroring BearerFilter.
func isBearer(header string) bool {
	const prefix = "Bearer "
	return len(header) > len(prefix) && header[:len(prefix)] == prefix
}

// isValidUUID accepts only canonical 8-4-4-4-12 hex UUIDs, mirroring Spring's
// UUID @PathVariable conversion.
func isValidUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		c := v[i]
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return false
			}
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(domain.ErrorEnvelope{Message: message})
}
