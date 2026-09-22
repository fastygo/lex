package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/evidence"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/wire"
)

// evaluationRequest is the decoded wire body. Exactly one of Sources or
// Context is present; the published schema enforces it and decode re-checks.
type evaluationRequest struct {
	ProjectID string                `json:"project_id"`
	Entity    entityRequest         `json:"entity"`
	Query     string                `json:"query"`
	Sources   []sourceDocument      `json:"sources,omitempty"`
	Context   *frozenContextRequest `json:"context,omitempty"`
	Metadata  map[string]string     `json:"metadata,omitempty"`
}

type entityRequest struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	SchemaVersion string `json:"schema_version"`
	Version       string `json:"version"`
}

type sourceDocument struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Text    string `json:"text"`
}

// frozenContextRequest is a state the caller already obtained from Context.
type frozenContextRequest struct {
	Pack        json.RawMessage `json:"pack"`
	Snapshot    json.RawMessage `json:"snapshot"`
	PackRequest json.RawMessage `json:"pack_request"`
}

func (body evaluationRequest) inputKind() evidence.Kind {
	if body.Context != nil {
		return evidence.KindFrozenContext
	}
	return evidence.KindSources
}

func (body evaluationRequest) entity() wire.Entity {
	return wire.Entity{
		ID: body.Entity.ID, ProjectID: body.ProjectID, Type: body.Entity.Type,
		SchemaVersion: body.Entity.SchemaVersion, Version: body.Entity.Version,
	}
}

// requestProblem is a request-stage refusal before any stage trace exists.
type requestProblem struct {
	status int
	reason string
	detail string
}

func (p requestProblem) Error() string { return p.detail }

// readEvaluation negotiates, bounds, validates, and decodes one evaluation body.
// It performs no retrieval and calls no provider.
func readEvaluation(request *http.Request, prof profile.Profile) (evaluationRequest, error) {
	if request.Method != http.MethodPost {
		return evaluationRequest{}, requestProblem{http.StatusMethodNotAllowed, reasonMethodNotAllowed, "only POST is supported"}
	}
	if !acceptsJSON(request.Header.Get("Accept")) {
		return evaluationRequest{}, requestProblem{http.StatusNotAcceptable, reasonNotAcceptable, "Accept must allow application/json"}
	}
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return evaluationRequest{}, requestProblem{http.StatusUnsupportedMediaType, reasonUnsupportedMediaType, "Content-Type must be application/json"}
	}
	raw, err := io.ReadAll(io.LimitReader(request.Body, defaultMaxBodyBytes))
	if err != nil {
		return evaluationRequest{}, requestProblem{http.StatusRequestEntityTooLarge, reasonBodyTooLarge, "request body exceeds the configured limit"}
	}
	invalid := requestProblem{http.StatusBadRequest, reasonInvalidJSON, "request body must be one evaluation object"}
	if _, err = canonical.DecodeJSON(raw); err != nil {
		return evaluationRequest{}, invalid
	}
	if err = wire.ValidateEvaluationRequest(raw); err != nil {
		return evaluationRequest{}, invalid
	}
	var body evaluationRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&body); err != nil {
		return evaluationRequest{}, invalid
	}
	projects, _ := request.Context().Value(projectsKey{}).([]string)
	if !containsProject(projects, body.ProjectID) {
		return evaluationRequest{}, requestProblem{http.StatusForbidden, reasonProjectForbidden, "the authenticated principal cannot access this project"}
	}
	if !validEntity(prof, body.Entity) || strings.TrimSpace(body.Query) == "" {
		return evaluationRequest{}, requestProblem{http.StatusUnprocessableEntity, reasonQuestionError, "entity and query are required"}
	}
	switch body.inputKind() {
	case evidence.KindSources:
		if len(body.Sources) == 0 {
			return evaluationRequest{}, requestProblem{http.StatusUnprocessableEntity, reasonQuestionError, "at least one source is required"}
		}
		if len(body.Sources) > contextmemory.MaxSources {
			return evaluationRequest{}, requestProblem{http.StatusUnprocessableEntity, reasonQuestionError, "a request can include at most 128 sources"}
		}
	case evidence.KindFrozenContext:
		if len(body.Sources) != 0 {
			return evaluationRequest{}, invalid
		}
	}
	return body, nil
}

func validEntity(prof profile.Profile, entity entityRequest) bool {
	if !prof.AcceptsEntity(entity.Type, entity.SchemaVersion) {
		return false
	}
	for _, field := range []string{entity.ID, entity.Type, entity.SchemaVersion, entity.Version} {
		if field == "" || len(field) > 256 {
			return false
		}
	}
	return true
}

func containsProject(projects []string, projectID string) bool {
	for _, project := range projects {
		if project == projectID {
			return true
		}
	}
	return false
}

func writeRequestProblem(w http.ResponseWriter, err error) bool {
	var problem requestProblem
	if !errors.As(err, &problem) {
		return false
	}
	if problem.status == http.StatusMethodNotAllowed {
		w.Header().Set("Allow", http.MethodPost)
	}
	writeProblem(w, problem.status, problem.reason, problem.detail)
	return true
}
