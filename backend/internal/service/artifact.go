package service

import (
	"context"
	"errors"
	"strings"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/jdlgen"
	"github.com/ai-uml-architect/gobackend/internal/jhipster"
	"github.com/ai-uml-architect/gobackend/internal/store"
)

// GenerationError reports a failure inside the JHipster generation flow
// (runner failure, packaging failure, or a missing generator). The HTTP layer
// maps it to 502 with a sanitized, bounded message.
type GenerationError struct{ Message string }

func (e GenerationError) Error() string { return e.Message }

// ArtifactGenerator produces the downloadable JHipster backend artifact.
// Production wires jhipster.NewGenerator; tests inject a fake.
type ArtifactGenerator interface {
	Generate(ctx context.Context, doc domain.DiagramDocument, o jdlgen.Options) (jhipster.Result, error)
}

// JhipsterResult is the downloadable artifact envelope the service returns.
type JhipsterResult = jhipster.Result

// NewWithJhipster returns a Service that can generate downloadable JHipster
// backend artifacts. Callers that never expose the artifact endpoint can keep
// using New; hitting that endpoint without a generator yields
// GenerationError (mapped to 502).
func NewWithJhipster(s store.Store, gen ArtifactGenerator) *Service {
	return &Service{store: s, jhipGen: gen}
}

// GenerateJhipsterArtifact gates generation on membership and diagram
// existence, validates the supplied document and application options, then
// delegates to the pinned JHipster generator. The returned artifact is a ZIP
// (and manifest) listing exactly the files the generator produced.
//
// Errors: ForbiddenError (403) for non-members, NotFoundError (404) for a
// missing diagram, ValidationError (400) for invalid documents/options, and
// GenerationError (502) when the generator fails or is not configured.
func (s *Service) GenerateJhipsterArtifact(ctx context.Context, projectID, diagramID, userID string, raw domain.DiagramDocument, o jdlgen.Options) (JhipsterResult, error) {
	if err := s.requireMember(ctx, projectID, userID); err != nil {
		return JhipsterResult{}, err
	}
	if _, err := s.store.FindDiagram(ctx, projectID, diagramID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return JhipsterResult{}, NotFoundError{Message: "Diagram not found"}
		}
		return JhipsterResult{}, err
	}
	if errs := domain.ValidateDiagramInput(raw); len(errs) > 0 {
		return JhipsterResult{}, ValidationError{Message: strings.Join(errs, "; ")}
	}
	if errs := domain.ValidateDocument(raw); len(errs) > 0 {
		return JhipsterResult{}, ValidationError{Message: strings.Join(errs, "; ")}
	}
	if errs := jdlgen.ValidateOptions(o); len(errs) > 0 {
		return JhipsterResult{}, ValidationError{Message: strings.Join(errs, "; ")}
	}
	if s.jhipGen == nil {
		return JhipsterResult{}, GenerationError{Message: "JHipster artifact generation is not configured"}
	}
	res, err := s.jhipGen.Generate(ctx, raw, o)
	if err != nil {
		return JhipsterResult{}, GenerationError{Message: err.Error()}
	}
	return res, nil
}
