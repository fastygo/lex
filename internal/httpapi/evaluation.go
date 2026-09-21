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
	"github.com/fastygo/lex/internal/lifecycle"
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
	ProjectID string            `json:"project_id"`
	Entity    entityRequest     `json:"entity"`
	Query     string            `json:"query"`
	Sources   []sourceDocument  `json:"sources"`
	Metadata  map[string]string `json:"metadata,omitempty"`
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
		writeProblem(w, http.StatusMethodNotAllowed, reasonMethodNotAllowed, "only POST is supported")
		return
	}
	if !acceptsJSON(request.Header.Get("Accept")) {
		writeProblem(w, http.StatusNotAcceptable, reasonNotAcceptable, "Accept must allow application/json")
		return
	}
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		writeProblem(w, http.StatusUnsupportedMediaType, reasonUnsupportedMediaType, "Content-Type must be application/json")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(request.Body, defaultMaxBodyBytes))
	if err != nil {
		writeProblem(w, http.StatusRequestEntityTooLarge, reasonBodyTooLarge, "request body exceeds the configured limit")
		return
	}
	if _, err = canonical.DecodeJSON(raw); err != nil {
		writeProblem(w, http.StatusBadRequest, reasonInvalidJSON, "request body must be one evaluation object")
		return
	}
	if err = wire.ValidateEvaluationRequest(raw); err != nil {
		writeProblem(w, http.StatusBadRequest, reasonInvalidJSON, "request body must be one evaluation object")
		return
	}
	var body evaluationRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&body); err != nil {
		writeProblem(w, http.StatusBadRequest, reasonInvalidJSON, "request body must be one evaluation object")
		return
	}
	projects, _ := request.Context().Value(projectsKey{}).([]string)
	if !containsProject(projects, body.ProjectID) {
		writeProblem(w, http.StatusForbidden, reasonProjectForbidden, "the authenticated principal cannot access this project")
		return
	}
	if !validEntity(body.Entity) || body.Query == "" || len(body.Sources) == 0 {
		writeProblem(w, http.StatusUnprocessableEntity, reasonQuestionError, "entity, query, and at least one source are required")
		return
	}
	if len(body.Sources) > contextmemory.MaxSources {
		writeProblem(w, http.StatusUnprocessableEntity, reasonQuestionError, "a request can include at most 128 sources")
		return
	}

	sources := make([]contextmemory.Source, len(body.Sources))
	packRequest := contextmemory.PackRequest{
		ProjectID: body.ProjectID,
		Query:     body.Query,
		Focus: contextmemory.Focus{
			ID:                 profile.FocusID,
			Objective:          profile.FocusObjective,
			RequiredTrustLevel: profile.FocusTrust,
			Budget:             contextmemory.Budget{MaxItems: profile.FocusMaxItems, MaxChars: profile.FocusMaxChars},
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
		if writeRequestStop(w, err, trace("receive", "completed", "pack", "failed")) {
			return
		}
		tracedProblem(w, http.StatusUnprocessableEntity, reasonPackError, "Context could not freeze the supplied sources", trace("receive", "completed", "pack", "failed"))
		return
	}
	items, err := admissibleEvidence(pack.ContextPack)
	if err != nil {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonPackError, "frozen evidence failed admission", trace("receive", "completed", "pack", "failed"))
		return
	}
	if len(items) == 0 {
		if writeRequestStop(w, request.Context().Err(), trace("receive", "completed", "pack", "completed", "decide", "failed")) {
			return
		}
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
			Trace:           trace("receive", "completed", "pack", "completed", "decide", "skipped", "verify", "completed"),
			Policy:          policyDisclosure(),
			Retention:       retentionDisclosure(),
		}, maxBodyBytes)
		return
	}
	if decider == nil {
		tracedProblem(w, http.StatusServiceUnavailable, reasonDecisionProviderUnavailable, "no decision adapter is configured", trace("receive", "completed", "pack", "completed", "decide", "skipped"))
		return
	}
	snapshotRaw, err := json.Marshal(pack.Snapshot)
	if err != nil {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonPackError, "Context snapshot could not be measured", trace("receive", "completed", "pack", "failed"))
		return
	}
	if int64(len(pack.ContextPack)+len(snapshotRaw))+responseReserve > maxBodyBytes {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonResponseBudget, "the frozen pack does not fit the response budget", trace("receive", "completed", "pack", "completed", "decide", "skipped"))
		return
	}
	if writeRequestStop(w, request.Context().Err(), trace("receive", "completed", "pack", "completed", "decide", "failed")) {
		return
	}
	decision, err := decider.Evaluate(request.Context(), map[string]any{
		"claim": body.Query, "evidence": items,
	}, profile.ProviderQuestions())
	if err != nil {
		if writeRequestStop(w, err, trace("receive", "completed", "pack", "completed", "decide", "failed")) {
			return
		}
		tracedProblem(w, http.StatusBadGateway, reasonDecisionError, "the decision adapter failed", trace("receive", "completed", "pack", "completed", "decide", "failed"))
		return
	}
	if writeRequestStop(w, request.Context().Err(), trace("receive", "completed", "pack", "completed", "decide", "completed", "verify", "failed")) {
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
		tracedProblem(w, http.StatusBadGateway, reasonDecisionError, "the decision adapter returned answers that cannot be sealed", trace("receive", "completed", "pack", "completed", "decide", "completed", "verify", "failed"))
		return
	}
	report, err := wire.Replay(bundle)
	if err != nil {
		tracedProblem(w, http.StatusInternalServerError, reasonVerificationError, "the sealed bundle could not be verified", trace("receive", "completed", "pack", "completed", "decide", "completed", "verify", "failed"))
		return
	}
	if report.Verdict == verify.VerdictError {
		writeVerdictProblem(w, http.StatusUnprocessableEntity, reasonDecisionError, "typed answers failed deterministic checks", report, bundle, maxBodyBytes)
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
		Trace:           trace("receive", "completed", "pack", "completed", "decide", "completed", "verify", "completed"),
		Policy:          policyDisclosure(),
		Retention:       retentionDisclosure(),
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

func tracedProblem(w http.ResponseWriter, status int, reason, detail string, stages []traceStage) {
	writeProblemBody(w, status, reason, detail, map[string]any{"trace": stages})
}

func trace(pairs ...string) []traceStage {
	events := make([]lifecycle.Event, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		events = append(events, lifecycle.Event{Name: pairs[i], Status: pairs[i+1]})
	}
	if err := lifecycle.Run(lifecycle.Evaluation, events); err != nil {
		panic(err)
	}
	stages := make([]traceStage, 0, len(events))
	for _, event := range events {
		stages = append(stages, traceStage{Name: event.Name, Status: event.Status})
	}
	return stages
}

func replayTrace() []traceStage {
	return replayTraceStatus("completed")
}

func replayTraceStatus(status string) []traceStage {
	event := lifecycle.Event{Name: "replay", Status: status}
	if err := lifecycle.Run(lifecycle.Replay, []lifecycle.Event{event}); err != nil {
		panic(err)
	}
	return []traceStage{{Name: event.Name, Status: event.Status}}
}

const statusClientClosedRequest = 499

func writeRequestStop(w http.ResponseWriter, err error, stages []traceStage) bool {
	status, reason, detail, stopped := requestStopped(err)
	if !stopped {
		return false
	}
	tracedProblem(w, status, reason, detail, stages)
	return true
}

func requestStopped(err error) (int, string, string, bool) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout, reasonDeadlineExceeded, "the request deadline elapsed before the stage finished", true
	case errors.Is(err, context.Canceled):
		return statusClientClosedRequest, reasonClientCanceled, "the client disconnected before the stage finished", true
	default:
		return 0, "", "", false
	}
}

func writeEvaluation(w http.ResponseWriter, body evaluationResponse, maxBodyBytes int64) {
	if body.Findings == nil {
		body.Findings = []verify.Finding{}
	}
	encoded, err := json.Marshal(body)
	if err != nil || int64(len(encoded)) > maxBodyBytes {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonResponseBudget, "the evaluation response does not fit the response budget", trace("receive", "completed", "pack", "completed", "decide", "completed", "verify", "failed"))
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
	if entity.Type != profile.EntityType || entity.SchemaVersion != profile.EntitySchemaVersion {
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
