package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/store"
)

const maxDiagramImageBytes int64 = 10 << 20

// imageImportConfig is kept in the handler so provider credentials never cross
// the HTTP boundary. The provider is called only after authentication and
// project membership have been checked.
type imageImportConfig struct {
	APIKey  string
	Models  []string
	BaseURL string
}

// GeminiImageImporter is the narrow provider port used by the HTTP handler.
type GeminiImageImporter struct {
	cfg    imageImportConfig
	client *http.Client
}

func NewGeminiImageImporter(apiKey string, models []string, baseURL string) *GeminiImageImporter {
	return &GeminiImageImporter{cfg: imageImportConfig{APIKey: apiKey, Models: append([]string(nil), models...), BaseURL: strings.TrimRight(baseURL, "/")}, client: &http.Client{Timeout: 90 * time.Second}}
}

func (g *GeminiImageImporter) Import(ctx context.Context, mime string, image []byte) (domain.DiagramDocument, error) {
	if strings.TrimSpace(g.cfg.APIKey) == "" {
		return domain.DiagramDocument{}, errGeminiNotConfigured{}
	}
	prompt := `Analyze this UML class diagram and return ONLY one JSON object compatible with UMLDiagramDocument. The exact shape is {"schemaVersion":1,"name":string,"classes":[{"id":string,"name":string,"stereotype":string|null,"package":string|null,"tableBinding":string|null,"x":integer,"y":integer,"width":integer|null,"attributes":[{"id":string,"name":string,"type":string,"visibility":string|null}],"methods":[{"id":string,"name":string,"returnType":string,"visibility":string|null}],"isAssociationClass":boolean|null,"attachedRelationshipId":string|null}],"relationships":[{"id":string,"sourceId":string,"targetId":string,"type":"association|aggregation|composition|generalization|realization|dependency","sourceMultiplicity":string|null,"targetMultiplicity":string|null,"label":string|null}]}. Use stable unique ids, integer coordinates, and empty arrays when absent. Do not include markdown or explanatory text.`
	reqBody := map[string]any{"contents": []any{map[string]any{"parts": []any{map[string]any{"text": prompt}, map[string]any{"inline_data": map[string]string{"mime_type": mime, "data": base64.StdEncoding.EncodeToString(image)}}}}}, "generationConfig": map[string]any{"responseMimeType": "application/json", "temperature": 0.1}}
	body, _ := json.Marshal(reqBody)
	if len(g.cfg.Models) == 0 {
		return domain.DiagramDocument{}, errors.New("Gemini has no configured models")
	}
	var lastErr error
	var raw []byte
	usedModel := ""
	for _, model := range g.cfg.Models {
		endpoint := fmt.Sprintf("%s/models/%s:generateContent", g.cfg.BaseURL, url.PathEscape(model))
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return domain.DiagramDocument{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		// Authenticate via the official Gemini header. The API key must
		// never appear as a query parameter: URLs are logged by proxies,
		// load balancers, and browser history, leaking credentials.
		req.Header.Set("X-Goog-Api-Key", g.cfg.APIKey)
		resp, err := g.client.Do(req)
		if err != nil {
			return domain.DiagramDocument{}, err
		}
		candidateRaw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if readErr != nil {
			return domain.DiagramDocument{}, errors.New("Gemini response could not be read")
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			raw, usedModel = candidateRaw, model
			break
		}
		lastErr = fmt.Errorf("Gemini request failed (%d; model %s; attempted %s)", resp.StatusCode, model, strings.Join(g.cfg.Models, ", "))
		if !retryableGeminiStatus(resp.StatusCode) {
			return domain.DiagramDocument{}, lastErr
		}
	}
	if len(raw) == 0 {
		if lastErr != nil {
			return domain.DiagramDocument{}, lastErr
		}
		return domain.DiagramDocument{}, errors.New("Gemini returned no response")
	}
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Candidates) == 0 {
		return domain.DiagramDocument{}, fmt.Errorf("Gemini model %s returned no diagram", usedModel)
	}
	if len(result.Candidates[0].Content.Parts) == 0 {
		return domain.DiagramDocument{}, fmt.Errorf("Gemini model %s returned no diagram", usedModel)
	}
	text := strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)
	text = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "```json"), "```"))
	var doc domain.DiagramDocument
	dec := json.NewDecoder(strings.NewReader(text))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return domain.DiagramDocument{}, fmt.Errorf("Gemini model %s returned invalid UML JSON", usedModel)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return domain.DiagramDocument{}, fmt.Errorf("Gemini model %s returned multiple JSON values", usedModel)
	}
	normalizeDocument(&doc)
	if len(doc.Classes) == 0 {
		return domain.DiagramDocument{}, errors.New("No UML classes were detected")
	}
	if errs := domain.ValidateDocument(doc); len(errs) > 0 {
		return domain.DiagramDocument{}, fmt.Errorf("Generated UML is invalid: %s", strings.Join(errs, "; "))
	}
	return doc, nil
}

