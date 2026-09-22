package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/wire"
)

func TestDecisionPassesCallerStateAndQuestionsWithoutDomainInterpretation(t *testing.T) {
	decider := &capturingDecider{answers: []byte(validAnswers)}
	handler := mustHandlerWithDecider(t, decider)
	recorder := postJSON(t, handler, "/v1/decisions", decisionBody)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"structural_status":"valid"`) || !strings.Contains(recorder.Body.String(), `"findings":[]`) {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	state, _ := json.Marshal(decider.state)
	questions, _ := json.Marshal(decider.questions)
	if string(state) != `{"request":"Create a storefront","requirements":{"transactions":true}}` {
		t.Fatalf("state = %s", state)
	}
	if !strings.Contains(string(questions), `"portfolio"`) || !strings.Contains(string(questions), `"criteria"`) || strings.Contains(string(questions), "client_ref") {
		t.Fatalf("questions = %s", questions)
	}
	bundle := replayBundle(t, recorder)
	if strings.Contains(string(bundle), "client_ref") || strings.Contains(recorder.Body.String(), "agent-step-1\"}") {
		t.Fatalf("metadata entered the response or bundle: %s", recorder.Body)
	}
	replayed := postJSON(t, handler, "/v1/replays", string(bundle))
	if replayed.Code != http.StatusOK || !strings.Contains(replayed.Body.String(), `"replay_status":"decision_reproduced"`) || !strings.Contains(replayed.Body.String(), `"structural_status":"valid"`) {
		t.Fatalf("replay status = %d body = %s", replayed.Code, replayed.Body)
	}
}

func frozenContextBody(t *testing.T, projectID string, mutate func(map[string]json.RawMessage)) string {
	t.Helper()
	runtime, err := contextmemory.New(context.Background(), contextmemory.Config{
		ProjectID: projectID,
		Sources: []contextmemory.Source{{
			SourceID: "source-1", Version: "v1", Text: "The account is locked.",
			TrustLevel: "project", EvidenceClass: "source_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := contextmemory.PackRequest{
		ProjectID: projectID, Query: "account",
		Focus: contextmemory.Focus{
			ID: "example-focus", Objective: "Select source text.", RequiredTrustLevel: "project",
			Budget: contextmemory.Budget{MaxItems: 8, MaxChars: 65536},
		},
	}
	result, err := runtime.ContextPack(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _ := json.Marshal(result.Snapshot)
	packRequest, _ := json.Marshal(request)
	frozen := map[string]json.RawMessage{"pack": result.ContextPack, "snapshot": snapshot, "pack_request": packRequest}
	if mutate != nil {
		mutate(frozen)
	}
	body, err := json.Marshal(map[string]any{
		"project_id":   "project-test",
		"decision":     map[string]any{"id": "agent-step-2", "version": "1"},
		"state":        map[string]any{"request": "classify only"},
		"question_set": map[string]any{"id": "example", "version": "1", "questions": map[string]any{"yes": map[string]any{"type": "noul", "instructions": "Is the state classified?"}}},
		"context":      frozen,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wire.ValidateDecisionRequest(body); err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestDecisionBindsOptionalContextWithoutMergingItIntoState(t *testing.T) {
	decider := &capturingDecider{answers: []byte(`{"yes":{"type":"noul","noul":0.9}}`)}
	handler := mustHandlerWithDecider(t, decider)
	recorder := postJSON(t, handler, "/v1/decisions", frozenContextBody(t, "project-test", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	if encoded, _ := json.Marshal(decider.state); string(encoded) != `{"request":"classify only"}` {
		t.Fatalf("context was merged into state: %s", encoded)
	}
	bundle := replayBundle(t, recorder)
	if !strings.Contains(string(bundle), `"context_pack_hash"`) {
		t.Fatalf("bundle does not bind the context: %s", bundle)
	}
	if replayed := postJSON(t, handler, "/v1/replays", string(bundle)); replayed.Code != http.StatusOK || !strings.Contains(replayed.Body.String(), `"structural_status":"valid"`) {
		t.Fatalf("replay status = %d body = %s", replayed.Code, replayed.Body)
	}
}

func TestContextThatContextCannotReproduceNeverReachesTheProvider(t *testing.T) {
	cases := map[string]struct {
		body    string
		finding string
	}{
		"rewritten pack request": {frozenContextBody(t, "project-test", func(frozen map[string]json.RawMessage) {
			frozen["pack_request"] = json.RawMessage(strings.Replace(string(frozen["pack_request"]), `"query":"account"`, `"query":"locked"`, 1))
		}), "pack_rebuild"},
		"rewritten source": {frozenContextBody(t, "project-test", func(frozen map[string]json.RawMessage) {
			frozen["snapshot"] = json.RawMessage(strings.Replace(string(frozen["snapshot"]), "locked", "open", 1))
		}), "snapshot_identity"},
		"foreign project": {frozenContextBody(t, "project-other", nil), "project_binding"},
		"smuggled field": {frozenContextBody(t, "project-test", func(frozen map[string]json.RawMessage) {
			frozen["pack_request"] = json.RawMessage(strings.Replace(string(frozen["pack_request"]), `"query"`, `"approved":true,"query"`, 1))
		}), "pack_shape"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			decider := &scriptedDecider{answers: []byte(`{"yes":{"type":"noul","noul":0.9}}`)}
			recorder := postJSON(t, mustHandlerWithDecider(t, decider), "/v1/decisions", tc.body)
			if recorder.Code != http.StatusUnprocessableEntity || decider.calls != 0 || !strings.Contains(recorder.Body.String(), `"reason":"pack_error"`) || !strings.Contains(recorder.Body.String(), `"code":"`+tc.finding+`"`) {
				t.Fatalf("status = %d calls = %d body = %s", recorder.Code, decider.calls, recorder.Body)
			}
		})
	}
}

func TestDecisionRejectsMalformedQuestionSetBeforeProvider(t *testing.T) {
	decider := &scriptedDecider{answers: []byte(validAnswers)}
	body := strings.Replace(decisionBody, `"levels":["weak","adequate","strong"]`, `"levels":["weak"]`, 1)
	recorder := postJSON(t, mustHandlerWithDecider(t, decider), "/v1/decisions", body)
	if recorder.Code != http.StatusUnprocessableEntity || decider.calls != 0 || !strings.Contains(recorder.Body.String(), `"reason":"question_error"`) || !strings.Contains(recorder.Body.String(), "/question_set/questions/fit") {
		t.Fatalf("status = %d calls = %d body = %s", recorder.Code, decider.calls, recorder.Body)
	}
}

func TestDecisionSchemaErrorsNameTheFieldAndKeepInvalidJSONSeparate(t *testing.T) {
	decider := &scriptedDecider{answers: []byte(validAnswers)}
	handler := mustHandlerWithDecider(t, decider)
	rawJev := `{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":{"text":"hi"},"question_set":{"id":"set","version":"1","questions":{"route":{"type":"choice","instructions":"Which route?","criteria":{"a":"A","b":"B"}}}}}`
	cases := []struct {
		name   string
		body   string
		status int
		want   []string
	}{
		{"raw jev criteria", rawJev, http.StatusUnprocessableEntity, []string{`"reason":"question_error"`, "/question_set/questions/route", "options"}},
		{"missing decision", `{"project_id":"project-test","state":{},"question_set":{"id":"set","version":"1","questions":{"q":{"type":"noul","instructions":"x"}}}}`, http.StatusUnprocessableEntity, []string{`"reason":"question_error"`, "required"}},
		{"broken json", `{"project_id":`, http.StatusBadRequest, []string{`"reason":"invalid_json"`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := postJSON(t, handler, "/v1/decisions", tc.body)
			if recorder.Code != tc.status || decider.calls != 0 {
				t.Fatalf("status = %d calls = %d body = %s", recorder.Code, decider.calls, recorder.Body)
			}
			for _, want := range tc.want {
				if !strings.Contains(recorder.Body.String(), want) {
					t.Fatalf("body lacks %q: %s", want, recorder.Body)
				}
			}
			if strings.Contains(recorder.Body.String(), "Which route?") {
				t.Fatalf("problem echoed caller text: %s", recorder.Body)
			}
		})
	}
}
