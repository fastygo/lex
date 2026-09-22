package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fastygo/lex/internal/verify"
	"github.com/fastygo/lex/internal/wire"
)

// Decider obtains raw typed answers for one caller State and QuestionSet.
type Decider interface {
	AdapterID() string
	AdapterVersion() string
	Evaluate(ctx context.Context, state any, questions map[string]any) (Decision, error)
}

// Decision preserves provider output without interpreting it as authority.
// Metadata stays empty when the provider omits it.
type Decision struct {
	ResolvedModel    string
	Answers          json.RawMessage
	RequestID        string
	EvaluationTimeMS *float64
	Usage            json.RawMessage
}

// decisionResponse is the decision response. StructuralStatus says only whether
// LeX could validate the typed-decision contract; it is never a domain verdict.
type decisionResponse struct {
	ProtocolVersion  string                `json:"protocol_version"`
	Decision         wire.DecisionIdentity `json:"decision"`
	StateHash        string                `json:"state_hash"`
	QuestionSet      json.RawMessage       `json:"question_set"`
	DecisionSet      json.RawMessage       `json:"decision_set"`
	StructuralStatus string                `json:"structural_status"`
	Findings         []verify.Finding      `json:"findings"`
	ReplayAvailable  bool                  `json:"replay_available"`
	ReplayBundle     json.RawMessage       `json:"replay_bundle"`
	Trace            []traceStage          `json:"trace"`
	Retention        retentionView         `json:"retention"`
	AdapterMetadata  map[string]any        `json:"adapter_metadata,omitempty"`
}

func decide(w http.ResponseWriter, request *http.Request, config Config, verifier wire.DecisionVerifier) {
	body, err := readDecision(request)
	if err != nil {
		writeRequestProblem(w, err)
		return
	}
	st := decisionStages()
	if config.Decider == nil {
		tracedProblem(w, http.StatusServiceUnavailable, reasonDecisionProviderUnavailable, "no decision adapter is configured", st.ending("decide", "failed"))
		return
	}
	contextRecord, err := body.contextRecord()
	if err != nil {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonPackError, "the supplied context cannot be bound", st.ending("decide", "failed"))
		return
	}
	if contextRecord != nil {
		findings, err := wire.CheckContext(request.Context(), body.ProjectID, *contextRecord)
		if err != nil {
			if writeRequestStop(w, err, st.ending("decide", "failed")) {
				return
			}
			tracedProblem(w, http.StatusInternalServerError, reasonVerificationError, "the supplied context could not be verified", st.ending("decide", "failed"))
			return
		}
		if len(findings) > 0 {
			tracedProblemWithFindings(w, http.StatusUnprocessableEntity, reasonPackError, "the supplied context failed deterministic verification", st.ending("decide", "failed"), findings)
			return
		}
	}
	remaining, ok := responseBudget(body, contextRecord, config.MaxBodyBytes)
	if !ok {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonResponseBudget, "the decision inputs do not fit the response budget", st.ending("decide", "failed"))
		return
	}
	if writeRequestStop(w, request.Context().Err(), st.ending("decide", "failed")) {
		return
	}
	decision, err := config.Decider.Evaluate(withAnswerBudget(request.Context(), remaining), json.RawMessage(body.State), body.QuestionSet.ProviderQuestions())
	if err != nil {
		writeDecisionFailure(w, err, st)
		return
	}
	if int64(len(decision.Answers)) > remaining {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonResponseBudget, "the decision answers do not fit the response budget", st.completed("decide").ending("verify", "failed"))
		return
	}
	st.completed("decide")
	bundle, err := wire.BuildDecisionBundle(wire.DecisionBundleInput{
		ProjectID: body.ProjectID, Decision: body.Decision, State: body.State,
		QuestionSet: body.QuestionSet, Context: contextRecord,
		AdapterID: config.Decider.AdapterID(), AdapterVersion: config.Decider.AdapterVersion(),
		ResolvedModel: decision.ResolvedModel, Answers: decision.Answers,
	})
	if err != nil {
		tracedProblem(w, http.StatusBadGateway, reasonDecisionError, "the decision adapter returned answers that cannot be sealed", st.ending("verify", "failed"))
		return
	}
	report, err := verifier.ReplayContext(request.Context(), bundle)
	if err != nil {
		if writeRequestStop(w, err, st.ending("verify", "failed")) {
			return
		}
		tracedProblem(w, http.StatusInternalServerError, reasonVerificationError, "the sealed decision bundle could not be verified", st.ending("verify", "failed"))
		return
	}
	sealed, err := wire.DecodeDecisionBundle(bundle)
	if err != nil {
		tracedProblem(w, http.StatusInternalServerError, reasonVerificationError, "the sealed decision bundle could not be decoded", st.ending("verify", "failed"))
		return
	}
	st.completed("verify")
	response := decisionResponse{
		ProtocolVersion:  wire.DecisionProtocolVersion,
		Decision:         body.Decision,
		StateHash:        sealed.State.Hash,
		QuestionSet:      sealed.QuestionSet.Questions,
		DecisionSet:      decisionSetJSON(sealed.DecisionSet),
		StructuralStatus: report.StructuralStatus,
		Findings:         report.Findings,
		ReplayAvailable:  true,
		ReplayBundle:     bundle,
		Trace:            st.view(),
		Retention:        retentionDisclosure(),
		AdapterMetadata:  adapterMetadata(decision),
	}
	if report.StructuralStatus != "valid" {
		writeDecisionStructuralProblem(w, response, config.MaxBodyBytes, st)
		return
	}
	writeDecision(w, http.StatusOK, response, config.MaxBodyBytes, st)
}

