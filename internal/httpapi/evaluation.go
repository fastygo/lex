package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/evidence"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/verify"
	"github.com/fastygo/lex/internal/wire"
)

// Decider obtains raw typed answers for one frozen state.
type Decider interface {
	AdapterID() string
	AdapterVersion() string
	Evaluate(ctx context.Context, state any, questions map[string]any) (Decision, error)
}

// Decision preserves provider output without interpreting it as authority.
type Decision struct {
	ResolvedModel string
	Answers       json.RawMessage
}

type evaluationRequest struct {
	ProjectID string           `json:"project_id"`
	Entity    entityRequest    `json:"entity"`
	Query     string           `json:"query"`
	Sources   []sourceDocument `json:"sources"`
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

type evidenceView struct {
	EvidenceItems []struct {
		Class      string `json:"class"`
		TrustLevel string `json:"trust_level"`
		Surface    string `json:"surface"`
		SourceRef  struct {
			SourceID string `json:"source_id"`
		} `json:"source_ref"`
	} `json:"evidence_items"`
}

func evaluate(w http.ResponseWriter, request *http.Request, decider Decider, maxBodyBytes int64) {
	if request.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeProblem(w, http.StatusMethodNotAllowed, "method_not_allowed", "only POST is supported")
		return
	}
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		writeProblem(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(request.Body, defaultMaxBodyBytes))
	if err != nil {
		writeProblem(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds the configured limit")
		return
	}
	if _, err = canonical.DecodeJSON(raw); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_json", "request body must be one evaluation object")
		return
	}
	var body evaluationRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&body); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_json", "request body must be one evaluation object")
		return
	}
	projects, _ := request.Context().Value(projectsKey{}).([]string)
	if !containsProject(projects, body.ProjectID) {
		writeProblem(w, http.StatusForbidden, "project_forbidden", "the authenticated principal cannot access this project")
		return
	}
	if !validEntity(body.Entity) || body.Query == "" || len(body.Sources) == 0 {
		writeProblem(w, http.StatusUnprocessableEntity, "question_error", "entity, query, and at least one source are required")
		return
	}

	sources := make([]contextmemory.Source, len(body.Sources))
	packRequest := contextmemory.PackRequest{
		ProjectID: body.ProjectID,
		Query:     body.Query,
		Focus: contextmemory.Focus{
			ID:                 profile.FocusID,
			Objective:          "Select admissible source text for the stated claim.",
			RequiredTrustLevel: "project",
			Budget:             contextmemory.Budget{MaxItems: 8, MaxChars: 65536},
		},
	}
	for i, source := range body.Sources {
		sources[i] = contextmemory.Source{
			SourceID: source.ID, Version: source.Version, Text: source.Text,
			TrustLevel: "project", EvidenceClass: "source_text",
		}
	}
	pack, err := evidence.BuildPack(request.Context(), body.ProjectID, sources, packRequest)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "pack_error", "Context could not freeze the supplied sources")
		return
	}
	items, err := admissibleEvidence(pack.ContextPack)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "pack_error", "frozen evidence failed admission")
		return
	}
	if len(items) == 0 {
		writeEvaluation(w, evaluationResponse{
			ProtocolVersion: "0.1-draft",
			Verdict:         string(verify.VerdictInsufficient),
			Findings: []verify.Finding{{
				Code: "no_eligible_evidence", Verdict: verify.VerdictInsufficient,
				Detail: "exact retrieval selected no admissible evidence",
			}},
			ContextRuntime:  pack.Snapshot.RuntimeVersion,
			ReplayAvailable: false,
			Stage:           "retrieval",
			Trace: []traceStage{
				{Name: "receive", Status: "completed"},
				{Name: "pack", Status: "completed"},
				{Name: "decide", Status: "skipped"},
				{Name: "verify", Status: "completed"},
			},
			Policy:    policyDisclosure(),
			Retention: retentionDisclosure(),
		}, maxBodyBytes)
		return
	}
	if decider == nil {
		writeProblem(w, http.StatusServiceUnavailable, "decision_provider_unavailable", "no decision adapter is configured")
		return
	}
	snapshotRaw, err := json.Marshal(pack.Snapshot)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "pack_error", "Context snapshot could not be measured")
		return
	}
	if int64(len(pack.ContextPack)+len(snapshotRaw))+responseReserve > maxBodyBytes {
		writeProblem(w, http.StatusUnprocessableEntity, "response_budget", "the frozen pack does not fit the response budget")
		return
	}
	decision, err := decider.Evaluate(request.Context(), map[string]any{
		"claim": body.Query, "evidence": items,
	}, profile.ProviderQuestions())
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			writeProblem(w, http.StatusGatewayTimeout, "deadline_exceeded", "the decision adapter did not finish before the deadline")
			return
		}
		writeProblem(w, http.StatusBadGateway, "decision_error", "the decision adapter failed")
		return
	}
	bundle, err := wire.BuildBundle(wire.BundleInput{
		Entity: wire.Entity{
			ID: body.Entity.ID, ProjectID: body.ProjectID, Type: body.Entity.Type,
			SchemaVersion: body.Entity.SchemaVersion, Version: body.Entity.Version,
		},
		Pack: pack.ContextPack, Snapshot: pack.Snapshot, PackRequest: packRequest,
		AdapterID: decider.AdapterID(), AdapterVersion: decider.AdapterVersion(),
		ResolvedModel: decision.ResolvedModel, Answers: decision.Answers,
	})
	if err != nil {
		writeProblem(w, http.StatusBadGateway, "decision_error", "the decision adapter returned answers that cannot be sealed")
		return
	}
	report, err := wire.Replay(bundle)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "verification_error", "the sealed bundle could not be verified")
		return
	}
	if report.Verdict == verify.VerdictError {
		writeVerdictProblem(w, http.StatusUnprocessableEntity, "decision_error", "typed answers failed deterministic checks", report)
		return
	}
	writeEvaluation(w, evaluationResponse{
		ProtocolVersion: "0.1-draft",
		Verdict:         string(report.Verdict),
		Findings:        report.Findings,
		ContextRuntime:  pack.Snapshot.RuntimeVersion,
		ResolvedModel:   decision.ResolvedModel,
		ReplayAvailable: true,
		ReplayBundle:    bundle,
		Trace: []traceStage{
			{Name: "receive", Status: "completed"},
			{Name: "pack", Status: "completed"},
			{Name: "decide", Status: "completed"},
			{Name: "verify", Status: "completed"},
		},
		Policy:    policyDisclosure(),
		Retention: retentionDisclosure(),
	}, maxBodyBytes)
}

