package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
)

// artifactTimeout bounds one generation run. The first run also downloads the
// pinned generator into pnpm's store, which is the slowest path; the limit
// gives a single request room for that without hanging the process.
const artifactTimeout = 10 * time.Minute

// artifactRequest is the POST body of the artifact endpoint: the full UML
// document (same schema the client autosaves via PUT) plus optional
// application configuration. Omitted config fields fall back to
// jdlgen.DefaultOptions so the body stays minimal.
type artifactRequest struct {
	Document domain.DiagramDocument `json:"document"`
	Config   *artifactConfig        `json:"config"`
}

type artifactConfig struct {
	BaseName           string `json:"baseName"`
	PackageName        string `json:"packageName"`
	BuildTool          string `json:"buildTool"`
	AuthenticationType string `json:"authenticationType"`
}

// optionsFromRequest merges an optional config over the neutral defaults.
func optionsFromRequest(c *artifactConfig) jdlgen.Options {
	o := jdlgen.DefaultOptions()
	if c == nil {
		return o
	}
	if c.BaseName != "" {
		o.BaseName = c.BaseName
	}
	if c.PackageName != "" {
		o.PackageName = c.PackageName
	}
	if c.BuildTool != "" {
		o.BuildTool = c.BuildTool
	}
	if c.AuthenticationType != "" {
		o.AuthenticationType = c.AuthenticationType
	}
	return o
}

// handleGenerateArtifact serves
// POST /api/v1/projects/{projectId}/diagrams/{id}/artifact: it validates the
// supplied document, generates a standalone backend-only JHipster project
// from it, and returns a ZIP (plus manifest.json) that lists exactly the
// files the pinned generator produced.
func (s *Server) handleGenerateArtifact(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectOf(w, r)
	if !ok {
		return
	}
	diagramID, ok := diagramOf(w, r)
	if !ok {
		return
	}
	var req artifactRequest
	if !decode(w, r, &req) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), artifactTimeout)
	defer cancel()
	res, err := s.services.GenerateJhipsterArtifact(ctx, projectID, diagramID, userOf(r), req.Document, optionsFromRequest(req.Config))
	if mapError(w, err) {
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+res.FileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(res.Content)
}
