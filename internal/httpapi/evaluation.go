package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

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
// Metadata stays empty when the provider omits it.
type Decision struct {
	ResolvedModel    string
	Answers          json.RawMessage
	RequestID        string
	EvaluationTimeMS *float64
	Usage            json.RawMessage
}

// evaluate runs one synchronous evaluation:
//
//	receive -> pack (freeze or accept evidence) -> decide -> verify
//
// Each stage either advances the recorder or ends the request with a traced problem.
func evaluate(w http.ResponseWriter, request *http.Request, config Config, verifier wire.Verifier) {
	prof := config.profile()
	body, err := readEvaluation(request, prof)
	if err != nil {
		writeRequestProblem(w, err)
		return
	}
	st := evaluationStages()

	frozen, err := freezeEvidence(request.Context(), prof, body)
	if err != nil {
		if writeRequestStop(w, err, st.ending("pack", "failed")) {
			return
		}
		var refusal packRefusal
		if errors.As(err, &refusal) {
			tracedProblemWithFindings(w, http.StatusUnprocessableEntity, refusal.reason, refusal.detail, st.ending("pack", "failed"), refusal.findings)
			return
		}
		tracedProblem(w, http.StatusUnprocessableEntity, reasonPackError, "Context could not freeze the supplied sources", st.ending("pack", "failed"))
		return
	}
	items, err := evidence.Items(frozen.Pack)
	if err != nil {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonPackError, "frozen evidence failed admission", st.ending("pack", "failed"))
		return
	}
	st.completed("pack")
	if len(items) == 0 {
		if writeRequestStop(w, request.Context().Err(), st.ending("decide", "failed")) {
			return
		}
		writeSkippedEvaluation(w, request, prof, body, frozen, config, verifier, st)
		return
	}
	if config.Decider == nil {
		tracedProblem(w, http.StatusServiceUnavailable, reasonDecisionProviderUnavailable, "no decision adapter is configured", st.ending("decide", "skipped"))
		return
	}
	remaining, ok := responseBudget(frozen, config.MaxBodyBytes)
	if !ok {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonResponseBudget, "the frozen pack does not fit the response budget", st.ending("decide", "skipped"))
		return
	}
	if writeRequestStop(w, request.Context().Err(), st.ending("decide", "failed")) {
		return
	}

	decision, err := config.Decider.Evaluate(withAnswerBudget(request.Context(), remaining), prof.State(body.Query, items), prof.ProviderQuestions())
	if err != nil {
		writeDecisionFailure(w, err, st)
		return
	}
	if int64(len(decision.Answers)) > remaining {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonResponseBudget, "the decision answers do not fit the response budget", st.completed("decide").ending("verify", "failed"))
		return
	}
	st.completed("decide")
	if writeRequestStop(w, request.Context().Err(), st.ending("verify", "failed")) {
		return
	}

	bundle, err := wire.BuildBundle(prof, wire.BundleInput{
		Entity: body.entity(), Frozen: frozen,
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
		status, reason, detail := replayFailure(err)
		tracedProblem(w, status, reason, detail, st.ending("verify", "failed"))
		return
	}
	st.completed("verify")
	if report.Verdict == verify.VerdictError {
		writeVerdictProblem(w, http.StatusUnprocessableEntity, reasonDecisionError, "typed answers failed deterministic checks", report, bundle, adapterMetadata(decision), config.MaxBodyBytes, st)
		return
	}
	writeEvaluation(w, evaluationResponse{
		ProtocolVersion: wire.ProtocolVersion,
		Verdict:         string(report.Verdict),
		Findings:        report.Findings,
		ContextRuntime:  frozen.Snapshot.RuntimeVersion,
		ResolvedModel:   decision.ResolvedModel,
		ReplayAvailable: true,
		ReplayBundle:    bundle,
		Trace:           st.view(),
		Policy:          policyDisclosure(prof),
		Retention:       retentionDisclosure(),
		AdapterMetadata: adapterMetadata(decision),
	}, config.MaxBodyBytes, st)
}

// packRefusal is a deterministic pack-stage refusal with optional findings.
type packRefusal struct {
	reason   string
	detail   string
	findings []verify.Finding
}

func (r packRefusal) Error() string { return r.detail }

