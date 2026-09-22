package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fastygo/lex/internal/evidence"
	"github.com/fastygo/lex/internal/profile/claimvalidation"
	"github.com/fastygo/lex/internal/wire"
)

const genericAnswers = `{
  "intent":{"type":"choice","choice":"portfolio","probabilities":{"products":0.1,"portfolio":0.8,"cases":0.1}},
  "has_transactions":{"type":"noul","noul":0.9},
  "fit":{"type":"score","score":1.5}
}`

const genericDecisionBody = `{
  "project_id":"project-test",
  "decision":{"id":"agent-step-1","version":"1"},
  "state":{"request":"Create a storefront","requirements":{"transactions":true}},
  "question_set":{
    "id":"agent.intent","version":"1",
    "questions":{
      "intent":{"type":"choice","instructions":"Which storefront type is best supported?","options":{"products":"Product catalogue","portfolio":"Portfolio of work","cases":"Client case studies"}},
      "has_transactions":{"type":"noul","instructions":"Does the supplied state require transactions?"},
      "fit":{"type":"score","instructions":"How strongly does the stack fit?","levels":["weak","adequate","strong"]}
    }
  },
  "metadata":{"client_ref":"agent-step-1"}
}`

func TestDecisionPassesCallerStateAndQuestionsWithoutDomainInterpretation(t *testing.T) {
	decider := &capturingDecider{answers: []byte(genericAnswers)}
	handler := mustHandlerWithDecider(t, decider)
	request := httptest.NewRequest(http.MethodPost, "/v1/decisions", strings.NewReader(genericDecisionBody))
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	var response struct {
		StructuralStatus string          `json:"structural_status"`
		Findings         []any           `json:"findings"`
		ReplayBundle     json.RawMessage `json:"replay_bundle"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.StructuralStatus != "valid" || len(response.Findings) != 0 || len(response.ReplayBundle) == 0 {
		t.Fatalf("response = %s", recorder.Body)
	}
	if err := openAPISchema(t, "DecisionResponse").Validate(strictJSON(t, recorder.Body.Bytes())); err != nil {
		t.Fatal(err)
	}
	state, err := json.Marshal(decider.state)
	if err != nil {
		t.Fatal(err)
	}
	questions, err := json.Marshal(decider.questions)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(state), `"Create a storefront"`) || strings.Contains(string(state), "client_ref") {
		t.Fatalf("state = %s", state)
	}
	if !strings.Contains(string(questions), `"portfolio"`) || !strings.Contains(string(questions), `"criteria"`) || strings.Contains(string(questions), "claim-validation") {
		t.Fatalf("questions = %s", questions)
	}
	if strings.Contains(string(response.ReplayBundle), "client_ref") {
		t.Fatalf("metadata entered replay bundle: %s", response.ReplayBundle)
	}
	replay := httptest.NewRequest(http.MethodPost, "/v1/replays", bytes.NewReader(response.ReplayBundle))
	replay.Header.Set("Authorization", "Bearer test-token")
	replay.Header.Set("Content-Type", "application/json")
	replayed := httptest.NewRecorder()
	handler.ServeHTTP(replayed, replay)
	if replayed.Code != http.StatusOK || !strings.Contains(replayed.Body.String(), `"replay_status":"decision_reproduced"`) || !strings.Contains(replayed.Body.String(), `"structural_status":"valid"`) {
		t.Fatalf("replay status = %d body = %s", replayed.Code, replayed.Body)
	}
}

func TestDecisionAcceptsOptionalContextBindingWithoutMergingItIntoState(t *testing.T) {
	frozen, err := evidence.FromSources(context.Background(), "project-test",
		evidence.PackRequest("project-test", "account", claimvalidation.Focus()),
		[]evidence.Source{{ID: "source-1", Version: "v1", Text: "The account is locked."}},
	)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _ := json.Marshal(frozen.Snapshot)
	request, _ := json.Marshal(frozen.PackRequest)
	var state any
	var questions any
	if err := json.Unmarshal([]byte(`{"request":"classify only"}`), &state); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"id":"generic","version":"1","questions":{"yes":{"type":"noul","instructions":"Is the state classified?"}}}`), &questions); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"project_id":   "project-test",
		"decision":     map[string]any{"id": "agent-step-2", "version": "1"},
		"state":        state,
		"question_set": questions,
		"context": map[string]json.RawMessage{
			"pack": frozen.Pack, "snapshot": snapshot, "pack_request": request,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wire.ValidateDecisionRequest(body); err != nil {
		t.Fatal(err)
	}
	decider := &capturingDecider{answers: []byte(`{"yes":{"type":"noul","noul":0.9}}`)}
	handler := mustHandlerWithDecider(t, decider)
	httpRequest := httptest.NewRequest(http.MethodPost, "/v1/decisions", bytes.NewReader(body))
	httpRequest.Header.Set("Authorization", "Bearer test-token")
	httpRequest.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httpRequest)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	encoded, _ := json.Marshal(decider.state)
	if string(encoded) != `{"request":"classify only"}` {
		t.Fatalf("context was merged into state: %s", encoded)
	}
}

func TestDecisionRejectsMalformedQuestionSetBeforeProvider(t *testing.T) {
	decider := &scriptedDecider{answers: []byte(genericAnswers)}
	handler := mustHandlerWithDecider(t, decider)
	body := strings.Replace(genericDecisionBody, `"levels":["weak","adequate","strong"]`, `"levels":["weak"]`, 1)
	request := httptest.NewRequest(http.MethodPost, "/v1/decisions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity || decider.calls != 0 || !strings.Contains(recorder.Body.String(), `"reason":"question_error"`) || !strings.Contains(recorder.Body.String(), "/question_set/questions/fit") {
		t.Fatalf("status = %d calls = %d body = %s", recorder.Code, decider.calls, recorder.Body)
	}
}

func TestDecisionSchemaErrorsNameTheFieldAndKeepInvalidJSONSeparate(t *testing.T) {
	decider := &scriptedDecider{answers: []byte(genericAnswers)}
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
