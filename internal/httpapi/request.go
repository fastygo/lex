package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
)

// frozenContextRequest is a state the caller already obtained from Context.
type frozenContextRequest struct {
	Pack        json.RawMessage `json:"pack"`
	Snapshot    json.RawMessage `json:"snapshot"`
	PackRequest json.RawMessage `json:"pack_request"`
}

// requestProblem is a request-stage refusal before any stage trace exists.
type requestProblem struct {
	status int
	reason string
	detail string
}

func (p requestProblem) Error() string { return p.detail }

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