func retryableGeminiStatus(status int) bool {
	return status == http.StatusNotFound || status == http.StatusTooManyRequests || (status >= 500 && status <= 599)
}

func normalizeDocument(doc *domain.DiagramDocument) {
	doc.SchemaVersion = 1
	doc.Version = 0
	doc.ReviewNumber = 0
	if strings.TrimSpace(doc.Name) == "" {
		doc.Name = "Imported UML diagram"
	}
	for i := range doc.Classes {
		if strings.TrimSpace(doc.Classes[i].ID) == "" {
			doc.Classes[i].ID = store.NewUUID()
		}
		if doc.Classes[i].Attributes == nil {
			doc.Classes[i].Attributes = []domain.Attribute{}
		}
		if doc.Classes[i].Methods == nil {
			doc.Classes[i].Methods = []domain.Method{}
		}
		for j := range doc.Classes[i].Attributes {
			if strings.TrimSpace(doc.Classes[i].Attributes[j].ID) == "" {
				doc.Classes[i].Attributes[j].ID = store.NewUUID()
			}
		}
		for j := range doc.Classes[i].Methods {
			if strings.TrimSpace(doc.Classes[i].Methods[j].ID) == "" {
				doc.Classes[i].Methods[j].ID = store.NewUUID()
			}
		}
	}
	for i := range doc.Relationships {
		if strings.TrimSpace(doc.Relationships[i].ID) == "" {
			doc.Relationships[i].ID = store.NewUUID()
		}
	}
	if doc.Classes == nil {
		doc.Classes = []domain.UmlClass{}
	}
	if doc.Relationships == nil {
		doc.Relationships = []domain.Relationship{}
	}
}

type errGeminiNotConfigured struct{}

func (errGeminiNotConfigured) Error() string {
	return "Image import is not configured: set GEMINI_API_KEY"
}

func (s *Server) handleImportDiagramImage(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectOf(w, r)
	if !ok {
		return
	}
	diagramID, ok := diagramOf(w, r)
	if !ok {
		return
	}
	if _, err := s.services.GetDiagram(r.Context(), projectID, diagramID, userOf(r)); err != nil {
		if mapError(w, err) {
			return
		}
		return
	}
	member, err := s.services.Store().IsMember(r.Context(), projectID, userOf(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Request failed")
		return
	}
	if !member {
		writeError(w, http.StatusForbidden, "Project membership required")
		return
	}
	if r.ContentLength > maxDiagramImageBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Image exceeds 10 MB limit")
		return
	}
	if err := r.ParseMultipartForm(maxDiagramImageBytes); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid multipart image")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Image file is required")
		return
	}
	defer file.Close()
	if header.Size > maxDiagramImageBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Image exceeds 10 MB limit")
		return
	}
	mime := header.Header.Get("Content-Type")
	if mime == "" {
		mime = "application/octet-stream"
	}
	allowed := map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true}
	if !allowed[mime] {
		writeError(w, http.StatusUnsupportedMediaType, "Only PNG, JPEG, or WEBP images are supported")
		return
	}
	image, err := io.ReadAll(io.LimitReader(file, maxDiagramImageBytes+1))
	if err != nil || int64(len(image)) > maxDiagramImageBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Image exceeds 10 MB limit")
		return
	}
	if s.imageImporter == nil {
		writeError(w, http.StatusServiceUnavailable, "Image import is not configured: set GEMINI_API_KEY")
		return
	}
	doc, err := s.imageImporter.Import(r.Context(), mime, image)
	if err != nil {
		var missing errGeminiNotConfigured
		if errors.As(err, &missing) {
			writeError(w, http.StatusServiceUnavailable, err.Error())
		} else {
			writeError(w, http.StatusBadGateway, err.Error())
		}
		return
	}
	s.writeJSON(w, http.StatusOK, doc)
}