type evaluationResponse struct {
	ProtocolVersion string               `json:"protocol_version"`
	Verdict         string               `json:"verdict"`
	Findings        []verify.Finding     `json:"findings"`
	ContextRuntime  string               `json:"context_runtime"`
	ResolvedModel   string               `json:"resolved_model,omitempty"`
	ReplayAvailable bool                 `json:"replay_available"`
	ReplayBundle    json.RawMessage      `json:"replay_bundle,omitempty"`
	Stage           string               `json:"stage,omitempty"`
	Trace           []traceStage         `json:"trace"`
	Policy          policyDisclosureView `json:"policy"`
	Retention       retentionView        `json:"retention"`
}

type traceStage struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type policyDisclosureView struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Calibration string `json:"calibration"`
}

func policyDisclosure() policyDisclosureView {
	return policyDisclosureView{ID: profile.PolicyID, Version: profile.PolicyVersion, Calibration: profile.Calibration}
}

func writeEvaluation(w http.ResponseWriter, body evaluationResponse, maxBodyBytes int64) {
	if body.Findings == nil {
		body.Findings = []verify.Finding{}
	}
	encoded, err := json.Marshal(body)
	if err != nil || int64(len(encoded)) > maxBodyBytes {
		writeProblem(w, http.StatusUnprocessableEntity, "response_budget", "the evaluation response does not fit the response budget")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(encoded)
}

func admissibleEvidence(raw json.RawMessage) ([]map[string]string, error) {
	var pack evidenceView
	if err := json.Unmarshal(raw, &pack); err != nil {
		return nil, err
	}
	items := make([]map[string]string, 0, len(pack.EvidenceItems))
	for _, item := range pack.EvidenceItems {
		if item.Class != "source_text" || item.TrustLevel != "project" || item.Surface == "" {
			return nil, errIneligible
		}
		items = append(items, map[string]string{
			"class": item.Class, "trust_level": item.TrustLevel, "surface": item.Surface, "source_id": item.SourceRef.SourceID,
		})
	}
	return items, nil
}

var errIneligible = ineligibleError{}

type ineligibleError struct{}

func (ineligibleError) Error() string { return "ineligible evidence" }

func validEntity(entity entityRequest) bool {
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
