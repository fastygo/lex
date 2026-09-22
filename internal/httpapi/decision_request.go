package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/wire"
)

// decisionRequest is the caller-owned generic decision contract. State and
// QuestionSet are passed to the adapter without domain interpretation.
type decisionRequest struct {
	ProjectID   string                  `json:"project_id"`
	Decision    wire.DecisionIdentity   `json:"decision"`
	State       json.RawMessage         `json:"state"`
	QuestionSet wire.GenericQuestionSet `json:"question_set"`
	Context     *frozenContextRequest   `json:"context,omitempty"`
	Metadata    map[string]string       `json:"metadata,omitempty"`
}

// readDecision validates and decodes a generic decision request. It does not
// retrieve, construct a Context pack, select a profile, or call a provider.
func readDecision(request *http.Request) (decisionRequest, error) {
	if request.Method != http.MethodPost {
		return decisionRequest{}, requestProblem{http.StatusMethodNotAllowed, reasonMethodNotAllowed, "only POST is supported"}
	}
	if !acceptsJSON(request.Header.Get("Accept")) {
		return decisionRequest{}, requestProblem{http.StatusNotAcceptable, reasonNotAcceptable, "Accept must allow application/json"}
	}
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return decisionRequest{}, requestProblem{http.StatusUnsupportedMediaType, reasonUnsupportedMediaType, "Content-Type must be application/json"}
	}
	raw, err := io.ReadAll(io.LimitReader(request.Body, defaultMaxBodyBytes))
	if err != nil {
		return decisionRequest{}, requestProblem{http.StatusRequestEntityTooLarge, reasonBodyTooLarge, "request body exceeds the configured limit"}
	}
	invalid := requestProblem{http.StatusBadRequest, reasonInvalidJSON, "request body must be one decision object"}
	if _, err = canonical.DecodeJSON(raw); err != nil {
		return decisionRequest{}, invalid
	}
	if err = wire.ValidateDecisionRequest(raw); err != nil {
		return decisionRequest{}, invalid
	}
	var body decisionRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&body); err != nil {
		return decisionRequest{}, invalid
	}
	projects, _ := request.Context().Value(projectsKey{}).([]string)
	if !containsProject(projects, body.ProjectID) {
		return decisionRequest{}, requestProblem{http.StatusForbidden, reasonProjectForbidden, "the authenticated principal cannot access this project"}
	}
	if err := body.QuestionSet.Validate(); err != nil {
		return decisionRequest{}, requestProblem{http.StatusUnprocessableEntity, reasonQuestionError, "question set is not a supported typed-decision contract"}
	}
	return body, nil
}

func (body decisionRequest) contextRecord() (*wire.ContextRecord, error) {
	if body.Context == nil {
		return nil, nil
	}
	hash, err := canonical.HashJSON(body.Context.Pack)
	if err != nil {
		return nil, err
	}
	return &wire.ContextRecord{
		Pack: body.Context.Pack, Snapshot: body.Context.Snapshot,
		PackRequest: body.Context.PackRequest, PackHash: hash,
	}, nil
}