// freezeEvidence obtains the frozen state through the input the caller chose.
// Both inputs end in the same Frozen state and the same verifier checks.
func freezeEvidence(ctx context.Context, prof profile.Profile, body evaluationRequest) (evidence.Frozen, error) {
	switch body.inputKind() {
	case evidence.KindFrozenContext:
		return acceptFrozenContext(ctx, prof, body)
	default:
		sources := make([]evidence.Source, len(body.Sources))
		for i, source := range body.Sources {
			sources[i] = evidence.Source{ID: source.ID, Version: source.Version, Text: source.Text}
		}
		return evidence.FromSources(ctx, body.ProjectID, evidence.PackRequest(body.ProjectID, body.Query, prof.Focus()), sources)
	}
}

// acceptFrozenContext takes a state the caller already obtained from Context.
// LeX does not trust it: the same controls the verifier applies at replay run
// here first, so a state Context cannot reproduce never reaches a provider.
func acceptFrozenContext(ctx context.Context, prof profile.Profile, body evaluationRequest) (evidence.Frozen, error) {
	frozen, err := evidence.Decode(body.Context.Pack, body.Context.Snapshot, body.Context.PackRequest)
	if err != nil {
		return evidence.Frozen{}, packRefusal{reason: reasonPackError, detail: "the supplied context is not a Context frozen state"}
	}
	if frozen.PackRequest.Query != body.Query {
		return evidence.Frozen{}, packRefusal{reason: reasonQuestionError, detail: "query must equal the frozen pack request query"}
	}
	findings, err := wire.CheckContext(ctx, prof, body.ProjectID, wire.ContextRecord{
		Pack: body.Context.Pack, Snapshot: body.Context.Snapshot, PackRequest: body.Context.PackRequest,
	})
	if err != nil {
		return evidence.Frozen{}, err
	}
	for _, finding := range findings {
		if finding.Verdict == verify.VerdictError {
			return evidence.Frozen{}, packRefusal{reason: reasonPackError, detail: "the supplied frozen context failed deterministic verification", findings: findings}
		}
	}
	return frozen, nil
}

// responseBudget reports how many bytes remain for provider answers once the
// frozen state is echoed in the response, or false when the state alone does not fit.
func responseBudget(frozen evidence.Frozen, maxBodyBytes int64) (int64, bool) {
	snapshot, err := json.Marshal(frozen.Snapshot)
	if err != nil {
		return 0, false
	}
	request, err := json.Marshal(frozen.PackRequest)
	if err != nil {
		return 0, false
	}
	used := int64(len(frozen.Pack)+len(snapshot)+len(request)) + responseReserve
	if used >= maxBodyBytes {
		return 0, false
	}
	return maxBodyBytes - used, true
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

// writeSkippedEvaluation seals and verifies an evaluation whose pack has no
// admissible evidence. No provider is called; the verdict is insufficient.
func writeSkippedEvaluation(w http.ResponseWriter, request *http.Request, prof profile.Profile, body evaluationRequest, frozen evidence.Frozen, config Config, verifier wire.Verifier, st *stages) {
	if _, ok := responseBudget(frozen, config.MaxBodyBytes); !ok {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonResponseBudget, "the frozen pack does not fit the response budget", st.ending("decide", "skipped"))
		return
	}
	bundle, err := wire.BuildBundle(prof, wire.BundleInput{Entity: body.entity(), Frozen: frozen, Skipped: true})
	if err != nil {
		tracedProblem(w, http.StatusInternalServerError, reasonVerificationError, "the sealed bundle could not be verified", st.ending("decide", "failed"))
		return
	}
	report, err := verifier.ReplayContext(request.Context(), bundle)
	if err != nil {
		if writeRequestStop(w, err, st.ending("decide", "failed")) {
			return
		}
		tracedProblem(w, http.StatusInternalServerError, reasonVerificationError, "the sealed bundle could not be verified", st.ending("decide", "failed"))
		return
	}
	if report.Verdict != verify.VerdictInsufficient {
		tracedProblem(w, http.StatusInternalServerError, reasonVerificationError, "the sealed bundle could not be verified", st.ending("decide", "failed"))
		return
	}
	st.skipped("decide").completed("verify")
	writeEvaluation(w, evaluationResponse{
		ProtocolVersion: wire.ProtocolVersion,
		Verdict:         string(report.Verdict),
		Findings:        report.Findings,
		ContextRuntime:  frozen.Snapshot.RuntimeVersion,
		ReplayAvailable: true,
		ReplayBundle:    bundle,
		Stage:           "retrieval",
		Trace:           st.view(),
		Policy:          policyDisclosure(prof),
		Retention:       retentionDisclosure(),
	}, config.MaxBodyBytes, st)
}
