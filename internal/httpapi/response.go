package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fastygo/lex/internal/lifecycle"
	"github.com/fastygo/lex/internal/verify"
)

type retentionView struct {
	ServerHistory bool   `json:"server_history"`
	Replay        string `json:"replay"`
	Idempotency   string `json:"idempotency"`
}

func retentionDisclosure() retentionView {
	return retentionView{ServerHistory: false, Replay: "caller_owned", Idempotency: "none"}
}

// verifyFailed renders a trace whose verify stage failed after the recorder
// already marked it completed, as happens when the response itself does not fit.
func verifyFailed(st *stages) []traceStage {
	events := append([]lifecycle.Event(nil), st.events...)
	for i := range events {
		if events[i].Name == "verify" {
			events[i].Status = "failed"
		}
	}
	return render(st.kind, events)
}

func adapterMetadata(decision Decision) map[string]any {
	if decision.RequestID == "" && decision.EvaluationTimeMS == nil && len(decision.Usage) == 0 {
		return nil
	}
	view := map[string]any{}
	if decision.RequestID != "" {
		view["request_id"] = decision.RequestID
	}
	if decision.EvaluationTimeMS != nil {
		view["evaluation_time_ms"] = *decision.EvaluationTimeMS
	}
	if len(decision.Usage) > 0 {
		var usage any
		if err := json.Unmarshal(decision.Usage, &usage); err != nil {
			return nil
		}
		view["usage"] = usage
	}
	return view
}

func tracedProblem(w http.ResponseWriter, status int, reason, detail string, trace []traceStage) {
	writeProblemBody(w, status, reason, detail, map[string]any{"trace": trace})
}

func tracedProblemWithFindings(w http.ResponseWriter, status int, reason, detail string, trace []traceStage, findings []verify.Finding) {
	if findings == nil {
		findings = []verify.Finding{}
	}
	writeProblemBody(w, status, reason, detail, map[string]any{"trace": trace, "findings": findings})
}

const statusClientClosedRequest = 499

func writeRequestStop(w http.ResponseWriter, err error, trace []traceStage) bool {
	status, reason, detail, stopped := requestStopped(err)
	if !stopped {
		return false
	}
	tracedProblem(w, status, reason, detail, trace)
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

type providerRetryable interface {
	ProviderRetryable() bool
}

func retryableProvider(err error) bool {
	var retryable providerRetryable
	return errors.As(err, &retryable) && retryable.ProviderRetryable()
}

type responseLimited interface {
	ExceedsResponseBudget() bool
}

func exceedsResponseBudget(err error) bool {
	var limited responseLimited
	return errors.As(err, &limited) && limited.ExceedsResponseBudget()
}

type answerBudgetKey struct{}

func withAnswerBudget(ctx context.Context, n int64) context.Context {
	return context.WithValue(ctx, answerBudgetKey{}, n)
}

// AnswerBudget is the maximum provider body that still fits this response.
func AnswerBudget(ctx context.Context) (int64, bool) {
	n, ok := ctx.Value(answerBudgetKey{}).(int64)
	return n, ok && n > 0
}