func decisionSetJSON(record wire.DecisionRecord) json.RawMessage {
	raw, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	return raw
}

// responseBudget reports how many bytes remain for provider answers once the
// caller inputs are echoed in the response, or false when the inputs alone do
// not fit. Questions and answers appear twice: in the response and in the bundle.
func responseBudget(body decisionRequest, contextRecord *wire.ContextRecord, maxBodyBytes int64) (int64, bool) {
	questions, err := json.Marshal(body.QuestionSet)
	if err != nil {
		return 0, false
	}
	used := int64(len(body.State)+2*len(questions)) + responseReserve
	if contextRecord != nil {
		encoded, err := json.Marshal(contextRecord)
		if err != nil {
			return 0, false
		}
		used += int64(len(encoded))
	}
	remaining := (maxBodyBytes - used) / 2
	return remaining, remaining > 0
}

func writeDecisionFailure(w http.ResponseWriter, err error, st *stages) {
	failed := st.ending("decide", "failed")
	switch {
	case writeRequestStop(w, err, failed):
	case exceedsResponseBudget(err):
		tracedProblem(w, http.StatusUnprocessableEntity, reasonResponseBudget, "the decision response does not fit the response budget", failed)
	case retryableProvider(err):
		tracedProblem(w, http.StatusServiceUnavailable, reasonProviderUnavailable, "the decision provider is unavailable", failed)
	default:
		tracedProblem(w, http.StatusBadGateway, reasonDecisionError, "the decision adapter failed", failed)
	}
}

func writeDecision(w http.ResponseWriter, status int, body decisionResponse, maxBodyBytes int64, st *stages) {
	if body.Findings == nil {
		body.Findings = []verify.Finding{}
	}
	encoded, err := json.Marshal(body)
	if err != nil || int64(len(encoded)) > maxBodyBytes {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonResponseBudget, "the decision response does not fit the response budget", verifyFailed(st))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(encoded)
}

func writeDecisionStructuralProblem(w http.ResponseWriter, body decisionResponse, maxBodyBytes int64, st *stages) {
	payload := map[string]any{
		"structural_status": body.StructuralStatus,
		"findings":          body.Findings,
		"replay_bundle":     body.ReplayBundle,
		"trace":             body.Trace,
	}
	if body.AdapterMetadata != nil {
		payload["adapter_metadata"] = body.AdapterMetadata
	}
	encoded, err := encodeProblem(http.StatusUnprocessableEntity, reasonDecisionError, "typed answers failed deterministic structural checks", payload)
	if err != nil || int64(len(encoded)) > maxBodyBytes {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonResponseBudget, "the decision response does not fit the response budget", verifyFailed(st))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", problemMediaType)
	w.WriteHeader(http.StatusUnprocessableEntity)
	_, _ = w.Write(encoded)
}

func replayDecision(w http.ResponseWriter, request *http.Request, verifier wire.DecisionVerifier, raw []byte) {
	report, err := verifier.ReplayContext(request.Context(), raw)
	if err != nil {
		if writeRequestStop(w, err, replayTraceStatus("failed")) {
			return
		}
		writeProblem(w, http.StatusUnprocessableEntity, reasonInvalidReplayBundle, "replay bundle failed deterministic structural validation")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"replay_status":     "decision_reproduced",
		"structural_status": report.StructuralStatus,
		"findings":          report.Findings,
		"retention":         retentionDisclosure(),
		"trace":             replayTrace(),
	})
}
